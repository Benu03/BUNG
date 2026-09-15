package main

import (
	"database/sql"
	"errors"
)

// ErrNotFound is returned when a file/folder doesn't exist, or exists but
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

// ---- folders ----

// nullableID treats "" as SQL NULL - used for the root-level (no parent /
// no folder) case throughout this file.
func nullableID(id string) any {
	if id == "" {
		return nil
	}
	return id
}

func (s *Store) GetFolder(ownerID, id string) (*Folder, error) {
	var f Folder
	var parentID sql.NullString
	err := s.db.QueryRow(
		`SELECT id, owner_id, parent_id, name, created_at FROM folders WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	).Scan(&f.ID, &f.OwnerID, &parentID, &f.Name, &f.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	f.ParentID = parentID.String
	return &f, nil
}

func (s *Store) ListFolders(ownerID, parentID string) ([]*Folder, error) {
	rows, err := s.db.Query(
		`SELECT id, owner_id, parent_id, name, created_at FROM folders
		 WHERE owner_id = $1 AND parent_id IS NOT DISTINCT FROM $2
		 ORDER BY name`,
		ownerID, nullableID(parentID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	folders := []*Folder{}
	for rows.Next() {
		var f Folder
		var pid sql.NullString
		if err := rows.Scan(&f.ID, &f.OwnerID, &pid, &f.Name, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.ParentID = pid.String
		folders = append(folders, &f)
	}
	return folders, rows.Err()
}

func (s *Store) CreateFolder(ownerID, parentID, name string) (*Folder, error) {
	// A non-root parent must actually belong to this owner - otherwise a
	// user could nest a folder under someone else's by guessing an id.
	if parentID != "" {
		if _, err := s.GetFolder(ownerID, parentID); err != nil {
			return nil, err
		}
	}
	f := &Folder{OwnerID: ownerID, ParentID: parentID, Name: name}
	err := s.db.QueryRow(
		`INSERT INTO folders (owner_id, parent_id, name) VALUES ($1, $2, $3) RETURNING id, created_at`,
		ownerID, nullableID(parentID), name,
	).Scan(&f.ID, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Breadcrumb walks up from folderID to the root, returning entries in
// root-to-current order.
func (s *Store) Breadcrumb(ownerID, folderID string) ([]BreadcrumbEntry, error) {
	var entries []BreadcrumbEntry
	for folderID != "" {
		f, err := s.GetFolder(ownerID, folderID)
		if err != nil {
			return nil, err
		}
		entries = append([]BreadcrumbEntry{{ID: f.ID, Name: f.Name}}, entries...)
		folderID = f.ParentID
	}
	return entries, nil
}

// DeleteFolder removes a folder and (via ON DELETE CASCADE) every
// descendant folder/file row. Call ListFilesUnder first to clean up their
// on-disk blobs - this only touches the database.
func (s *Store) DeleteFolder(ownerID, id string) error {
	res, err := s.db.Exec(`DELETE FROM folders WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListFilesUnder returns every file in folderID and all of its descendant
// folders (recursively) - used to clean up on-disk blobs before deleting a
// folder, since the DB-level CASCADE only removes rows.
func (s *Store) ListFilesUnder(ownerID, folderID string) ([]*FileMeta, error) {
	rows, err := s.db.Query(`
		WITH RECURSIVE descendants AS (
			SELECT id FROM folders WHERE id = $1 AND owner_id = $2
			UNION ALL
			SELECT f.id FROM folders f JOIN descendants d ON f.parent_id = d.id
		)
		SELECT id, owner_id, folder_id, filename, content_type, size, storage_path, created_at
		FROM files WHERE owner_id = $2 AND folder_id IN (SELECT id FROM descendants)`,
		folderID, ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []*FileMeta{}
	for rows.Next() {
		var f FileMeta
		var fid sql.NullString
		if err := rows.Scan(&f.ID, &f.OwnerID, &fid, &f.Filename, &f.ContentType, &f.Size, &f.StoragePath, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.FolderID = fid.String
		files = append(files, &f)
	}
	return files, rows.Err()
}

// ---- files ----

func (s *Store) ListFiles(ownerID, folderID string) ([]*FileMeta, error) {
	rows, err := s.db.Query(
		`SELECT id, owner_id, folder_id, filename, content_type, size, storage_path, created_at
		 FROM files WHERE owner_id = $1 AND folder_id IS NOT DISTINCT FROM $2 ORDER BY created_at DESC`,
		ownerID, nullableID(folderID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []*FileMeta{}
	for rows.Next() {
		var f FileMeta
		var fid sql.NullString
		if err := rows.Scan(&f.ID, &f.OwnerID, &fid, &f.Filename, &f.ContentType, &f.Size, &f.StoragePath, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.FolderID = fid.String
		files = append(files, &f)
	}
	return files, rows.Err()
}

func (s *Store) GetFile(ownerID, id string) (*FileMeta, error) {
	var f FileMeta
	var fid sql.NullString
	err := s.db.QueryRow(
		`SELECT id, owner_id, folder_id, filename, content_type, size, storage_path, created_at
		 FROM files WHERE id = $1 AND owner_id = $2`, id, ownerID,
	).Scan(&f.ID, &f.OwnerID, &fid, &f.Filename, &f.ContentType, &f.Size, &f.StoragePath, &f.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	f.FolderID = fid.String
	return &f, nil
}

// CreateFile inserts a file row scoped to a folder (folderID == "" means
// root). If folderID is non-empty, it must belong to ownerID.
func (s *Store) CreateFile(f *FileMeta) (*FileMeta, error) {
	if f.FolderID != "" {
		if _, err := s.GetFolder(f.OwnerID, f.FolderID); err != nil {
			return nil, err
		}
	}
	err := s.db.QueryRow(
		`INSERT INTO files (owner_id, folder_id, filename, content_type, size, storage_path)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`,
		f.OwnerID, nullableID(f.FolderID), f.Filename, f.ContentType, f.Size, f.StoragePath,
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
