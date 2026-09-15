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

	mux.HandleFunc("GET /tickets", a.list)
	mux.HandleFunc("POST /tickets", a.create)
	mux.HandleFunc("GET /tickets/{id}", a.get)
	mux.HandleFunc("PUT /tickets/{id}", a.update)
	mux.HandleFunc("DELETE /tickets/{id}", a.delete)
	mux.HandleFunc("POST /tickets/{id}/comments", a.createComment)

	mux.HandleFunc("GET /users", a.listUsers) // for the requester/assignee picker

	log.Printf("ticketing-backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
