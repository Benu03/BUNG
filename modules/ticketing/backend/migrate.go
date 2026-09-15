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

-- Tickets are a shared helpdesk queue (see the comment on the Ticket
-- struct in models.go) - requester_id/assignee_id have no FK to
-- app_maintenance.users (modules stay loosely coupled at the schema
-- level, same note as every other module's migrate.go).
CREATE TABLE IF NOT EXISTS tickets (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	requester_id UUID NOT NULL,
	assignee_id UUID,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'open',
	priority TEXT NOT NULL DEFAULT 'medium',
	category TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS tickets_status_idx ON tickets (status);
CREATE INDEX IF NOT EXISTS tickets_requester_idx ON tickets (requester_id);
CREATE INDEX IF NOT EXISTS tickets_assignee_idx ON tickets (assignee_id);

CREATE TABLE IF NOT EXISTS ticket_comments (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
	author_id UUID NOT NULL,
	body TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ticket_comments_ticket_idx ON ticket_comments (ticket_id, created_at);
`

// migrate creates this module's schema (if missing) and its tables. No
// seed data - an empty ticket queue is a perfectly normal starting state.
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
