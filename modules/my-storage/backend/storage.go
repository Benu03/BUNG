package main

import (
	"io"
	"os"
	"path/filepath"
)

// blobStore writes/reads/removes file bytes on local disk, under a Docker
// volume (see DATA_DIR / the my-storage-data volume in docker-compose.yml).
// Only this module's backend ever touches this path - Postgres just tracks
// metadata (see FileMeta.StoragePath).
type blobStore struct {
	root string
}

func newBlobStore(root string) *blobStore {
	return &blobStore{root: root}
}

// path returns where a given owner+file's bytes live, laid out as
// <root>/<ownerId>/<fileId> so one owner's files are trivially easy to
// find (or bulk-delete) on disk if ever needed outside the API.
func (b *blobStore) path(ownerID, fileID string) string {
	return filepath.Join(b.root, ownerID, fileID)
}

func (b *blobStore) Save(ownerID, fileID string, r io.Reader) (path string, size int64, err error) {
	path = b.path(ownerID, fileID)
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
