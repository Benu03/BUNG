package main

import (
	"encoding/json"
	"errors"
	"log"
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

// handleErr maps a Store error to the right HTTP response.
func handleErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
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
	var in User
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	u, err := a.store.CreateUser(&in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (a *api) updateUser(w http.ResponseWriter, r *http.Request) {
	var in User
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	u, err := a.store.UpdateUser(r.PathValue("id"), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (a *api) deleteUser(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteUser(r.PathValue("id")); err != nil {
		handleErr(w, err)
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
	u, err := a.store.SetUserRoles(r.PathValue("id"), in.RoleIDs)
	if err != nil {
		handleErr(w, err)
		return
	}
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
	role, err := a.store.CreateRole(&in)
	if err != nil {
		handleErr(w, err)
		return
	}
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
	writeJSON(w, http.StatusOK, role)
}

func (a *api) deleteRole(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteRole(r.PathValue("id")); err != nil {
		handleErr(w, err)
		return
	}
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
	writeJSON(w, http.StatusOK, m)
}

func (a *api) deleteModule(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteModule(r.PathValue("id")); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
