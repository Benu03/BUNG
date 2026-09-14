package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// schemaSQL creates this module's tables. It relies on the connection's
// search_path (set in openDB) already pointing at this module's schema, so
// table names here are unqualified on purpose.
const schemaSQL = `
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

CREATE TABLE IF NOT EXISTS boards (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS columns (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	position INT NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS cards (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	column_id UUID NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	position INT NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

// migrate creates this module's schema (if missing) and its tables, then
// seeds a starter board so the UI is usable right away.
func migrate(db *sql.DB, schema string) error {
	createSchema := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s;`, quoteIdent(schema))
	if _, err := db.Exec(createSchema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}
	if err := seed(db); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	return nil
}

// quoteIdent does minimal safe quoting for identifiers that come from our
// own env config (module schema names), not from user input.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func seed(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM boards`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var boardID string
	if err := db.QueryRow(
		`INSERT INTO boards (name, description) VALUES ($1, $2) RETURNING id`,
		"Getting Started", "Default starter board",
	).Scan(&boardID); err != nil {
		return fmt.Errorf("seed board: %w", err)
	}

	columns := []string{"To Do", "In Progress", "Done"}
	var todoID string
	for i, name := range columns {
		var colID string
		if err := db.QueryRow(
			`INSERT INTO columns (board_id, name, position) VALUES ($1, $2, $3) RETURNING id`,
			boardID, name, i,
		).Scan(&colID); err != nil {
			return fmt.Errorf("seed column: %w", err)
		}
		if i == 0 {
			todoID = colID
		}
	}

	if _, err := db.Exec(
		`INSERT INTO cards (column_id, title, description, position) VALUES ($1, $2, $3, $4)`,
		todoID, "Welcome to your Kanban board", "Drag cards across columns by moving them - add your own board to get started.", 0,
	); err != nil {
		return fmt.Errorf("seed card: %w", err)
	}

	log.Println("seeded starter board")
	return nil
}
