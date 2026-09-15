package main

import (
	"database/sql"
	"errors"
	"time"
)

// ErrNotFound is returned when the requested row doesn't exist, or - for
// boards - when it exists but the caller isn't a member (boards are
// private, see board_members; we don't distinguish "doesn't exist" from
// "not yours" in the response, same reasoning as a 404 instead of a 403
// for a resource you shouldn't know exists).
var ErrNotFound = errors.New("not found")

// ErrForbidden is for actions that require a stronger relationship than
// plain membership (e.g. only the owner can delete a board or manage its
// members).
var ErrForbidden = errors.New("forbidden")

// Store is backed by Postgres, scoped to this module's schema via the
// connection's search_path (see db.go).
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ---- membership ----

func (s *Store) memberRole(boardID, userID string) (string, error) {
	var role string
	err := s.db.QueryRow(`SELECT role FROM board_members WHERE board_id = $1 AND user_id = $2`, boardID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}

func (s *Store) requireMember(boardID, userID string) error {
	_, err := s.memberRole(boardID, userID)
	return err
}

func (s *Store) requireOwner(boardID, userID string) error {
	role, err := s.memberRole(boardID, userID)
	if err != nil {
		return err
	}
	if role != "owner" {
		return ErrForbidden
	}
	return nil
}

// boardIDForColumn/boardIDForCard resolve the board a column/card belongs
// to, so mutation endpoints can check membership even though the request
// only carries a column/card id.
func (s *Store) boardIDForColumn(columnID string) (string, error) {
	var boardID string
	err := s.db.QueryRow(`SELECT board_id FROM columns WHERE id = $1`, columnID).Scan(&boardID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return boardID, err
}

func (s *Store) boardIDForCard(cardID string) (string, error) {
	var boardID string
	err := s.db.QueryRow(`
		SELECT c.board_id FROM cards ca JOIN columns c ON c.id = ca.column_id WHERE ca.id = $1`, cardID).Scan(&boardID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return boardID, err
}

// ---- boards ----

func (s *Store) ListBoards(userID string) ([]*Board, error) {
	rows, err := s.db.Query(`
		SELECT b.id, b.owner_id, b.name, b.description, b.created_at, b.updated_at
		FROM boards b
		JOIN board_members bm ON bm.board_id = b.id
		WHERE bm.user_id = $1
		ORDER BY b.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	boards := []*Board{}
	for rows.Next() {
		var b Board
		if err := rows.Scan(&b.ID, &b.OwnerID, &b.Name, &b.Description, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		boards = append(boards, &b)
	}
	return boards, rows.Err()
}

func (s *Store) getBoard(id string) (*Board, error) {
	var b Board
	err := s.db.QueryRow(`SELECT id, owner_id, name, description, created_at, updated_at FROM boards WHERE id = $1`, id).
		Scan(&b.ID, &b.OwnerID, &b.Name, &b.Description, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &b, err
}

// CreateBoardWithColumns creates a board, makes the creator its owner, and
// creates the given initial columns (the workflow) in one go.
func (s *Store) CreateBoardWithColumns(userID, name, description string, columnNames []string) (*Board, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var b Board
	b.OwnerID = userID
	if err := tx.QueryRow(
		`INSERT INTO boards (owner_id, name, description) VALUES ($1, $2, $3) RETURNING id, name, description, created_at, updated_at`,
		userID, name, description,
	).Scan(&b.ID, &b.Name, &b.Description, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(`INSERT INTO board_members (board_id, user_id, role) VALUES ($1, $2, 'owner')`, b.ID, userID); err != nil {
		return nil, err
	}

	for i, name := range columnNames {
		if name == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO columns (board_id, name, position) VALUES ($1, $2, $3)`, b.ID, name, i); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) UpdateBoard(userID, id string, in *Board) (*Board, error) {
	if err := s.requireMember(id, userID); err != nil {
		return nil, err
	}
	res, err := s.db.Exec(
		`UPDATE boards SET name = $1, description = $2, updated_at = now() WHERE id = $3`,
		in.Name, in.Description, id,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.getBoard(id)
}

// DeleteBoard is owner-only.
func (s *Store) DeleteBoard(userID, id string) error {
	if err := s.requireOwner(id, userID); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM boards WHERE id = $1`, id)
	return err
}

// GetBoardFull returns a board together with all of its columns and each
// column's cards, so the frontend can render the whole board in one call.
// Returns ErrNotFound if userID isn't a member.
func (s *Store) GetBoardFull(userID, id string) (*BoardFull, error) {
	if err := s.requireMember(id, userID); err != nil {
		return nil, err
	}
	board, err := s.getBoard(id)
	if err != nil {
		return nil, err
	}

	colRows, err := s.db.Query(`SELECT id, board_id, name, position, created_at, updated_at FROM columns WHERE board_id = $1 ORDER BY position, created_at`, id)
	if err != nil {
		return nil, err
	}
	defer colRows.Close()

	full := &BoardFull{Board: *board, Columns: []*ColumnWithCards{}}
	for colRows.Next() {
		var c Column
		if err := colRows.Scan(&c.ID, &c.BoardID, &c.Name, &c.Position, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		full.Columns = append(full.Columns, &ColumnWithCards{Column: c, Cards: []*Card{}})
	}
	if err := colRows.Err(); err != nil {
		return nil, err
	}

	for _, col := range full.Columns {
		cards, err := s.listCardsByColumn(col.ID)
		if err != nil {
			return nil, err
		}
		col.Cards = cards
	}

	return full, nil
}

// ---- members ----

func (s *Store) ListMembers(boardID string) ([]*BoardMember, error) {
	rows, err := s.db.Query(`
		SELECT bm.user_id, u.username, u.full_name, bm.role, bm.joined_at
		FROM board_members bm
		JOIN app_maintenance.users u ON u.id = bm.user_id
		WHERE bm.board_id = $1
		ORDER BY bm.joined_at`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []*BoardMember{}
	for rows.Next() {
		var m BoardMember
		if err := rows.Scan(&m.UserID, &m.Username, &m.FullName, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, &m)
	}
	return members, rows.Err()
}

// AddMember is owner-only; looks the invitee up by username in
// app-maintenance's user directory (cross-schema read).
func (s *Store) AddMember(actorUserID, boardID, username string) (*BoardMember, error) {
	if err := s.requireOwner(boardID, actorUserID); err != nil {
		return nil, err
	}

	var m BoardMember
	err := s.db.QueryRow(`SELECT id, username, full_name FROM app_maintenance.users WHERE username = $1`, username).
		Scan(&m.UserID, &m.Username, &m.FullName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	m.Role = "member"
	if _, err := s.db.Exec(
		`INSERT INTO board_members (board_id, user_id, role) VALUES ($1, $2, 'member') ON CONFLICT DO NOTHING`,
		boardID, m.UserID,
	); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(`SELECT joined_at FROM board_members WHERE board_id = $1 AND user_id = $2`, boardID, m.UserID).Scan(&m.JoinedAt); err != nil {
		return nil, err
	}
	return &m, nil
}

// RemoveMember is owner-only; an owner can't remove themself (transfer
// ownership isn't implemented yet - delete the board instead).
func (s *Store) RemoveMember(actorUserID, boardID, targetUserID string) error {
	if err := s.requireOwner(boardID, actorUserID); err != nil {
		return err
	}
	if targetUserID == actorUserID {
		return ErrForbidden
	}
	res, err := s.db.Exec(`DELETE FROM board_members WHERE board_id = $1 AND user_id = $2`, boardID, targetUserID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListAllUsers is a cross-schema read of app-maintenance's user directory,
// used to populate the "add member" search. Read-only, no FK - see the
// comment on board_members in migrate.go.
func (s *Store) ListAllUsers() ([]*UserRef, error) {
	rows, err := s.db.Query(`SELECT id, username, full_name FROM app_maintenance.users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*UserRef{}
	for rows.Next() {
		var u UserRef
		if err := rows.Scan(&u.ID, &u.Username, &u.FullName); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

// ---- columns ----

func (s *Store) CreateColumn(userID string, in *Column) (*Column, error) {
	if err := s.requireMember(in.BoardID, userID); err != nil {
		return nil, err
	}
	err := s.db.QueryRow(
		`INSERT INTO columns (board_id, name, position) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		in.BoardID, in.Name, in.Position,
	).Scan(&in.ID, &in.CreatedAt, &in.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return in, nil
}

func (s *Store) UpdateColumn(userID, id string, in *Column) (*Column, error) {
	boardID, err := s.boardIDForColumn(id)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(boardID, userID); err != nil {
		return nil, err
	}
	res, err := s.db.Exec(
		`UPDATE columns SET name = $1, position = $2, updated_at = now() WHERE id = $3`,
		in.Name, in.Position, id,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	var c Column
	err = s.db.QueryRow(`SELECT id, board_id, name, position, created_at, updated_at FROM columns WHERE id = $1`, id).
		Scan(&c.ID, &c.BoardID, &c.Name, &c.Position, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) DeleteColumn(userID, id string) error {
	boardID, err := s.boardIDForColumn(id)
	if err != nil {
		return err
	}
	if err := s.requireMember(boardID, userID); err != nil {
		return err
	}
	res, err := s.db.Exec(`DELETE FROM columns WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- cards ----

const cardColumns = `id, column_id, title, description, position, assignee_id, due_date, color, created_at, updated_at`

// scanCard reads one cards row, handling assignee_id/due_date's nullability
// (plain sql.Scan into *string/*time.Time fails on NULL, so these go
// through sql.NullString/sql.NullTime first).
func scanCard(row interface{ Scan(...any) error }) (*Card, error) {
	var c Card
	var assigneeID sql.NullString
	var dueDate sql.NullTime
	if err := row.Scan(&c.ID, &c.ColumnID, &c.Title, &c.Description, &c.Position, &assigneeID, &dueDate, &c.Color, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	if assigneeID.Valid {
		c.AssigneeID = &assigneeID.String
	}
	if dueDate.Valid {
		c.DueDate = &dueDate.Time
	}
	return &c, nil
}

// nullableStr/nullableTime treat a nil pointer as SQL NULL - used when
// writing AssigneeID/DueDate below.
func nullableStr(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}
func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func (s *Store) listCardsByColumn(columnID string) ([]*Card, error) {
	rows, err := s.db.Query(`SELECT `+cardColumns+` FROM cards WHERE column_id = $1 ORDER BY position, created_at`, columnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cards := []*Card{}
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

func (s *Store) CreateCard(userID string, in *Card) (*Card, error) {
	boardID, err := s.boardIDForColumn(in.ColumnID)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(boardID, userID); err != nil {
		return nil, err
	}
	row := s.db.QueryRow(
		`INSERT INTO cards (column_id, title, description, position, assignee_id, due_date, color)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+cardColumns,
		in.ColumnID, in.Title, in.Description, in.Position, nullableStr(in.AssigneeID), nullableTime(in.DueDate), in.Color,
	)
	return scanCard(row)
}

func (s *Store) UpdateCard(userID, id string, in *Card) (*Card, error) {
	boardID, err := s.boardIDForCard(id)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(boardID, userID); err != nil {
		return nil, err
	}
	res, err := s.db.Exec(
		`UPDATE cards SET title = $1, description = $2, assignee_id = $3, due_date = $4, color = $5, updated_at = now() WHERE id = $6`,
		in.Title, in.Description, nullableStr(in.AssigneeID), nullableTime(in.DueDate), in.Color, id,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.getCard(id)
}

// MoveCard moves a card to another column (or reorders it within the same
// column) by setting its column_id and position. Both the source and
// destination columns must belong to a board userID is a member of.
func (s *Store) MoveCard(userID, id string, columnID string, position int) (*Card, error) {
	currentBoardID, err := s.boardIDForCard(id)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(currentBoardID, userID); err != nil {
		return nil, err
	}
	destBoardID, err := s.boardIDForColumn(columnID)
	if err != nil {
		return nil, err
	}
	if destBoardID != currentBoardID {
		return nil, ErrForbidden
	}

	res, err := s.db.Exec(
		`UPDATE cards SET column_id = $1, position = $2, updated_at = now() WHERE id = $3`,
		columnID, position, id,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.getCard(id)
}

func (s *Store) getCard(id string) (*Card, error) {
	row := s.db.QueryRow(`SELECT `+cardColumns+` FROM cards WHERE id = $1`, id)
	c, err := scanCard(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) DeleteCard(userID, id string) error {
	boardID, err := s.boardIDForCard(id)
	if err != nil {
		return err
	}
	if err := s.requireMember(boardID, userID); err != nil {
		return err
	}
	res, err := s.db.Exec(`DELETE FROM cards WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
