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

	db := openDB()
	defer db.Close()

	a := &api{store: NewStore(db)}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.health)

	mux.HandleFunc("GET /notifications", a.list)
	mux.HandleFunc("GET /notifications/unread-count", a.unreadCount)
	mux.HandleFunc("POST /notifications/{id}/read", a.markRead)
	mux.HandleFunc("POST /notifications/read-all", a.markAllRead)

	log.Printf("notifications-backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
