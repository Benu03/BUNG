package main

import (
	"encoding/json"
	"errors"
	"fmt"
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

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func handleErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	log.Printf("internal error: %v", err)
	writeErr(w, http.StatusInternalServerError, "internal error")
}

// currentUserID reads the identity nginx's auth_request already verified
// (see /nginx/auth-common.conf) - this module trusts the header because
// nginx only forwards it after a successful check, and only nginx can
// reach this backend's port (it isn't published to the host).
func currentUserID(r *http.Request) string {
	return r.Header.Get("X-User-Id")
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "module": "ticketing"})
}

// ---- tickets ----

func (a *api) list(w http.ResponseWriter, r *http.Request) {
	tickets, err := a.store.List()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tickets)
}

func (a *api) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := a.store.Get(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	comments, err := a.store.ListComments(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, &ticketDetail{Ticket: t, Comments: comments})
}

func (a *api) create(w http.ResponseWriter, r *http.Request) {
	var in createTicketRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Title == "" {
		writeErr(w, http.StatusBadRequest, "title is required")
		return
	}
	if in.Priority == "" {
		in.Priority = "medium"
	}
	if !validPriorities[in.Priority] {
		writeErr(w, http.StatusBadRequest, "invalid priority")
		return
	}
	t, err := a.store.Create(currentUserID(r), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "ticket.create", "ticket", t.ID, map[string]any{"title": t.Title})
	writeJSON(w, http.StatusCreated, t)
}

func (a *api) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in updateTicketRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Title == "" {
		writeErr(w, http.StatusBadRequest, "title is required")
		return
	}
	if !validStatuses[in.Status] {
		writeErr(w, http.StatusBadRequest, "invalid status")
		return
	}
	if !validPriorities[in.Priority] {
		writeErr(w, http.StatusBadRequest, "invalid priority")
		return
	}

	before, err := a.store.Get(id)
	if err != nil {
		handleErr(w, err)
		return
	}

	t, err := a.store.Update(id, &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "ticket.update", "ticket", t.ID, map[string]any{"title": t.Title, "status": t.Status})

	actor := currentUserID(r)
	link := "/ticketing/"

	// Newly assigned (or reassigned) to someone else - let them know.
	if t.AssigneeID != nil && (before.AssigneeID == nil || *before.AssigneeID != *t.AssigneeID) && *t.AssigneeID != actor {
		notify(a.store.db, *t.AssigneeID, "ticketing", "ticket.assigned",
			"You were assigned a ticket",
			fmt.Sprintf("%q was assigned to you.", t.Title),
			link,
		)
	}
	// Just got resolved/closed - let the requester know (unless they did it themselves).
	statusJustClosed := (t.Status == "resolved" || t.Status == "closed") && before.Status != t.Status
	if statusJustClosed && t.RequesterID != actor {
		notify(a.store.db, t.RequesterID, "ticketing", "ticket.status",
			"Your ticket was "+t.Status,
			fmt.Sprintf("%q is now %s.", t.Title, t.Status),
			link,
		)
	}

	writeJSON(w, http.StatusOK, t)
}

func (a *api) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.store.Delete(id); err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "ticket.delete", "ticket", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

// ---- comments ----

func (a *api) createComment(w http.ResponseWriter, r *http.Request) {
	ticketID := r.PathValue("id")
	var in createCommentRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Body == "" {
		writeErr(w, http.StatusBadRequest, "body is required")
		return
	}

	t, err := a.store.Get(ticketID)
	if err != nil {
		handleErr(w, err)
		return
	}

	actor := currentUserID(r)
	c, err := a.store.CreateComment(ticketID, actor, in.Body)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "ticket.comment", "ticket", ticketID, nil)

	// Notify whoever's on the ticket that isn't the commenter - the
	// requester, and the assignee if one is set.
	link := "/ticketing/"
	notifyBody := fmt.Sprintf("New comment on %q.", t.Title)
	if t.RequesterID != actor {
		notify(a.store.db, t.RequesterID, "ticketing", "ticket.comment", "New comment on your ticket", notifyBody, link)
	}
	if t.AssigneeID != nil && *t.AssigneeID != actor && *t.AssigneeID != t.RequesterID {
		notify(a.store.db, *t.AssigneeID, "ticketing", "ticket.comment", "New comment on an assigned ticket", notifyBody, link)
	}

	writeJSON(w, http.StatusCreated, c)
}

// ---- users ----

func (a *api) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.store.ListAllUsers()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}
