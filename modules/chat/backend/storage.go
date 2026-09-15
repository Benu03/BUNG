package main

import (
	"io"
	"os"
	"path/filepath"
)

// blobStore writes/reads/removes attachment bytes on local disk, under a
// Docker volume (see DATA_DIR / the chat-data volume in
// docker-compose.yml) - same pattern as my-storage/ticketing's blobStore.
type blobStore struct {
	root string
}

func newBlobStore(root string) *blobStore {
	return &blobStore{root: root}
}

// path lays out bytes as <root>/<conversationId>/<messageId> - grouped by
// conversation purely for discoverability on disk, not access control.
func (b *blobStore) path(conversationID, messageID string) string {
	return filepath.Join(b.root, conversationID, messageID)
}

func (b *blobStore) Save(conversationID, messageID string, r io.Reader) (path string, size int64, err error) {
	path = b.path(conversationID, messageID)
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
