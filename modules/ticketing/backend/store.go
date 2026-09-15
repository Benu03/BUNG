package main

import (
	"database/sql"
	"errors"
)

// ErrNotFound is returned when a ticket/comment doesn't exist.
var ErrNotFound = errors.New("not found")

// Store is backed by Postgres, scoped to this module's schema via the
// connection's search_path (see db.go).
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func nullableID(id *string) any {
	if id == nil || *id == "" {
		return nil
	}
	return *id
}

func scanTicket(row interface{ Scan(...any) error }) (*Ticket, error) {
	var t Ticket
	var assigneeID sql.NullString
	var resolvedAt sql.NullTime
	if err := row.Scan(
		&t.ID, &t.RequesterID, &assigneeID, &t.Title, &t.Description,
		&t.Status, &t.Priority, &t.Category, &t.CreatedAt, &t.UpdatedAt, &resolvedAt,
	); err != nil {
		return nil, err
	}
	if assigneeID.Valid {
		t.AssigneeID = &assigneeID.String
	}
	if resolvedAt.Valid {
		t.ResolvedAt = &resolvedAt.Time
	}
	return &t, nil
}

const ticketColumns = `id, requester_id, assignee_id, title, description, status, priority, category, created_at, updated_at, resolved_at`

// ---- tickets ----

// List returns every ticket (this is a shared queue, not scoped to the
// caller - see the Ticket struct's comment in models.go), most recently
// created first.
func (s *Store) List() ([]*Ticket, error) {
	rows, err := s.db.Query(`SELECT ` + ticketColumns + ` FROM tickets ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tickets := []*Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (s *Store) Get(id string) (*Ticket, error) {
	row := s.db.QueryRow(`SELECT `+ticketColumns+` FROM tickets WHERE id = $1`, id)
	t, err := scanTicket(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *Store) Create(requesterID string, in *createTicketRequest) (*Ticket, error) {
	row := s.db.QueryRow(
		`INSERT INTO tickets (requester_id, title, description, category, priority)
		 VALUES ($1, $2, $3, $4, $5) RETURNING `+ticketColumns,
		requesterID, in.Title, in.Description, in.Category, in.Priority,
	)
	return scanTicket(row)
}

// Update applies every editable field at once. resolved_at is managed
// here: it's set the moment status becomes resolved/closed (if not
// already), and cleared if the ticket is reopened afterwards.
func (s *Store) Update(id string, in *updateTicketRequest) (*Ticket, error) {
	row := s.db.QueryRow(
		`UPDATE tickets SET
			title = $1, description = $2, category = $3, priority = $4, status = $5,
			assignee_id = $6, updated_at = now(),
			resolved_at = CASE
				WHEN $5 IN ('resolved', 'closed') AND resolved_at IS NULL THEN now()
				WHEN $5 NOT IN ('resolved', 'closed') THEN NULL
				ELSE resolved_at
			END
		 WHERE id = $7
		 RETURNING `+ticketColumns,
		in.Title, in.Description, in.Category, in.Priority, in.Status, nullableID(in.AssigneeID), id,
	)
	t, err := scanTicket(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *Store) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM tickets WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- comments ----

func (s *Store) ListComments(ticketID string) ([]*Comment, error) {
	rows, err := s.db.Query(
		`SELECT id, ticket_id, author_id, body, created_at FROM ticket_comments WHERE ticket_id = $1 ORDER BY created_at`,
		ticketID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []*Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, &c)
	}
	return comments, rows.Err()
}

func (s *Store) CreateComment(ticketID, authorID, body string) (*Comment, error) {
	c := &Comment{TicketID: ticketID, AuthorID: authorID, Body: body}
	err := s.db.QueryRow(
		`INSERT INTO ticket_comments (ticket_id, author_id, body) VALUES ($1, $2, $3) RETURNING id, created_at`,
		ticketID, authorID, body,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ---- attachments ----

func (s *Store) ListAttachments(ticketID string) ([]*Attachment, error) {
	rows, err := s.db.Query(
		`SELECT id, ticket_id, uploader_id, filename, content_type, size, storage_path, created_at
		 FROM ticket_attachments WHERE ticket_id = $1 ORDER BY created_at`,
		ticketID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attachments := []*Attachment{}
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.TicketID, &a.UploaderID, &a.Filename, &a.ContentType, &a.Size, &a.StoragePath, &a.CreatedAt); err != nil {
			return nil, err
		}
		attachments = append(attachments, &a)
	}
	return attachments, rows.Err()
}

func (s *Store) GetAttachment(id string) (*Attachment, error) {
	var a Attachment
	err := s.db.QueryRow(
		`SELECT id, ticket_id, uploader_id, filename, content_type, size, storage_path, created_at
		 FROM ticket_attachments WHERE id = $1`, id,
	).Scan(&a.ID, &a.TicketID, &a.UploaderID, &a.Filename, &a.ContentType, &a.Size, &a.StoragePath, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// CreateAttachment reserves a row (storage_path filled with a placeholder
// until the blob is actually written - see UpdateAttachmentStorage), same
// two-step pattern as my-storage's CreateFile.
func (s *Store) CreateAttachment(a *Attachment) (*Attachment, error) {
	err := s.db.QueryRow(
		`INSERT INTO ticket_attachments (ticket_id, uploader_id, filename, content_type, size, storage_path)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`,
		a.TicketID, a.UploaderID, a.Filename, a.ContentType, a.Size, a.StoragePath,
	).Scan(&a.ID, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Store) UpdateAttachmentStorage(id, storagePath string, size int64) error {
	_, err := s.db.Exec(`UPDATE ticket_attachments SET storage_path = $1, size = $2 WHERE id = $3`, storagePath, size, id)
	return err
}

// DeleteAttachment deletes the metadata row and returns it (so the caller
// can remove the underlying blob) - or ErrNotFound if it doesn't exist.
func (s *Store) DeleteAttachment(id string) (*Attachment, error) {
	a, err := s.GetAttachment(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(`DELETE FROM ticket_attachments WHERE id = $1`, id); err != nil {
		return nil, err
	}
	return a, nil
}

// ---- users (cross-schema) ----

// ListAllUsers is a cross-schema read of app-maintenance's user directory,
// used for the requester/assignee display and the assignee picker.
// Read-only, no FK - see the comment on tickets in migrate.go. Same
// pattern as kanban's ListAllUsers.
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
