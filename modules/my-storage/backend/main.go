package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dataDir := getenv("DATA_DIR", "/data")

	db := openDB()
	defer db.Close()

	a := &api{store: NewStore(db), blobs: newBlobStore(dataDir)}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.health)

	// browse?folderId= (omitted/empty = root) returns that folder's
	// subfolders + files + breadcrumb in one call.
	mux.HandleFunc("GET /browse", a.browse)

	mux.HandleFunc("POST /folders", a.createFolder)
	mux.HandleFunc("DELETE /folders/{id}", a.deleteFolder)

	mux.HandleFunc("POST /files", a.uploadFile) // multipart: file, optional folderId
	mux.HandleFunc("GET /files/{id}/download", a.downloadFile)
	mux.HandleFunc("DELETE /files/{id}", a.deleteFile)

	log.Printf("my-storage-backend listening on :%s (data dir: %s)", port, dataDir)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
