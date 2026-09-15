package main

import (
	"database/sql"
	"log"
)

// notify creates a notification for recipientID in the shared
// notifications.inbox table (bootstrapped by the notifications module,
// see its migrate.go) - a fully-qualified cross-schema INSERT, same
// pattern as writing to audit.activity_log. Best-effort: a failure here
// (e.g. the notifications schema not existing yet on a very first boot)
// never fails the request that triggered it.
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
