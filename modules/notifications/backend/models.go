package main

import "time"

// Notification is one entry in a user's notification inbox. Any module's
// backend can create one for any user via a fully-qualified INSERT into
// notifications.inbox (same pattern as writing to the shared audit log) -
// see the notify() helper duplicated into each module's backend
// (kanban/notify.go, etc.).
type Notification struct {
	ID          string     `json:"id"`
	RecipientID string     `json:"recipientId"`
	ModuleCode  string     `json:"moduleCode"` // which module this notification is about
	Type        string     `json:"type"`       // e.g. "board.member_added"
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Link        string     `json:"link"` // where clicking it should navigate to, e.g. /kanban/
	ReadAt      *time.Time `json:"readAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}
