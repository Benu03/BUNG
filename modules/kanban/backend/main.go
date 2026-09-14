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

	mux.HandleFunc("GET /boards", a.listBoards)
	mux.HandleFunc("POST /boards", a.createBoard)
	mux.HandleFunc("GET /boards/{id}", a.getBoard)
	mux.HandleFunc("PUT /boards/{id}", a.updateBoard)
	mux.HandleFunc("DELETE /boards/{id}", a.deleteBoard)
	mux.HandleFunc("POST /boards/{id}/columns", a.createColumn)

	mux.HandleFunc("GET /boards/{id}/members", a.listMembers)
	mux.HandleFunc("POST /boards/{id}/members", a.addMember)
	mux.HandleFunc("DELETE /boards/{id}/members/{userId}", a.removeMember)
	mux.HandleFunc("GET /users", a.listUsers) // for the "add member" search

	mux.HandleFunc("PUT /columns/{id}", a.updateColumn)
	mux.HandleFunc("DELETE /columns/{id}", a.deleteColumn)
	mux.HandleFunc("POST /columns/{id}/cards", a.createCard)

	mux.HandleFunc("PUT /cards/{id}", a.updateCard)
	mux.HandleFunc("PUT /cards/{id}/move", a.moveCard)
	mux.HandleFunc("DELETE /cards/{id}", a.deleteCard)

	log.Printf("kanban-backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
