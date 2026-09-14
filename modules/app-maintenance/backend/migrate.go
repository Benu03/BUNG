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
`

// migrate creates this module's schema (if missing) and its tables, then
// seeds an initial admin user/role/module so the UI is usable right away.
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

	// Default admin password - change it after first login (Users tab has
	// no self-service change yet, use the "Set Password" action).
	hash, err := hashPassword("admin123")
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
