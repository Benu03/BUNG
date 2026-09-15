package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type api struct {
	store *Store
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// currentUserID reads the identity nginx's auth_request already verified
// (see /nginx/auth-common.conf) - this module trusts the header because
// nginx only forwards it after a successful check, and only nginx can
// reach this backend's port (it isn't published to the host).
func currentUserID(r *http.Request) string {
	return r.Header.Get("X-User-Id")
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "module": "search"})
}

// search is deliberately reachable by any logged-in user (see
// /modules/search/nginx.conf, which doesn't set $auth_module_code, same
// as notifications) - but every result set below is itself scoped to
// only the modules/data userID actually has access to, so the open
// gateway route never leaks anything a per-module grant would have
// blocked. Results across modules run independently; one module's query
// failing doesn't take down the others.
func (a *api) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		writeJSON(w, http.StatusOK, []Result{})
		return
	}
	userID := currentUserID(r)
	like := "%" + q + "%"

	granted, err := a.store.GrantedModules(userID)
	if err != nil {
		log.Printf("granted modules lookup failed: %v", err)
		writeJSON(w, http.StatusOK, []Result{})
		return
	}

	results := []Result{}
	add := func(rs []Result, err error, module string) {
		if err != nil {
			log.Printf("search %s failed: %v", module, err)
			return
		}
		results = append(results, rs...)
	}

	if granted["ticketing"] {
		rs, err := a.store.SearchTickets(like)
		add(rs, err, "ticketing")
	}
	if granted["calendar"] {
		rs, err := a.store.SearchEvents(like, userID)
		add(rs, err, "calendar")
	}
	if granted["my-storage"] {
		rs, err := a.store.SearchFiles(like, userID)
		add(rs, err, "my-storage")
	}
	if granted["kanban"] {
		rs, err := a.store.SearchBoards(like, userID)
		add(rs, err, "kanban")
	}
	if granted["app-maintenance"] {
		rs, err := a.store.SearchUsers(like)
		add(rs, err, "app-maintenance")
	}

	writeJSON(w, http.StatusOK, results)
}
