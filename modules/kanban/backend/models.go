package main

import "time"

// Board is a kanban board, private to its members (see BoardMember) -
// containing an ordered list of columns.
type Board struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"ownerId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Column is a lane within a board (e.g. "To Do", "In Progress", "Done").
type Column struct {
	ID        string    `json:"id"`
	BoardID   string    `json:"boardId"`
	Name      string    `json:"name"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Card is a single kanban card living inside a column. AssigneeID has no
// FK to app_maintenance.users, same loose-coupling reasoning as
// board_members (see migrate.go).
type Card struct {
	ID          string     `json:"id"`
	ColumnID    string     `json:"columnId"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Position    int        `json:"position"`
	AssigneeID  *string    `json:"assigneeId"`
	DueDate     *time.Time `json:"dueDate"`
	Color       string     `json:"color"` // '' or one of validCardColors
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// validCardColors is the small preset palette cards can be tagged with -
// matching the app's pastel theme rather than letting free-text colors in.
var validCardColors = map[string]bool{
	"": true, "rose": true, "amber": true, "emerald": true, "sky": true, "violet": true,
}

// BoardMember is one user's membership in a board. "owner" can manage
// members and delete the board; "member" can do everything else (edit the
// board, manage columns/cards).
type BoardMember struct {
	UserID   string    `json:"userId"`
	Username string    `json:"username"`
	FullName string    `json:"fullName"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joinedAt"`
}

// UserRef is a minimal, read-only view of an app-maintenance user, used for
// the "invite a member" search (see store.go's ListAllUsers - a
// cross-schema read, same pattern as the audit log).
type UserRef struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"fullName"`
}

// ColumnWithCards and BoardFull compose a whole board (with its columns and
// each column's cards) into a single response, so the frontend can render a
// board with one request.
type ColumnWithCards struct {
	Column
	Cards []*Card `json:"cards"`
}

type BoardFull struct {
	Board
	Columns []*ColumnWithCards `json:"columns"`
}

// createBoardRequest is the payload for POST /boards - creating a board
// together with its initial workflow (columns) in one step, rather than an
// empty board you then add columns to one at a time.
type createBoardRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Columns     []string `json:"columns"` // initial column names, in order
}

type addMemberRequest struct {
	Username string `json:"username"`
}
