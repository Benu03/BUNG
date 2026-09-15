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
	mux.HandleFunc("GET /me", a.me)

	// list?from=&to= (both optional, RFC3339) - omit both for everything.
	mux.HandleFunc("GET /events", a.list)
	mux.HandleFunc("POST /events", a.create)
	mux.HandleFunc("GET /events/{id}", a.get)
	mux.HandleFunc("PUT /events/{id}", a.update)
	mux.HandleFunc("DELETE /events/{id}", a.delete)

	mux.HandleFunc("POST /events/{id}/invite", a.invite)
	mux.HandleFunc("DELETE /events/{id}/invite/{userId}", a.removeAttendee)

	mux.HandleFunc("GET /users", a.listUsers) // for the invite search box

	log.Printf("calendar-backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
