package main

import (
	"database/sql"
	"errors"
)

// ErrNotFound is returned by Store methods when the requested row doesn't
// exist, so handlers can tell that apart from a real (5xx) failure.
var ErrNotFound = errors.New("not found")

// Store is backed by Postgres, scoped to this module's schema via the
// connection's search_path (see db.go).
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ---- boards ----

func (s *Store) ListBoards() ([]*Board, error) {
	rows, err := s.db.Query(`SELECT id, name, description, created_at, updated_at FROM boards ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var boards []*Board
	for rows.Next() {
		var b Board
		if err := rows.Scan(&b.ID, &b.Name, &b.Description, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		boards = append(boards, &b)
	}
	return boards, rows.Err()
}

func (s *Store) GetBoard(id string) (*Board, error) {
	var b Board
	err := s.db.QueryRow(`SELECT id, name, description, created_at, updated_at FROM boards WHERE id = $1`, id).
		Scan(&b.ID, &b.Name, &b.Description, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) CreateBoard(in *Board) (*Board, error) {
	err := s.db.QueryRow(
		`INSERT INTO boards (name, description) VALUES ($1, $2) RETURNING id, created_at, updated_at`,
		in.Name, in.Description,
	).Scan(&in.ID, &in.CreatedAt, &in.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return in, nil
}

func (s *Store) UpdateBoard(id string, in *Board) (*Board, error) {
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
	return s.GetBoard(id)
}

func (s *Store) DeleteBoard(id string) error {
	res, err := s.db.Exec(`DELETE FROM boards WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetBoardFull returns a board together with all of its columns and each
// column's cards, so the frontend can render the whole board in one call.
func (s *Store) GetBoardFull(id string) (*BoardFull, error) {
	board, err := s.GetBoard(id)
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

// ---- columns ----

func (s *Store) CreateColumn(in *Column) (*Column, error) {
	err := s.db.QueryRow(
		`INSERT INTO columns (board_id, name, position) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		in.BoardID, in.Name, in.Position,
	).Scan(&in.ID, &in.CreatedAt, &in.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return in, nil
}

func (s *Store) UpdateColumn(id string, in *Column) (*Column, error) {
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

func (s *Store) DeleteColumn(id string) error {
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

func (s *Store) listCardsByColumn(columnID string) ([]*Card, error) {
	rows, err := s.db.Query(`SELECT id, column_id, title, description, position, created_at, updated_at FROM cards WHERE column_id = $1 ORDER BY position, created_at`, columnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cards := []*Card{}
	for rows.Next() {
		var c Card
		if err := rows.Scan(&c.ID, &c.ColumnID, &c.Title, &c.Description, &c.Position, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cards = append(cards, &c)
	}
	return cards, rows.Err()
}

func (s *Store) CreateCard(in *Card) (*Card, error) {
	err := s.db.QueryRow(
		`INSERT INTO cards (column_id, title, description, position) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		in.ColumnID, in.Title, in.Description, in.Position,
	).Scan(&in.ID, &in.CreatedAt, &in.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return in, nil
}

func (s *Store) UpdateCard(id string, in *Card) (*Card, error) {
	res, err := s.db.Exec(
		`UPDATE cards SET title = $1, description = $2, updated_at = now() WHERE id = $3`,
		in.Title, in.Description, id,
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
// column) by setting its column_id and position.
func (s *Store) MoveCard(id string, columnID string, position int) (*Card, error) {
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
	var c Card
	err := s.db.QueryRow(`SELECT id, column_id, title, description, position, created_at, updated_at FROM cards WHERE id = $1`, id).
		Scan(&c.ID, &c.ColumnID, &c.Title, &c.Description, &c.Position, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) DeleteCard(id string) error {
	res, err := s.db.Exec(`DELETE FROM cards WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
