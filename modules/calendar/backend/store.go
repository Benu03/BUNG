package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is returned when an event doesn't exist, or exists but
// belongs to someone else - every query here is scoped to owner_id, so
// "not yours" and "doesn't exist" look the same on purpose.
var ErrNotFound = errors.New("not found")

// Store is backed by Postgres, scoped to this module's schema via the
// connection's search_path (see db.go).
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func scanEvent(row interface{ Scan(...any) error }) (*Event, error) {
	var e Event
	if err := row.Scan(&e.ID, &e.OwnerID, &e.Title, &e.Description, &e.Location, &e.StartAt, &e.EndAt, &e.AllDay, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}
	return &e, nil
}

// List returns ownerID's events ordered by start time. When from/to are
// non-nil, only events overlapping that range are returned (start_at <= to
// AND end_at >= from) - handy for a month/week view; pass both nil to get
// everything.
func (s *Store) List(ownerID string, from, to *time.Time) ([]*Event, error) {
	query := `SELECT id, owner_id, title, description, location, start_at, end_at, all_day, created_at, updated_at
	          FROM events WHERE owner_id = $1`
	args := []any{ownerID}

	if from != nil {
		args = append(args, *from)
		query += fmt.Sprintf(" AND end_at >= $%d", len(args))
	}
	if to != nil {
		args = append(args, *to)
		query += fmt.Sprintf(" AND start_at <= $%d", len(args))
	}
	query += " ORDER BY start_at"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []*Event{}
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) Get(ownerID, id string) (*Event, error) {
	row := s.db.QueryRow(
		`SELECT id, owner_id, title, description, location, start_at, end_at, all_day, created_at, updated_at
		 FROM events WHERE id = $1 AND owner_id = $2`, id, ownerID,
	)
	e, err := scanEvent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (s *Store) Create(ownerID string, in *eventRequest) (*Event, error) {
	e := &Event{
		OwnerID:     ownerID,
		Title:       in.Title,
		Description: in.Description,
		Location:    in.Location,
		StartAt:     in.StartAt,
		EndAt:       in.EndAt,
		AllDay:      in.AllDay,
	}
	err := s.db.QueryRow(
		`INSERT INTO events (owner_id, title, description, location, start_at, end_at, all_day)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at`,
		e.OwnerID, e.Title, e.Description, e.Location, e.StartAt, e.EndAt, e.AllDay,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (s *Store) Update(ownerID, id string, in *eventRequest) (*Event, error) {
	e := &Event{ID: id, OwnerID: ownerID}
	err := s.db.QueryRow(
		`UPDATE events SET title = $1, description = $2, location = $3, start_at = $4, end_at = $5, all_day = $6, updated_at = now()
		 WHERE id = $7 AND owner_id = $8
		 RETURNING title, description, location, start_at, end_at, all_day, created_at, updated_at`,
		in.Title, in.Description, in.Location, in.StartAt, in.EndAt, in.AllDay, id, ownerID,
	).Scan(&e.Title, &e.Description, &e.Location, &e.StartAt, &e.EndAt, &e.AllDay, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (s *Store) Delete(ownerID, id string) error {
	res, err := s.db.Exec(`DELETE FROM events WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
