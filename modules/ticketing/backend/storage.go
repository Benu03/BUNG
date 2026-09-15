package main

import (
	"io"
	"os"
	"path/filepath"
)

// blobStore writes/reads/removes attachment bytes on local disk, under a
// Docker volume (see DATA_DIR / the ticketing-data volume in
// docker-compose.yml). Only this module's backend ever touches this path -
// Postgres just tracks metadata (see Attachment.StoragePath). Same
// pattern as my-storage's blobStore.
type blobStore struct {
	root string
}

func newBlobStore(root string) *blobStore {
	return &blobStore{root: root}
}

// path returns where a given attachment's bytes live, laid out as
// <root>/<ticketId>/<attachmentId> - grouped by ticket purely for
// discoverability on disk, not access control (that's enforced entirely
// at the DB/API layer).
func (b *blobStore) path(ticketID, attachmentID string) string {
	return filepath.Join(b.root, ticketID, attachmentID)
}

func (b *blobStore) Save(ticketID, attachmentID string, r io.Reader) (path string, size int64, err error) {
	path = b.path(ticketID, attachmentID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", 0, err
	}
	f, err := os.Create(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return "", 0, err
	}
	return path, n, nil
}

func (b *blobStore) Open(path string) (*os.File, error) {
	return os.Open(path)
}

func (b *blobStore) Delete(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
