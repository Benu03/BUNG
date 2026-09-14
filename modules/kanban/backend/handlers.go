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

func handleErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, ErrForbidden) {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}
	log.Printf("internal error: %v", err)
	writeErr(w, http.StatusInternalServerError, "internal error")
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// currentUserID reads the identity nginx's auth_request already verified
// (see /nginx/auth-common.conf) - this module trusts the header because
// nginx only forwards it after a successful check, and only nginx can
// reach this backend's port (it isn't published to the host).
func currentUserID(r *http.Request) string {
	return r.Header.Get("X-User-Id")
}

// ---- health ----

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "module": "kanban"})
}

// me echoes back the identity nginx's auth_request already attached to the
// request (X-User-Id/X-Username), so the frontend can tell who's logged in
// - and e.g. whether they own a board - without needing access to the
// app-maintenance module itself.
func (a *api) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"id": currentUserID(r), "username": r.Header.Get("X-Username")})
}

// ---- boards ----

func (a *api) listBoards(w http.ResponseWriter, r *http.Request) {
	boards, err := a.store.ListBoards(currentUserID(r))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, boards)
}

func (a *api) getBoard(w http.ResponseWriter, r *http.Request) {
	full, err := a.store.GetBoardFull(currentUserID(r), r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, full)
}

// createBoard creates a board together with its initial columns (the
// workflow) in one request - see createBoardRequest.
func (a *api) createBoard(w http.ResponseWriter, r *http.Request) {
	var in createBoardRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	b, err := a.store.CreateBoardWithColumns(currentUserID(r), in.Name, in.Description, in.Columns)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "board.create", "board", b.ID, map[string]any{"name": b.Name})
	writeJSON(w, http.StatusCreated, b)
}

func (a *api) updateBoard(w http.ResponseWriter, r *http.Request) {
	var in Board
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b, err := a.store.UpdateBoard(currentUserID(r), r.PathValue("id"), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "board.update", "board", b.ID, map[string]any{"name": b.Name})
	writeJSON(w, http.StatusOK, b)
}

func (a *api) deleteBoard(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.store.DeleteBoard(currentUserID(r), id); err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "board.delete", "board", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

// ---- members ----

func (a *api) listMembers(w http.ResponseWriter, r *http.Request) {
	boardID := r.PathValue("id")
	// listing members still requires membership - piggyback on GetBoardFull's
	// check by requiring it explicitly here instead of duplicating a
	// membership-only lookup.
	if _, err := a.store.GetBoardFull(currentUserID(r), boardID); err != nil {
		handleErr(w, err)
		return
	}
	members, err := a.store.ListMembers(boardID)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, members)
}

func (a *api) addMember(w http.ResponseWriter, r *http.Request) {
	var in addMemberRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	boardID := r.PathValue("id")
	m, err := a.store.AddMember(currentUserID(r), boardID, in.Username)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "board.add_member", "board", boardID, map[string]any{"username": m.Username})
	writeJSON(w, http.StatusCreated, m)
}

func (a *api) removeMember(w http.ResponseWriter, r *http.Request) {
	boardID, userID := r.PathValue("id"), r.PathValue("userId")
	if err := a.store.RemoveMember(currentUserID(r), boardID, userID); err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "board.remove_member", "board", boardID, map[string]any{"userId": userID})
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.store.ListAllUsers()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// ---- columns ----

func (a *api) createColumn(w http.ResponseWriter, r *http.Request) {
	var in Column
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	in.BoardID = r.PathValue("id")
	c, err := a.store.CreateColumn(currentUserID(r), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "column.create", "column", c.ID, map[string]any{"name": c.Name, "boardId": c.BoardID})
	writeJSON(w, http.StatusCreated, c)
}

func (a *api) updateColumn(w http.ResponseWriter, r *http.Request) {
	var in Column
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	c, err := a.store.UpdateColumn(currentUserID(r), r.PathValue("id"), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "column.update", "column", c.ID, map[string]any{"name": c.Name})
	writeJSON(w, http.StatusOK, c)
}

func (a *api) deleteColumn(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.store.DeleteColumn(currentUserID(r), id); err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "column.delete", "column", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

// ---- cards ----

func (a *api) createCard(w http.ResponseWriter, r *http.Request) {
	var in Card
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	in.ColumnID = r.PathValue("id")
	c, err := a.store.CreateCard(currentUserID(r), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (a *api) updateCard(w http.ResponseWriter, r *http.Request) {
	var in Card
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	c, err := a.store.UpdateCard(currentUserID(r), r.PathValue("id"), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *api) moveCard(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ColumnID string `json:"columnId"`
		Position int    `json:"position"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	c, err := a.store.MoveCard(currentUserID(r), r.PathValue("id"), in.ColumnID, in.Position)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *api) deleteCard(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteCard(currentUserID(r), r.PathValue("id")); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
