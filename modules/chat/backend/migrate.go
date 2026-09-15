package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// schemaSQL creates this module's tables. It relies on the connection's
// search_path (set in openDB) already pointing at this module's schema, so
// table names here are unqualified on purpose. No FK to
// app_maintenance.users anywhere here - modules stay loosely coupled at
// the schema level, same reasoning as every other module.
const schemaSQL = `
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

-- One row per direction of a friend request. Once accepted, either user
-- messaging the other is allowed - see store.go's requireFriends.
CREATE TABLE IF NOT EXISTS friend_requests (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	from_user_id UUID NOT NULL,
	to_user_id UUID NOT NULL,
	status TEXT NOT NULL DEFAULT 'pending', -- pending | accepted | declined
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (from_user_id, to_user_id)
);

CREATE INDEX IF NOT EXISTS friend_requests_to_idx ON friend_requests (to_user_id, status);
CREATE INDEX IF NOT EXISTS friend_requests_from_idx ON friend_requests (from_user_id, status);

-- One 1:1 conversation per friend pair, created lazily on first message.
-- user_a_id/user_b_id are always stored with user_a_id < user_b_id (as
-- plain text comparison) so the pair is unique regardless of who started
-- it - see store.go's conversationKey.
CREATE TABLE IF NOT EXISTS conversations (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	user_a_id UUID NOT NULL,
	user_b_id UUID NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (user_a_id, user_b_id)
);

CREATE TABLE IF NOT EXISTS messages (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
	sender_id UUID NOT NULL,
	body TEXT NOT NULL DEFAULT '',
	attachment_filename TEXT,
	attachment_content_type TEXT,
	attachment_size BIGINT,
	attachment_storage_path TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	read_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS messages_conversation_idx ON messages (conversation_id, created_at);

-- Admin-only announcements, visible to every user regardless of friend
-- status - a separate concept from 1:1 messages on purpose (see
-- handlers.go's createBroadcast, gated by app-maintenance access).
CREATE TABLE IF NOT EXISTS broadcasts (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	sender_id UUID NOT NULL,
	body TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

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

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
