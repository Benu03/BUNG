package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

type api struct {
	store *Store
	blobs *blobStore
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func handleErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	log.Printf("internal error: %v", err)
	writeErr(w, http.StatusInternalServerError, "internal error")
}

// currentUserID reads the identity nginx's auth_request already verified
// (see /nginx/auth-common.conf) - this module trusts the header because
// nginx only forwards it after a successful check, and only nginx can
// reach this backend's port (it isn't published to the host).
func currentUserID(r *http.Request) string {
	return r.Header.Get("X-User-Id")
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "module": "my-storage"})
}

func (a *api) listFiles(w http.ResponseWriter, r *http.Request) {
	files, err := a.store.ListFiles(currentUserID(r))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, files)
}

// uploadFile accepts a single multipart file under field name "file".
// nginx's client_max_body_size (see /nginx/nginx.conf) caps upload size.
func (a *api) uploadFile(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)

	file, header, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	filename := header.Filename
	if filename == "" {
		filename = "unnamed"
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Reserve the row first to get an id, then write bytes under that id -
	// keeps the on-disk path collision-proof without a second query.
	// storage_path is filled in below once we know it.
	created, err := a.store.CreateFile(&FileMeta{OwnerID: userID, Filename: filename, ContentType: contentType, StoragePath: "pending"})
	if err != nil {
		handleErr(w, err)
		return
	}

	path, size, err := a.blobs.Save(userID, created.ID, file)
	if err != nil {
		_, _ = a.store.DeleteFile(userID, created.ID)
		handleErr(w, err)
		return
	}
	created.StoragePath = path
	created.Size = size

	// Persist the real path/size now that we know them.
	if err := a.store.UpdateFileStorage(created.ID, path, size); err != nil {
		handleErr(w, err)
		return
	}

	writeAudit(a.store.db, r, "file.upload", "file", created.ID, map[string]any{"filename": created.Filename, "size": size})
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) downloadFile(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	meta, err := a.store.GetFile(userID, r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}

	f, err := a.blobs.Open(meta.StoragePath)
	if err != nil {
		handleErr(w, err)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, meta.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", meta.Size))
	if _, err := io.Copy(w, f); err != nil {
		log.Printf("stream file %s: %v", meta.ID, err)
	}
}

func (a *api) deleteFile(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	meta, err := a.store.DeleteFile(userID, r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	if err := a.blobs.Delete(meta.StoragePath); err != nil {
		log.Printf("delete blob for file %s: %v", meta.ID, err)
	}
	writeAudit(a.store.db, r, "file.delete", "file", meta.ID, map[string]any{"filename": meta.Filename})
	w.WriteHeader(http.StatusNoContent)
}
