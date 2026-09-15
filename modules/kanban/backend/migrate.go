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
	owner_id UUID NOT NULL,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Boards are private: only members can see/access one (see store.go's
-- membership checks). user_id deliberately has no FK to app_maintenance's
-- users table - modules stay loosely coupled at the schema level, cross-
-- referenced by ID convention only (see ListAllUsers for the one place
-- this module reads app_maintenance.users directly, same pattern as the
-- audit log).
CREATE TABLE IF NOT EXISTS board_members (
	board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
	user_id UUID NOT NULL,
	role TEXT NOT NULL DEFAULT 'member', -- 'owner' | 'member'
	joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (board_id, user_id)
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

// alterSQL patches tables that already existed from an earlier version of
// this schema (CREATE TABLE IF NOT EXISTS above is a no-op once a table
// exists, so a new column needs its own idempotent statement here instead)
// - same pattern as every other module's migrate.go. assignee_id has no FK
// to app_maintenance.users, same loose-coupling reasoning as board_members.
const alterSQL = `
ALTER TABLE cards ADD COLUMN IF NOT EXISTS assignee_id UUID;
ALTER TABLE cards ADD COLUMN IF NOT EXISTS due_date TIMESTAMPTZ;
ALTER TABLE cards ADD COLUMN IF NOT EXISTS color TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS cards_assignee_idx ON cards (assignee_id);
`

// migrate creates this module's schema (if missing) and its tables, patches
// existing tables forward, then seeds a starter board so the UI is usable
// right away.
func migrate(db *sql.DB, schema string) error {
	createSchema := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s;`, quoteIdent(schema))
	if _, err := db.Exec(createSchema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}
	if _, err := db.Exec(alterSQL); err != nil {
		return fmt.Errorf("alter tables: %w", err)
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

	// The starter board needs an owner - reuse app-maintenance's seeded
	// "admin" user (cross-schema read, see the comment on board_members).
	// If app-maintenance hasn't seeded yet (a startup race between the two
	// backends), skip for now; this re-runs on every restart until it
	// succeeds, since `count(*) FROM boards` stays 0 either way.
	var ownerID string
	if err := db.QueryRow(`SELECT id FROM app_maintenance.users WHERE username = 'admin'`).Scan(&ownerID); err != nil {
		log.Printf("skip seeding starter board: admin user not resolvable yet (%v)", err)
		return nil
	}

	var boardID string
	if err := db.QueryRow(
		`INSERT INTO boards (owner_id, name, description) VALUES ($1, $2, $3) RETURNING id`,
		ownerID, "Getting Started", "Default starter board",
	).Scan(&boardID); err != nil {
		return fmt.Errorf("seed board: %w", err)
	}

	if _, err := db.Exec(
		`INSERT INTO board_members (board_id, user_id, role) VALUES ($1, $2, 'owner')`,
		boardID, ownerID,
	); err != nil {
		return fmt.Errorf("seed board_members: %w", err)
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
		todoID, "Welcome to your Kanban board", "Only board members can see this - add teammates from the Members panel.", 0,
	); err != nil {
		return fmt.Errorf("seed card: %w", err)
	}

	log.Println("seeded starter board")
	return nil
}
