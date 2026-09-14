package main

import (
	"encoding/json"
	"net/http"
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
	writeJSON(w, http.StatusOK, a.store.ListUsers())
}

func (a *api) getUser(w http.ResponseWriter, r *http.Request) {
	u, ok := a.store.GetUser(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (a *api) createUser(w http.ResponseWriter, r *http.Request) {
	var in User
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	writeJSON(w, http.StatusCreated, a.store.CreateUser(&in))
}

func (a *api) updateUser(w http.ResponseWriter, r *http.Request) {
	var in User
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	u, ok := a.store.UpdateUser(r.PathValue("id"), &in)
	if !ok {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (a *api) deleteUser(w http.ResponseWriter, r *http.Request) {
	if !a.store.DeleteUser(r.PathValue("id")) {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
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
	u, ok := a.store.SetUserRoles(r.PathValue("id"), in.RoleIDs)
	if !ok {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// ---- roles ----

func (a *api) listRoles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.store.ListRoles())
}

func (a *api) getRole(w http.ResponseWriter, r *http.Request) {
	role, ok := a.store.GetRole(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "role not found")
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
	writeJSON(w, http.StatusCreated, a.store.CreateRole(&in))
}

func (a *api) updateRole(w http.ResponseWriter, r *http.Request) {
	var in Role
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	role, ok := a.store.UpdateRole(r.PathValue("id"), &in)
	if !ok {
		writeErr(w, http.StatusNotFound, "role not found")
		return
	}
	writeJSON(w, http.StatusOK, role)
}

func (a *api) deleteRole(w http.ResponseWriter, r *http.Request) {
	if !a.store.DeleteRole(r.PathValue("id")) {
		writeErr(w, http.StatusNotFound, "role not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- modules ----

func (a *api) listModules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.store.ListModules())
}

func (a *api) getModule(w http.ResponseWriter, r *http.Request) {
	m, ok := a.store.GetModule(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "module not found")
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
	writeJSON(w, http.StatusCreated, a.store.CreateModule(&in))
}

func (a *api) updateModule(w http.ResponseWriter, r *http.Request) {
	var in Module
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	m, ok := a.store.UpdateModule(r.PathValue("id"), &in)
	if !ok {
		writeErr(w, http.StatusNotFound, "module not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (a *api) deleteModule(w http.ResponseWriter, r *http.Request) {
	if !a.store.DeleteModule(r.PathValue("id")) {
		writeErr(w, http.StatusNotFound, "module not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
