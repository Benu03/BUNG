package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// schemaSQL creates this module's tables. It relies on the connection's
// search_path (set in openDB) already pointing at this module's schema, so
// table names here are unqualified on purpose.
const schemaSQL = `
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

-- Files are private: every query is scoped to owner_id = the requesting
-- user (see store.go) - there's no sharing in this module (unlike kanban's
-- boards). owner_id has no FK to app_maintenance.users - modules stay
-- loosely coupled at the schema level (see the same note in kanban's
-- migrate.go).
CREATE TABLE IF NOT EXISTS files (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	owner_id UUID NOT NULL,
	filename TEXT NOT NULL,
	content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
	size BIGINT NOT NULL DEFAULT 0,
	storage_path TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS files_owner_id_idx ON files (owner_id);
`

// migrate creates this module's schema (if missing) and its tables. No
// seed data - an empty file list is a perfectly normal starting state.
func migrate(db *sql.DB, schema string) error {
	createSchema := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s;`, quoteIdent(schema))
	if _, err := db.Exec(createSchema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}
	return nil
}

// quoteIdent does minimal safe quoting for identifiers that come from our
// own env config (module schema names), not from user input.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
