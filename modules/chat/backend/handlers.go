package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type api struct {
	store *Store
	blobs *blobStore
	hub   *hub
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
	if errors.Is(err, ErrConflict) {
		writeErr(w, http.StatusConflict, "conflict")
		return
	}
	log.Printf("internal error: %v", err)
	writeErr(w, http.StatusInternalServerError, "internal error")
}

func currentUserID(r *http.Request) string {
	return r.Header.Get("X-User-Id")
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "module": "chat"})
}

func (a *api) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"id": currentUserID(r), "username": r.Header.Get("X-Username")})
}

// isAdmin lets the frontend decide whether to show the broadcast composer
// - the actual enforcement is still server-side in createBroadcast, this
// is just so non-admins don't see a button that would 403.
func (a *api) isAdmin(w http.ResponseWriter, r *http.Request) {
	ok, err := a.store.IsAppMaintenanceAdmin(currentUserID(r))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"isAdmin": ok})
}

func (a *api) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.store.ListAllUsers()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// ---- friends ----

func (a *api) listFriends(w http.ResponseWriter, r *http.Request) {
	friends, err := a.store.ListFriends(currentUserID(r))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, friends)
}

func (a *api) listFriendRequests(w http.ResponseWriter, r *http.Request) {
	requests, err := a.store.ListFriendRequests(currentUserID(r))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, requests)
}

func (a *api) createFriendRequest(w http.ResponseWriter, r *http.Request) {
	var in friendRequestBody
	if err := decodeJSON(r, &in); err != nil || in.Username == "" {
		writeErr(w, http.StatusBadRequest, "username is required")
		return
	}
	fr, err := a.store.SendFriendRequest(currentUserID(r), in.Username)
	if err != nil {
		handleErr(w, err)
		return
	}
	if fr.Status == "accepted" {
		a.hub.send(fr.FromUser.ID, map[string]any{"type": "friend_accepted", "friend": fr.ToUser})
		a.hub.send(fr.ToUser.ID, map[string]any{"type": "friend_accepted", "friend": fr.FromUser})
	} else {
		a.hub.send(fr.ToUser.ID, map[string]any{"type": "friend_request", "request": fr})
	}
	writeJSON(w, http.StatusCreated, fr)
}

func (a *api) acceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	fr, err := a.store.AcceptFriendRequest(currentUserID(r), r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	a.hub.send(fr.FromUser.ID, map[string]any{"type": "friend_accepted", "friend": fr.ToUser})
	a.hub.send(fr.ToUser.ID, map[string]any{"type": "friend_accepted", "friend": fr.FromUser})
	writeJSON(w, http.StatusOK, fr)
}

func (a *api) declineFriendRequest(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeclineFriendRequest(currentUserID(r), r.PathValue("id")); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- conversations ----

func (a *api) listConversations(w http.ResponseWriter, r *http.Request) {
	conversations, err := a.store.ListConversations(currentUserID(r))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, conversations)
}

// startConversation is how a chat actually begins from the frontend's
// "friends" list - requires an accepted friendship, then lazily creates
// (or returns the existing) conversation row.
func (a *api) startConversation(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UserID string `json:"userId"`
	}
	if err := decodeJSON(r, &in); err != nil || in.UserID == "" {
		writeErr(w, http.StatusBadRequest, "userId is required")
		return
	}
	userID := currentUserID(r)
	ok, err := a.store.AreFriends(userID, in.UserID)
	if err != nil {
		handleErr(w, err)
		return
	}
	if !ok {
		writeErr(w, http.StatusForbidden, "not friends")
		return
	}
	convID, err := a.store.GetOrCreateConversation(userID, in.UserID)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": convID})
}

func (a *api) listMessages(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	messages, err := a.store.ListMessages(currentUserID(r), r.PathValue("id"), limit)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

func (a *api) sendMessage(w http.ResponseWriter, r *http.Request) {
	var in sendMessageRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if strings.TrimSpace(in.Body) == "" {
		writeErr(w, http.StatusBadRequest, "body is required")
		return
	}
	conversationID := r.PathValue("id")
	userID := currentUserID(r)
	msg, err := a.store.SendMessage(userID, conversationID, in.Body)
	if err != nil {
		handleErr(w, err)
		return
	}
	a.notifyNewMessage(userID, conversationID, msg)
	writeJSON(w, http.StatusCreated, msg)
}

// uploadAttachment accepts a single multipart file under field name
// "file" and sends it as a message - same two-step create-row-then-write
// pattern as my-storage/ticketing.
func (a *api) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("id")
	userID := currentUserID(r)

	file, header, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	filename := header.Filename
	if filename == "" {
		filename = "unnamed"
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	created, err := a.store.CreateAttachmentMessage(userID, conversationID, filename, contentType)
	if err != nil {
		handleErr(w, err)
		return
	}

	path, size, err := a.blobs.Save(conversationID, created.ID, file)
	if err != nil {
		handleErr(w, err)
		return
	}
	if err := a.store.UpdateMessageAttachment(created.ID, path, size); err != nil {
		handleErr(w, err)
		return
	}
	created.AttachmentStoragePath = &path
	created.AttachmentSize = &size

	a.notifyNewMessage(userID, conversationID, created)
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	meta, err := a.store.GetMessage(r.PathValue("messageId"))
	if err != nil {
		handleErr(w, err)
		return
	}
	// Only a participant of the message's conversation may download it.
	if _, err := a.store.conversationOtherUser(meta.ConversationID, userID); err != nil {
		handleErr(w, err)
		return
	}
	if meta.AttachmentStoragePath == nil {
		writeErr(w, http.StatusNotFound, "no attachment")
		return
	}

	f, err := a.blobs.Open(*meta.AttachmentStoragePath)
	if err != nil {
		handleErr(w, err)
		return
	}
	defer f.Close()

	if meta.AttachmentContentType != nil {
		w.Header().Set("Content-Type", *meta.AttachmentContentType)
	}
	filename := "file"
	if meta.AttachmentFilename != nil {
		filename = sanitizeHeaderValue(*meta.AttachmentFilename)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if _, err := io.Copy(w, f); err != nil {
		log.Printf("stream attachment %s: %v", meta.ID, err)
	}
}

func (a *api) markRead(w http.ResponseWriter, r *http.Request) {
	if err := a.store.MarkRead(currentUserID(r), r.PathValue("id")); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// notifyNewMessage pushes the new message to both participants over
// WebSocket (best-effort - offline users just see it next time they load
// the conversation).
func (a *api) notifyNewMessage(senderID, conversationID string, msg *Message) {
	otherID, err := a.store.conversationOtherUser(conversationID, senderID)
	if err != nil {
		return
	}
	payload := map[string]any{"type": "message", "conversationId": conversationID, "message": msg}
	a.hub.send(otherID, payload)
	a.hub.send(senderID, payload)
}

// ---- broadcasts ----

func (a *api) listBroadcasts(w http.ResponseWriter, r *http.Request) {
	broadcasts, err := a.store.ListBroadcasts(50)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, broadcasts)
}

// createBroadcast is restricted to app-maintenance admins - checked here
// (cross-schema), not at the nginx gateway. The gateway only confirms the
// caller has the "chat" module at all (see nginx.conf) - broadcasting to
// everyone is a stricter, separate permission on top of that, so it's
// enforced in-handler regardless of module grants.
func (a *api) createBroadcast(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	isAdmin, err := a.store.IsAppMaintenanceAdmin(userID)
	if err != nil {
		handleErr(w, err)
		return
	}
	if !isAdmin {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}

	var in broadcastRequest
	if err := decodeJSON(r, &in); err != nil || strings.TrimSpace(in.Body) == "" {
		writeErr(w, http.StatusBadRequest, "body is required")
		return
	}
	b, err := a.store.CreateBroadcast(userID, in.Body)
	if err != nil {
		handleErr(w, err)
		return
	}
	a.hub.broadcastAll(map[string]any{"type": "broadcast", "broadcast": b})
	writeJSON(w, http.StatusCreated, b)
}

// sanitizeHeaderValue strips characters that would break the
// Content-Disposition header's quoted filename syntax - same helper as
// my-storage/ticketing.
func sanitizeHeaderValue(s string) string {
	return strings.NewReplacer(`"`, "'", "\r", "", "\n", "").Replace(s)
}
