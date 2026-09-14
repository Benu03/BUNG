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

CREATE TABLE IF NOT EXISTS modules (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	code TEXT UNIQUE NOT NULL,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	is_active BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS roles (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	name TEXT UNIQUE NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS role_modules (
	role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
	module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
	PRIMARY KEY (role_id, module_id)
);

CREATE TABLE IF NOT EXISTS users (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	username TEXT UNIQUE NOT NULL,
	full_name TEXT NOT NULL DEFAULT '',
	email TEXT NOT NULL DEFAULT '',
	password_hash TEXT NOT NULL DEFAULT '',
	is_active BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_roles (
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
	PRIMARY KEY (user_id, role_id)
);

-- Single-row config table for content shown on the portal's public landing
-- page (site name, tagline, an optional announcement banner). Managed from
-- the Settings tab.
CREATE TABLE IF NOT EXISTS site_settings (
	id TEXT PRIMARY KEY DEFAULT 'default',
	site_name TEXT NOT NULL DEFAULT 'BUNG',
	tagline TEXT NOT NULL DEFAULT '',
	announcement TEXT NOT NULL DEFAULT '',
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

// alterSQL patches tables that already existed from an earlier version of
// this schema (CREATE TABLE IF NOT EXISTS above is a no-op once a table
// exists, so a new column needs its own idempotent statement here instead).
// This is the upgrade path for existing deployments: add a line, redeploy,
// done - no separate migration tool/runner needed for changes this simple.
const alterSQL = `
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';
`

// auditSchemaSQL creates the shared, cross-module audit trail. It lives in
// its own schema (not any module's own), fully qualified in every query
// (audit.activity_log) so it works regardless of the connection's
// search_path - any module's backend could write to it the same way.
// app-maintenance just happens to be the one that bootstraps it, being the
// first/core module.
const auditSchemaSQL = `
CREATE SCHEMA IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.activity_log (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	actor_user_id UUID,
	actor_username TEXT NOT NULL DEFAULT '',
	module_code TEXT NOT NULL,
	action TEXT NOT NULL,
	entity_type TEXT NOT NULL DEFAULT '',
	entity_id TEXT NOT NULL DEFAULT '',
	detail JSONB,
	ip_address TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS activity_log_occurred_at_idx ON audit.activity_log (occurred_at DESC);
`

// migrate creates this module's schema (if missing) and its tables, patches
// existing tables forward (see alterSQL), then seeds initial data so the UI
// is usable right away.
func migrate(db *sql.DB, schema string) error {
	createSchema := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s;`, quoteIdent(schema))
	if _, err := db.Exec(createSchema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}
	if _, err := db.Exec(auditSchemaSQL); err != nil {
		return fmt.Errorf("create audit schema: %w", err)
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
	if _, err := db.Exec(`INSERT INTO site_settings (id, site_name, tagline) VALUES ('default', 'BUNG', 'Sign in to access your modules') ON CONFLICT (id) DO NOTHING`); err != nil {
		return fmt.Errorf("seed site_settings: %w", err)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var roleID, userID string

	// Seed one registry row per module that ships with this stack. This is
	// what drives the portal landing page's module list - when you add a
	// new module folder, also add it here (or via the Modules tab in the
	// UI) so it shows up on the portal without a manual step.
	knownModules := []struct{ code, name, description string }{
		{"app-maintenance", "App Maintenance", "User, module and role administration"},
		{"kanban", "Kanban", "Boards, columns and cards"},
	}

	moduleIDs := make([]string, 0, len(knownModules))
	for _, m := range knownModules {
		var id string
		if err := db.QueryRow(
			`INSERT INTO modules (code, name, description) VALUES ($1, $2, $3) RETURNING id`,
			m.code, m.name, m.description,
		).Scan(&id); err != nil {
			return fmt.Errorf("seed module %s: %w", m.code, err)
		}
		moduleIDs = append(moduleIDs, id)
	}

	if err := db.QueryRow(
		`INSERT INTO roles (name, description) VALUES ($1, $2) RETURNING id`,
		"Administrator", "Full access to all modules",
	).Scan(&roleID); err != nil {
		return fmt.Errorf("seed role: %w", err)
	}

	for _, moduleID := range moduleIDs {
		if _, err := db.Exec(`INSERT INTO role_modules (role_id, module_id) VALUES ($1, $2)`, roleID, moduleID); err != nil {
			return fmt.Errorf("seed role_modules: %w", err)
		}
	}

	// Default admin password comes from env (see .env's SEED_ADMIN_PASSWORD)
	// rather than being hardcoded here. Change it after first login via the
	// Users tab's "Set Password" action.
	hash, err := hashPassword(getenv("SEED_ADMIN_PASSWORD", "admin123"))
	if err != nil {
		return fmt.Errorf("seed password hash: %w", err)
	}

	if err := db.QueryRow(
		`INSERT INTO users (username, full_name, email, password_hash) VALUES ($1, $2, $3, $4) RETURNING id`,
		"admin", "System Administrator", "admin@example.com", hash,
	).Scan(&userID); err != nil {
		return fmt.Errorf("seed user: %w", err)
	}

	if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, userID, roleID); err != nil {
		return fmt.Errorf("seed user_roles: %w", err)
	}

	log.Println("seeded initial admin user/role/module")
	return nil
}
