package main

import (
	"database/sql"
	"log"
)

// notify pushes one row into notifications.inbox (a different module's
// schema) - same cross-schema pattern as every other module's notify.go
// (calendar, kanban, ticketing): a direct fully-qualified INSERT, no HTTP
// call and no shared Go library between modules. Best-effort: a failure
// here never fails the request that triggered it.
func notify(db *sql.DB, recipientID, moduleCode, notifType, title, body, link string) {
	if recipientID == "" {
		return
	}
	_, err := db.Exec(
		`INSERT INTO notifications.inbox (recipient_id, module_code, type, title, body, link)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		recipientID, moduleCode, notifType, title, body, link,
	)
	if err != nil {
		log.Printf("notify failed: %v", err)
	}
}
