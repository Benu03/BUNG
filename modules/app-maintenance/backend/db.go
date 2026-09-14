package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// openDB opens a connection pool to the shared Postgres instance, scoped to
// this module's own schema via the `search_path` connection parameter.
// Every module connects to the same database (DB_NAME) but lives in its own
// schema (DB_SCHEMA), so modules never collide and a new module never has
// to touch another module's tables.
func openDB() *sql.DB {
	host := getenv("DB_HOST", "localhost")
	port := getenv("DB_PORT", "5432")
	user := getenv("DB_USER", "bung")
	pass := getenv("DB_PASSWORD", "bung")
	name := getenv("DB_NAME", "bung")
	schema := getenv("DB_SCHEMA", "app_maintenance")
	sslmode := getenv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&search_path=%s",
		url.QueryEscape(user), url.QueryEscape(pass), host, port, name, sslmode, url.QueryEscape(schema+",public"),
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := waitForDB(ctx, db); err != nil {
		log.Fatalf("db not ready: %v", err)
	}

	if err := migrate(db, schema); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	return db
}

// waitForDB retries pinging the database until it succeeds or ctx expires,
// so the backend can start cleanly even if postgres is still booting.
func waitForDB(ctx context.Context, db *sql.DB) error {
	for {
		if err := db.PingContext(ctx); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
