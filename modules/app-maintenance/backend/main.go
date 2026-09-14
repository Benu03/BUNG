package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}
	tokenTTL := 12 * time.Hour
	if v := os.Getenv("AUTH_TOKEN_TTL_HOURS"); v != "" {
		if hours, err := time.ParseDuration(v + "h"); err == nil {
			tokenTTL = hours
		}
	}

	db := openDB()
	defer db.Close()

	store := NewStore(db)
	a := &api{store: store}
	auth := newAuthAPI(store, jwtSecret, tokenTTL)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.health)

	// auth - not gated by nginx's auth_request (login/verify would be
	// circular, and the frontend needs an unauthenticated way to ask
	// "am I logged in").
	mux.HandleFunc("POST /auth/login", auth.login)
	mux.HandleFunc("POST /auth/logout", auth.logout)
	mux.HandleFunc("GET /auth/me", auth.me)
	mux.HandleFunc("GET /auth/verify", auth.verify)

	mux.HandleFunc("GET /users", a.listUsers)
	mux.HandleFunc("POST /users", a.createUser)
	mux.HandleFunc("GET /users/{id}", a.getUser)
	mux.HandleFunc("PUT /users/{id}", a.updateUser)
	mux.HandleFunc("DELETE /users/{id}", a.deleteUser)
	mux.HandleFunc("PUT /users/{id}/roles", a.setUserRoles)
	mux.HandleFunc("PUT /users/{id}/password", a.setUserPassword)

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

	mux.HandleFunc("GET /settings", a.getSettings)
	mux.HandleFunc("PUT /settings", a.updateSettings)
	mux.HandleFunc("GET /settings/public", a.getPublicSettings) // public, see nginx.conf

	mux.HandleFunc("GET /audit-log", a.listAuditLog)

	log.Printf("app-maintenance-backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
