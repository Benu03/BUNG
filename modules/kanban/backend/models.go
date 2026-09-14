package main

import "time"

// Board is a kanban board, containing an ordered list of columns.
type Board struct {
	ID          string    `json:"id"`
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

// Card is a single kanban card living inside a column.
type Card struct {
	ID          string    `json:"id"`
	ColumnID    string    `json:"columnId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
