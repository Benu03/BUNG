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

-- Each role belongs to exactly one module (not a many-to-many like an
-- earlier version of this schema had via a role_modules table) - so
-- assigning a user a role is inherently "grant them this role IN this
-- module", and a user can hold several roles within the same module (see
-- user_roles below - nothing module-specific needed there, since each
-- role_id already implies a single module_id).
CREATE TABLE IF NOT EXISTS roles (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (module_id, name)
);

CREATE TABLE IF NOT EXISTS users (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	username TEXT UNIQUE NOT NULL,
	full_name TEXT NOT NULL DEFAULT '',
	email TEXT UNIQUE NOT NULL DEFAULT '',
	password_hash TEXT NOT NULL DEFAULT '',
	password_changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	is_active BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_roles (
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
	PRIMARY KEY (user_id, role_id)
);

-- Single-row general settings: content shown on the portal's public landing
-- page (site name, tagline, announcement) plus platform-wide policy
-- (password_expiry_days). Managed from the Settings tab.
CREATE TABLE IF NOT EXISTS site_settings (
	id TEXT PRIMARY KEY DEFAULT 'default',
	site_name TEXT NOT NULL DEFAULT 'BUNG',
	tagline TEXT NOT NULL DEFAULT '',
	announcement TEXT NOT NULL DEFAULT '',
	password_expiry_days INT NOT NULL DEFAULT 60, -- 0 = never expire
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Forgot-password tokens. Only a hash of the token is stored (same
-- reasoning as password_hash) - a DB leak alone can't be used to reset
-- anyone's password. One-time use (used_at) and short-lived (expires_at).
CREATE TABLE IF NOT EXISTS password_resets (
	id UUID PRIMARY KEY DEFAULT public.gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	token_hash TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL,
	used_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS password_resets_token_hash_idx ON password_resets (token_hash);
`

// alterSQL patches tables that already existed from an earlier version of
// this schema (CREATE TABLE IF NOT EXISTS above is a no-op once a table
// exists, so a new column needs its own idempotent statement here instead).
// This is the upgrade path for existing deployments: add a line, redeploy,
// done - no separate migration tool/runner needed for changes this simple.
const alterSQL = `
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx ON users (email);
ALTER TABLE site_settings ADD COLUMN IF NOT EXISTS password_expiry_days INT NOT NULL DEFAULT 60;
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

	var userID string

	// Seed one registry row per module that ships with this stack (drives
	// the portal's module list), plus one "Administrator" role scoped to
	// each - when you add a new module folder, add it here too (or via the
	// Modules + Roles tabs) so it shows up on the portal and has a role to
	// assign.
	knownModules := []struct{ code, name, description string }{
		{"app-maintenance", "App Maintenance", "User, module and role administration"},
		{"kanban", "Kanban", "Boards, columns and cards"},
	}

	adminRoleIDs := make([]string, 0, len(knownModules))
	for _, m := range knownModules {
		var moduleID string
		if err := db.QueryRow(
			`INSERT INTO modules (code, name, description) VALUES ($1, $2, $3) RETURNING id`,
			m.code, m.name, m.description,
		).Scan(&moduleID); err != nil {
			return fmt.Errorf("seed module %s: %w", m.code, err)
		}

		var roleID string
		if err := db.QueryRow(
			`INSERT INTO roles (module_id, name, description) VALUES ($1, $2, $3) RETURNING id`,
			moduleID, "Administrator", "Full access to this module",
		).Scan(&roleID); err != nil {
			return fmt.Errorf("seed role for %s: %w", m.code, err)
		}
		adminRoleIDs = append(adminRoleIDs, roleID)
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

	for _, roleID := range adminRoleIDs {
		if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, userID, roleID); err != nil {
			return fmt.Errorf("seed user_roles: %w", err)
		}
	}

	log.Println("seeded initial admin user with Administrator role in every module")
	return nil
}
