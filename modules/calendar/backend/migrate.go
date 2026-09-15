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

CREATE TABLE IF NOT EXISTS events (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	owner_id UUID NOT NULL,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	location TEXT NOT NULL DEFAULT '',
	start_at TIMESTAMPTZ NOT NULL,
	end_at TIMESTAMPTZ NOT NULL,
	all_day BOOLEAN NOT NULL DEFAULT false,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS events_owner_start_idx ON events (owner_id, start_at);

-- Invited attendees, added by the event's owner - lets an event "invite
-- anyone" (see store.go's Invite/RemoveAttendee): an invited user sees the
-- event in their own calendar too (List includes owned OR invited-to
-- events), and gets a notification. Only the owner can invite/remove
-- attendees or edit/delete the event; attendees can view but not edit -
-- same asymmetry as kanban's board owner vs member.
CREATE TABLE IF NOT EXISTS event_attendees (
	event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
	user_id UUID NOT NULL,
	invited_by UUID NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (event_id, user_id)
);

CREATE INDEX IF NOT EXISTS event_attendees_user_idx ON event_attendees (user_id);
`

// migrate creates this module's schema (if missing) and its tables. No seed
// data - an empty calendar is a perfectly normal starting state.
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
