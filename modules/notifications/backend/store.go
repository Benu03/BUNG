package main

import (
	"database/sql"
	"errors"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List(recipientID string, limit, offset int) ([]*Notification, error) {
	rows, err := s.db.Query(
		`SELECT id, recipient_id, module_code, type, title, body, link, read_at, created_at
		 FROM inbox WHERE recipient_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		recipientID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := []*Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.RecipientID, &n.ModuleCode, &n.Type, &n.Title, &n.Body, &n.Link, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, &n)
	}
	return notifications, rows.Err()
}

func (s *Store) UnreadCount(recipientID string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT count(*) FROM inbox WHERE recipient_id = $1 AND read_at IS NULL`, recipientID).Scan(&count)
	return count, err
}

// MarkRead marks one notification read - only if it belongs to recipientID
// (so a user can't mark someone else's notification read by guessing an id).
func (s *Store) MarkRead(recipientID, id string) error {
	res, err := s.db.Exec(
		`UPDATE inbox SET read_at = now() WHERE id = $1 AND recipient_id = $2 AND read_at IS NULL`,
		id, recipientID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Either it doesn't exist, isn't this user's, or was already read -
		// any of those is fine to treat as a no-op success rather than an
		// error the frontend has to handle specially.
		return nil
	}
	return nil
}

func (s *Store) MarkAllRead(recipientID string) error {
	_, err := s.db.Exec(`UPDATE inbox SET read_at = now() WHERE recipient_id = $1 AND read_at IS NULL`, recipientID)
	return err
}
