package main

import (
	"database/sql"
	"errors"
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

// auditAlterSQL patches audit.activity_log forward for deployments where
// it already existed before request_id was added - same "CREATE TABLE IF
// NOT EXISTS is a no-op, so a new column needs its own idempotent
// statement" reasoning as alterSQL below, just for the audit schema
// instead of this module's own.
const auditAlterSQL = `
ALTER TABLE audit.activity_log ADD COLUMN IF NOT EXISTS request_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS activity_log_request_id_idx ON audit.activity_log (request_id);
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
	if _, err := db.Exec(auditAlterSQL); err != nil {
		return fmt.Errorf("alter audit schema: %w", err)
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

// ensureModule inserts a modules row for `code` if one doesn't already
// exist (idempotent - safe to call on every startup, unlike the old "only
// seed once" approach), returning its id and whether this call is what
// created it.
func ensureModule(db *sql.DB, code, name, description string) (id string, created bool, err error) {
	err = db.QueryRow(
		`INSERT INTO modules (code, name, description) VALUES ($1, $2, $3)
		 ON CONFLICT (code) DO NOTHING RETURNING id`,
		code, name, description,
	).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}
	// ON CONFLICT DO NOTHING with no matching RETURNING row -> already existed.
	err = db.QueryRow(`SELECT id FROM modules WHERE code = $1`, code).Scan(&id)
	return id, false, err
}

// ensureAdministratorRole is the same idempotent pattern as ensureModule,
// for the one "Administrator" role every known module gets automatically.
func ensureAdministratorRole(db *sql.DB, moduleID string) (string, error) {
	var id string
	err := db.QueryRow(
		`INSERT INTO roles (module_id, name, description) VALUES ($1, 'Administrator', 'Full access to this module')
		 ON CONFLICT (module_id, name) DO NOTHING RETURNING id`,
		moduleID,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	err = db.QueryRow(`SELECT id FROM roles WHERE module_id = $1 AND name = 'Administrator'`, moduleID).Scan(&id)
	return id, err
}

func seed(db *sql.DB) error {
	if _, err := db.Exec(`INSERT INTO site_settings (id, site_name, tagline) VALUES ('default', 'BUNG', 'Sign in to access your modules') ON CONFLICT (id) DO NOTHING`); err != nil {
		return fmt.Errorf("seed site_settings: %w", err)
	}

	// Registry of modules that ship with this stack (drives the portal's
	// module list), plus one "Administrator" role scoped to each - when you
	// add a new module folder, add it here too (or via the Modules + Roles
	// tabs) so it shows up on the portal and has a role to assign. This
	// runs on every startup (ensureModule/ensureAdministratorRole are
	// idempotent), not just the very first one, so adding a module to an
	// already-running install just needs a redeploy - no destructive schema
	// reset required.
	knownModules := []struct{ code, name, description string }{
		{"app-maintenance", "App Maintenance", "User, module and role administration"},
		{"kanban", "Kanban", "Boards, columns and cards"},
		{"my-storage", "My Storage", "Your private files"},
		{"calendar", "Calendar", "Personal events and schedule"},
		{"ticketing", "Ticketing", "Support tickets and requests"},
	}

	var userCount int
	if err := db.QueryRow(`SELECT count(*) FROM users`).Scan(&userCount); err != nil {
		return err
	}
	firstBoot := userCount == 0

	// Best-effort: only used so a module registered *after* first boot
	// (i.e. added in a later release) is immediately usable by the
	// original seeded admin account too, without a manual Roles tab step.
	var adminUserID string
	if !firstBoot {
		_ = db.QueryRow(`SELECT id FROM users WHERE username = 'admin'`).Scan(&adminUserID)
	}

	bootstrapRoleIDs := make([]string, 0, len(knownModules))
	for _, m := range knownModules {
		moduleID, created, err := ensureModule(db, m.code, m.name, m.description)
		if err != nil {
			return fmt.Errorf("ensure module %s: %w", m.code, err)
		}
		roleID, err := ensureAdministratorRole(db, moduleID)
		if err != nil {
			return fmt.Errorf("ensure role for %s: %w", m.code, err)
		}

		if firstBoot {
			bootstrapRoleIDs = append(bootstrapRoleIDs, roleID)
			continue
		}
		if created && adminUserID != "" {
			if _, err := db.Exec(
				`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				adminUserID, roleID,
			); err != nil {
				return fmt.Errorf("grant new module %s to admin: %w", m.code, err)
			}
			log.Printf("registered new module %q and granted its Administrator role to the admin account", m.code)
		}
	}

	if !firstBoot {
		return nil
	}

	// Default admin password comes from env (see .env's SEED_ADMIN_PASSWORD)
	// rather than being hardcoded here. Change it after first login via the
	// Users tab's "Set Password" action.
	hash, err := hashPassword(getenv("SEED_ADMIN_PASSWORD", "admin123"))
	if err != nil {
		return fmt.Errorf("seed password hash: %w", err)
	}

	var userID string
	if err := db.QueryRow(
		`INSERT INTO users (username, full_name, email, password_hash) VALUES ($1, $2, $3, $4) RETURNING id`,
		"admin", "System Administrator", "admin@example.com", hash,
	).Scan(&userID); err != nil {
		return fmt.Errorf("seed user: %w", err)
	}

	for _, roleID := range bootstrapRoleIDs {
		if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, userID, roleID); err != nil {
			return fmt.Errorf("seed user_roles: %w", err)
		}
	}

	log.Println("seeded initial admin user with Administrator role in every module")
	return nil
}
