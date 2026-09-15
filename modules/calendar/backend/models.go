package main

import "time"

// Event is one calendar entry. Everything here is private: every query is
// scoped to owner_id = the requesting user (see store.go) - there's no
// sharing/invite in this module (unlike kanban's boards). owner_id has no
// FK to app_maintenance.users - modules stay loosely coupled at the schema
// level (same note as kanban/my-storage's migrate.go).
type Event struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"ownerId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	StartAt     time.Time `json:"startAt"`
	EndAt       time.Time `json:"endAt"`
	AllDay      bool      `json:"allDay"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type eventRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	StartAt     time.Time `json:"startAt"`
	EndAt       time.Time `json:"endAt"`
	AllDay      bool      `json:"allDay"`
}
