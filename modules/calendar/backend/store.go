package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is returned when an event doesn't exist, or exists but the
// caller can't see it (not the owner, not an invited attendee) - "not
// yours" and "doesn't exist" look the same on purpose.
var ErrNotFound = errors.New("not found")

// ErrForbidden is for actions that require ownership specifically (invite/
// remove attendees, edit, delete) - an attendee can see an event but not
// change it, same asymmetry as kanban's board owner vs member.
var ErrForbidden = errors.New("forbidden")

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

// List returns every event userID can see - their own, plus anything
// they've been invited to (see event_attendees) - ordered by start time.
// When from/to are non-nil, only events overlapping that range are
// returned (start_at <= to AND end_at >= from) - handy for a month/week
// view; pass both nil to get everything.
func (s *Store) List(userID string, from, to *time.Time) ([]*Event, error) {
	query := `SELECT DISTINCT e.id, e.owner_id, e.title, e.description, e.location, e.start_at, e.end_at, e.all_day, e.created_at, e.updated_at
	          FROM events e
	          LEFT JOIN event_attendees ea ON ea.event_id = e.id AND ea.user_id = $1
	          WHERE (e.owner_id = $1 OR ea.user_id = $1)`
	args := []any{userID}

	if from != nil {
		args = append(args, *from)
		query += fmt.Sprintf(" AND e.end_at >= $%d", len(args))
	}
	if to != nil {
		args = append(args, *to)
		query += fmt.Sprintf(" AND e.start_at <= $%d", len(args))
	}
	query += " ORDER BY e.start_at"

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

// Get returns eventID if userID can see it - owner or invited attendee.
func (s *Store) Get(userID, id string) (*Event, error) {
	row := s.db.QueryRow(
		`SELECT e.id, e.owner_id, e.title, e.description, e.location, e.start_at, e.end_at, e.all_day, e.created_at, e.updated_at
		 FROM events e
		 LEFT JOIN event_attendees ea ON ea.event_id = e.id AND ea.user_id = $2
		 WHERE e.id = $1 AND (e.owner_id = $2 OR ea.user_id = $2)`, id, userID,
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

// ---- attendees ----

// ListAttendees is a cross-schema read (event_attendees joined with
// app-maintenance's user directory) so callers get a display-ready
// username/fullName instead of a bare user id.
func (s *Store) ListAttendees(eventID string) ([]*AttendeeRef, error) {
	rows, err := s.db.Query(`
		SELECT u.id, u.username, u.full_name
		FROM event_attendees ea
		JOIN app_maintenance.users u ON u.id = ea.user_id
		WHERE ea.event_id = $1
		ORDER BY u.username`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attendees := []*AttendeeRef{}
	for rows.Next() {
		var a AttendeeRef
		if err := rows.Scan(&a.UserID, &a.Username, &a.FullName); err != nil {
			return nil, err
		}
		attendees = append(attendees, &a)
	}
	return attendees, rows.Err()
}

// eventOwner fetches just the owner_id, used by Invite/RemoveAttendee to
// check ownership without a full Get (which would also require the caller
// to already have access, which they might not yet when inviting no one
// has added them - not actually a concern here since only the owner
// calls this, but the direct query is simpler than reusing Get anyway).
func (s *Store) eventOwner(eventID string) (string, error) {
	var ownerID string
	err := s.db.QueryRow(`SELECT owner_id FROM events WHERE id = $1`, eventID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return ownerID, err
}

// Invite adds username as an attendee of eventID - owner-only (returns
// ErrForbidden otherwise). ErrNotFound covers both "no such event" and "no
// such username".
func (s *Store) Invite(ownerID, eventID, username string) (*AttendeeRef, error) {
	actualOwner, err := s.eventOwner(eventID)
	if err != nil {
		return nil, err
	}
	if actualOwner != ownerID {
		return nil, ErrForbidden
	}

	var a AttendeeRef
	err = s.db.QueryRow(`SELECT id, username, full_name FROM app_maintenance.users WHERE username = $1`, username).
		Scan(&a.UserID, &a.Username, &a.FullName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if _, err := s.db.Exec(
		`INSERT INTO event_attendees (event_id, user_id, invited_by) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		eventID, a.UserID, ownerID,
	); err != nil {
		return nil, err
	}
	return &a, nil
}

// RemoveAttendee is owner-only, same as Invite.
func (s *Store) RemoveAttendee(ownerID, eventID, userID string) error {
	actualOwner, err := s.eventOwner(eventID)
	if err != nil {
		return err
	}
	if actualOwner != ownerID {
		return ErrForbidden
	}
	_, err = s.db.Exec(`DELETE FROM event_attendees WHERE event_id = $1 AND user_id = $2`, eventID, userID)
	return err
}

// ListAllUsers is a cross-schema read of app-maintenance's user directory,
// used to power the invite search box (so users pick from a filtered
// list instead of having to type an exact username) - same pattern as
// kanban's ListAllUsers.
func (s *Store) ListAllUsers() ([]*AttendeeRef, error) {
	rows, err := s.db.Query(`SELECT id, username, full_name FROM app_maintenance.users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*AttendeeRef{}
	for rows.Next() {
		var u AttendeeRef
		if err := rows.Scan(&u.UserID, &u.Username, &u.FullName); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}
