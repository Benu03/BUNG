package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// schemaSQL creates this module's tables. It relies on the connection's
// search_path (set in openDB) already pointing at this module's schema, so
// table names here are unqualified on purpose.
//
// inbox is a shared, cross-module table (like audit.activity_log) - any
// module's backend can INSERT into notifications.inbox directly (fully
// qualified, regardless of its own search_path) to notify a user. This
// module only owns the read side (list/mark-read API + the bell dropdown
// UI duplicated into every other frontend).
const schemaSQL = `
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

CREATE TABLE IF NOT EXISTS inbox (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	recipient_id UUID NOT NULL,
	module_code TEXT NOT NULL,
	type TEXT NOT NULL,
	title TEXT NOT NULL,
	body TEXT NOT NULL DEFAULT '',
	link TEXT NOT NULL DEFAULT '',
	read_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS inbox_recipient_created_idx ON inbox (recipient_id, created_at DESC);
CREATE INDEX IF NOT EXISTS inbox_recipient_unread_idx ON inbox (recipient_id) WHERE read_at IS NULL;
`

// migrate creates this module's schema (if missing) and its tables.
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
