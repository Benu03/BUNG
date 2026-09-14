package main

import (
	"database/sql"
	"errors"
)

// ErrNotFound is returned when a file doesn't exist, or exists but belongs
// to someone else - every query here is scoped to owner_id, so "not yours"
// and "doesn't exist" look the same on purpose.
var ErrNotFound = errors.New("not found")

// Store is backed by Postgres, scoped to this module's schema via the
// connection's search_path (see db.go).
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) ListFiles(ownerID string) ([]*FileMeta, error) {
	rows, err := s.db.Query(
		`SELECT id, owner_id, filename, content_type, size, storage_path, created_at
		 FROM files WHERE owner_id = $1 ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []*FileMeta{}
	for rows.Next() {
		var f FileMeta
		if err := rows.Scan(&f.ID, &f.OwnerID, &f.Filename, &f.ContentType, &f.Size, &f.StoragePath, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	return files, rows.Err()
}

func (s *Store) GetFile(ownerID, id string) (*FileMeta, error) {
	var f FileMeta
	err := s.db.QueryRow(
		`SELECT id, owner_id, filename, content_type, size, storage_path, created_at
		 FROM files WHERE id = $1 AND owner_id = $2`, id, ownerID,
	).Scan(&f.ID, &f.OwnerID, &f.Filename, &f.ContentType, &f.Size, &f.StoragePath, &f.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) CreateFile(f *FileMeta) (*FileMeta, error) {
	err := s.db.QueryRow(
		`INSERT INTO files (owner_id, filename, content_type, size, storage_path)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		f.OwnerID, f.Filename, f.ContentType, f.Size, f.StoragePath,
	).Scan(&f.ID, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// UpdateFileStorage fills in the real storage_path and size once the blob
// has actually been written (see uploadFile in handlers.go, which creates
// the row first to get an id, then writes the file under that id).
func (s *Store) UpdateFileStorage(id, storagePath string, size int64) error {
	_, err := s.db.Exec(`UPDATE files SET storage_path = $1, size = $2 WHERE id = $3`, storagePath, size, id)
	return err
}

// DeleteFile deletes the metadata row and returns it (so the caller can
// remove the underlying blob) - or ErrNotFound if it doesn't exist / isn't
// owned by ownerID.
func (s *Store) DeleteFile(ownerID, id string) (*FileMeta, error) {
	f, err := s.GetFile(ownerID, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(`DELETE FROM files WHERE id = $1`, id); err != nil {
		return nil, err
	}
	return f, nil
}
