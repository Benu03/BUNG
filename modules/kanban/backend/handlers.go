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
	log.Printf("internal error: %v", err)
	writeErr(w, http.StatusInternalServerError, "internal error")
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// ---- health ----

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "module": "kanban"})
}

// ---- boards ----

func (a *api) listBoards(w http.ResponseWriter, r *http.Request) {
	boards, err := a.store.ListBoards()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, boards)
}

func (a *api) getBoard(w http.ResponseWriter, r *http.Request) {
	full, err := a.store.GetBoardFull(r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, full)
}

func (a *api) createBoard(w http.ResponseWriter, r *http.Request) {
	var in Board
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b, err := a.store.CreateBoard(&in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (a *api) updateBoard(w http.ResponseWriter, r *http.Request) {
	var in Board
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b, err := a.store.UpdateBoard(r.PathValue("id"), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (a *api) deleteBoard(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteBoard(r.PathValue("id")); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- columns ----

func (a *api) createColumn(w http.ResponseWriter, r *http.Request) {
	var in Column
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	in.BoardID = r.PathValue("id")
	c, err := a.store.CreateColumn(&in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (a *api) updateColumn(w http.ResponseWriter, r *http.Request) {
	var in Column
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	c, err := a.store.UpdateColumn(r.PathValue("id"), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *api) deleteColumn(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteColumn(r.PathValue("id")); err != nil {
		handleErr(w, err)
		return
	}
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
	c, err := a.store.CreateCard(&in)
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
	c, err := a.store.UpdateCard(r.PathValue("id"), &in)
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
	c, err := a.store.MoveCard(r.PathValue("id"), in.ColumnID, in.Position)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *api) deleteCard(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteCard(r.PathValue("id")); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
