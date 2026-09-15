package main

import "time"

var validStatuses = map[string]bool{"open": true, "in_progress": true, "resolved": true, "closed": true}
var validPriorities = map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}

// Ticket is a SHARED helpdesk queue entry - unlike kanban/my-storage/
// calendar, this is NOT private per owner. Anyone with access to the
// "ticketing" module can see, comment on, and update any ticket. There's
// no separate "agent" vs "requester" role in this app's per-module role
// model yet, so this module treats every user with ticketing access as
// staff+requester combined - a deliberate MVP simplification, not an
// oversight (see nginx's $auth_module_code "ticketing" gate, which is the
// only access control this module has).
type Ticket struct {
	ID          string     `json:"id"`
	RequesterID string     `json:"requesterId"`
	AssigneeID  *string    `json:"assigneeId"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`   // open | in_progress | resolved | closed
	Priority    string     `json:"priority"` // low | medium | high | urgent
	Category    string     `json:"category"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	ResolvedAt  *time.Time `json:"resolvedAt"`
}

// ticketDetail is what GET /tickets/{id} returns - the ticket plus its
// full comment thread in one round trip.
type ticketDetail struct {
	*Ticket
	Comments []*Comment `json:"comments"`
}

type Comment struct {
	ID        string    `json:"id"`
	TicketID  string    `json:"ticketId"`
	AuthorID  string    `json:"authorId"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

// UserRef is a minimal, read-only view of an app-maintenance user, used to
// populate the requester/assignee display and the assignee picker - same
// pattern as kanban's UserRef (cross-schema read, see store.go).
type UserRef struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"fullName"`
}

type createTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Priority    string `json:"priority"`
}

type updateTicketRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Priority    string  `json:"priority"`
	Status      string  `json:"status"`
	AssigneeID  *string `json:"assigneeId"`
}

type createCommentRequest struct {
	Body string `json:"body"`
}
