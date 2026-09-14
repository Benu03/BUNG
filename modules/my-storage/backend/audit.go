package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

// clientIP prefers the value nginx forwards (see /nginx/auth-common.conf),
// falling back to the raw connection.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

// writeAudit records one entry into the shared, cross-module audit trail
// (audit.activity_log - see app-maintenance's migrate.go, which bootstraps
// it). Same fully-qualified-table-name pattern used by every module that
// writes to it; best-effort, never fails the request that triggered it.
func writeAudit(db *sql.DB, r *http.Request, action, entityType, entityID string, detail map[string]any) {
	var detailJSON []byte
	if detail != nil {
		var err error
		detailJSON, err = json.Marshal(detail)
		if err != nil {
			log.Printf("audit log write failed: %v", err)
			return
		}
	}
	_, err := db.Exec(
		`INSERT INTO audit.activity_log (actor_user_id, actor_username, module_code, action, entity_type, entity_id, detail, ip_address)
		 VALUES (NULLIF($1, '')::uuid, $2, 'my-storage', $3, $4, $5, $6, $7)`,
		currentUserID(r), r.Header.Get("X-Username"), action, entityType, entityID, detailJSON, clientIP(r),
	)
	if err != nil {
		log.Printf("audit log write failed: %v", err)
	}
}
