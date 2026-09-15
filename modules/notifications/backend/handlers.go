package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type api struct {
	store *Store
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// currentUserID reads the identity nginx's auth_request already verified
// (see /nginx/auth-common.conf) - this module trusts the header because
// nginx only forwards it after a successful check, and only nginx can
// reach this backend's port (it isn't published to the host). Unlike
// other modules, notifications' gateway locations don't set
// $auth_module_code, so ANY logged-in user can reach this API regardless
// of which modules they've been granted - see /modules/notifications/nginx.conf.
func currentUserID(r *http.Request) string {
	return r.Header.Get("X-User-Id")
}

func queryInt(r *http.Request, key string, def, max int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "module": "notifications"})
}

func (a *api) list(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 20, 100)
	offset := queryInt(r, "offset", 0, 1_000_000)
	notifications, err := a.store.List(currentUserID(r), limit, offset)
	if err != nil {
		log.Printf("list notifications: %v", err)
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, notifications)
}

func (a *api) unreadCount(w http.ResponseWriter, r *http.Request) {
	count, err := a.store.UnreadCount(currentUserID(r))
	if err != nil {
		log.Printf("unread count: %v", err)
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (a *api) markRead(w http.ResponseWriter, r *http.Request) {
	if err := a.store.MarkRead(currentUserID(r), r.PathValue("id")); err != nil {
		log.Printf("mark read: %v", err)
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) markAllRead(w http.ResponseWriter, r *http.Request) {
	if err := a.store.MarkAllRead(currentUserID(r)); err != nil {
		log.Printf("mark all read: %v", err)
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
