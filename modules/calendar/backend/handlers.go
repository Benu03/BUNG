package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
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
	if errors.Is(err, ErrForbidden) {
		writeErr(w, http.StatusForbidden, "forbidden")
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
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "module": "calendar"})
}

// me echoes back the identity nginx's auth_request already attached to the
// request, so the frontend can tell who's logged in - e.g. whether they
// own an event they can see (same pattern as kanban's /me).
func (a *api) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"id": currentUserID(r), "username": r.Header.Get("X-Username")})
}

// parseTimeParam parses an RFC3339 query param, returning nil if absent.
func parseTimeParam(r *http.Request, name string) (*time.Time, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// list returns the current user's events, optionally narrowed to a
// ?from=&to= range (RFC3339) - used for month/week views. Omit both to get
// every event.
func (a *api) list(w http.ResponseWriter, r *http.Request) {
	from, err := parseTimeParam(r, "from")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid from")
		return
	}
	to, err := parseTimeParam(r, "to")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid to")
		return
	}
	events, err := a.store.List(currentUserID(r), from, to)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (a *api) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	e, err := a.store.Get(currentUserID(r), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	attendees, err := a.store.ListAttendees(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, &eventDetail{Event: e, Attendees: attendees})
}

func validateEventRequest(in *eventRequest) string {
	if in.Title == "" {
		return "title is required"
	}
	if in.StartAt.IsZero() || in.EndAt.IsZero() {
		return "startAt and endAt are required"
	}
	if in.EndAt.Before(in.StartAt) {
		return "endAt must not be before startAt"
	}
	return ""
}

func (a *api) create(w http.ResponseWriter, r *http.Request) {
	var in eventRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if msg := validateEventRequest(&in); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	e, err := a.store.Create(currentUserID(r), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "event.create", "event", e.ID, map[string]any{"title": e.Title})
	writeJSON(w, http.StatusCreated, e)
}

func (a *api) update(w http.ResponseWriter, r *http.Request) {
	var in eventRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if msg := validateEventRequest(&in); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	e, err := a.store.Update(currentUserID(r), r.PathValue("id"), &in)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "event.update", "event", e.ID, map[string]any{"title": e.Title})
	writeJSON(w, http.StatusOK, e)
}

func (a *api) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.store.Delete(currentUserID(r), id); err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "event.delete", "event", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

// ---- attendees ----

func (a *api) invite(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")
	var in inviteRequest
	if err := decodeJSON(r, &in); err != nil || in.Username == "" {
		writeErr(w, http.StatusBadRequest, "username is required")
		return
	}

	attendee, err := a.store.Invite(currentUserID(r), eventID, in.Username)
	if err != nil {
		handleErr(w, err)
		return
	}

	e, err := a.store.Get(currentUserID(r), eventID)
	if err == nil {
		notify(a.store.db, attendee.UserID, "calendar", "event.invited",
			"You were invited to an event",
			fmt.Sprintf("You were invited to %q.", e.Title),
			"/calendar/",
		)
	}

	writeAudit(a.store.db, r, "event.invite", "event", eventID, map[string]any{"username": attendee.Username})
	writeJSON(w, http.StatusCreated, attendee)
}

func (a *api) removeAttendee(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")
	userID := r.PathValue("userId")
	if err := a.store.RemoveAttendee(currentUserID(r), eventID, userID); err != nil {
		handleErr(w, err)
		return
	}
	writeAudit(a.store.db, r, "event.uninvite", "event", eventID, map[string]any{"userId": userID})
	w.WriteHeader(http.StatusNoContent)
}
