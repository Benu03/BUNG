package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// validEmail is a deliberately loose sanity check (not full RFC 5322) -
// its job is just to reject the pathological cases (empty, no "@", or
// embedded whitespace/control characters like CR/LF) before an address
// ever reaches forgot-password's outgoing email (see email.go's
// smtpSender.Send, which interpolates it into raw message text). Real
// deliverability is whatever the mail relay decides.
var validEmail = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

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

// handleErr maps a Store error to the right HTTP response.
func handleErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, ErrConflict) {
		writeErr(w, http.StatusConflict, "username or email is already in use")
		return
	}
	log.Printf("internal error: %v", err)
	writeErr(w, http.StatusInternalServerError, "internal error")
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// ---- health ----

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "module": "app-maintenance"})
}

// ---- users ----

func (a *api) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.store.ListUsers()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (a *api) getUser(w http.ResponseWriter, r *http.Request) {
	u, err := a.store.GetUser(r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (a *api) createUser(w http.ResponseWriter, r *http.Request) {
	var in createUserRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(in.Password) < 6 {
		writeErr(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}
	if !validEmail.MatchString(in.Email) {
		writeErr(w, http.StatusBadRequest, "invalid email address")
		return
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		handleErr(w, err)
		return
	}
	user := &User{Username: in.Username, FullName: in.FullName, Email: in.Email, IsActive: in.IsActive}
	u, err := a.store.CreateUser(user, hash)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store, r, "user.create", "user", u.ID, map[string]any{"username": u.Username})
	writeJSON(w, http.StatusCreated, u)
}

func (a *api) setUserPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(in.Password) < 6 {
		writeErr(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		handleErr(w, err)
		return
	}
	if err := a.store.SetUserPassword(r.PathValue("id"), hash); err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store, r, "user.set_password", "user", r.PathValue("id"), nil)
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) updateUser(w http.ResponseWriter, r *http.Request) {
	var in User
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if !validEmail.MatchString(in.Email) {
		writeErr(w, http.StatusBadRequest, "invalid email address")
		return
	}
	u, err := a.store.UpdateUser(r.PathValue("id"), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store, r, "user.update", "user", u.ID, map[string]any{"username": u.Username})
	writeJSON(w, http.StatusOK, u)
}

func (a *api) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.store.DeleteUser(id); err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store, r, "user.delete", "user", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) setUserRoles(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RoleIDs []string `json:"roleIds"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	u, err := a.store.SetUserRoles(r.PathValue("id"), in.RoleIDs)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store, r, "user.set_roles", "user", u.ID, map[string]any{"roleIds": in.RoleIDs})
	writeJSON(w, http.StatusOK, u)
}

// ---- roles ----

func (a *api) listRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := a.store.ListRoles()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, roles)
}

func (a *api) getRole(w http.ResponseWriter, r *http.Request) {
	role, err := a.store.GetRole(r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, role)
}

func (a *api) createRole(w http.ResponseWriter, r *http.Request) {
	var in Role
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	in.IsActive = true // new roles start active; use PUT to deactivate one
	role, err := a.store.CreateRole(&in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store, r, "role.create", "role", role.ID, map[string]any{"name": role.Name})
	writeJSON(w, http.StatusCreated, role)
}

func (a *api) updateRole(w http.ResponseWriter, r *http.Request) {
	var in Role
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	role, err := a.store.UpdateRole(r.PathValue("id"), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store, r, "role.update", "role", role.ID, map[string]any{"name": role.Name})
	writeJSON(w, http.StatusOK, role)
}

func (a *api) deleteRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.store.DeleteRole(id); err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store, r, "role.delete", "role", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

// ---- modules ----

func (a *api) listModules(w http.ResponseWriter, r *http.Request) {
	modules, err := a.store.ListModules()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, modules)
}

func (a *api) getModule(w http.ResponseWriter, r *http.Request) {
	m, err := a.store.GetModule(r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (a *api) createModule(w http.ResponseWriter, r *http.Request) {
	var in Module
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	m, err := a.store.CreateModule(&in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store, r, "module.create", "module", m.ID, map[string]any{"code": m.Code})
	writeJSON(w, http.StatusCreated, m)
}

func (a *api) updateModule(w http.ResponseWriter, r *http.Request) {
	var in Module
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	m, err := a.store.UpdateModule(r.PathValue("id"), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store, r, "module.update", "module", m.ID, map[string]any{"code": m.Code})
	writeJSON(w, http.StatusOK, m)
}

func (a *api) deleteModule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.store.DeleteModule(id); err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store, r, "module.delete", "module", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

// ---- site settings ----

func (a *api) getSettings(w http.ResponseWriter, r *http.Request) {
	st, err := a.store.GetSiteSettings()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (a *api) updateSettings(w http.ResponseWriter, r *http.Request) {
	var in SiteSettings
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	st, err := a.store.UpdateSiteSettings(&in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store, r, "settings.update", "site_settings", "default", nil)
	writeJSON(w, http.StatusOK, st)
}

// getPublicSettings is reachable without a session (see nginx.conf's exact
// `/app-maintenance/api/settings/public` bypass) - the portal needs it
// before a user is logged in, to render the login page.
func (a *api) getPublicSettings(w http.ResponseWriter, r *http.Request) {
	st, err := a.store.GetSiteSettings()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"siteName":     st.SiteName,
		"tagline":      st.Tagline,
		"announcement": st.Announcement,
	})
}

// ---- audit log ----

// listAuditLog supports optional ?from=&to= (RFC3339, bounding
// occurred_at), ?user= (exact actor_username), ?ip= (exact ip_address) and
// ?requestId= (exact request_id, for tracing one specific request across
// nginx's access log and every backend's own logs) filters, applied
// server-side (so "load more" pagination stays correct against the
// filtered set) - on top of ?limit=&offset=.
func (a *api) listAuditLog(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50, 200)
	offset := queryInt(r, "offset", 0, 1_000_000)

	var filter AuditFilter
	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid from")
			return
		}
		filter.From = &t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid to")
			return
		}
		filter.To = &t
	}
	filter.ActorUsername = r.URL.Query().Get("user")
	filter.IPAddress = r.URL.Query().Get("ip")
	filter.RequestID = r.URL.Query().Get("requestId")

	entries, err := a.store.ListAuditLog(limit, offset, filter)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// queryInt reads an integer query param, falling back to def and clamping
// to [0, max].
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
