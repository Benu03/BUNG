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

	a := &api{store: NewStore()}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.health)

	mux.HandleFunc("GET /users", a.listUsers)
	mux.HandleFunc("POST /users", a.createUser)
	mux.HandleFunc("GET /users/{id}", a.getUser)
	mux.HandleFunc("PUT /users/{id}", a.updateUser)
	mux.HandleFunc("DELETE /users/{id}", a.deleteUser)
	mux.HandleFunc("PUT /users/{id}/roles", a.setUserRoles)

	mux.HandleFunc("GET /roles", a.listRoles)
	mux.HandleFunc("POST /roles", a.createRole)
	mux.HandleFunc("GET /roles/{id}", a.getRole)
	mux.HandleFunc("PUT /roles/{id}", a.updateRole)
	mux.HandleFunc("DELETE /roles/{id}", a.deleteRole)

	mux.HandleFunc("GET /modules", a.listModules)
	mux.HandleFunc("POST /modules", a.createModule)
	mux.HandleFunc("GET /modules/{id}", a.getModule)
	mux.HandleFunc("PUT /modules/{id}", a.updateModule)
	mux.HandleFunc("DELETE /modules/{id}", a.deleteModule)

	log.Printf("app-maintenance-backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
