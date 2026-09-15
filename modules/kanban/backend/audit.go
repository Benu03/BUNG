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
// (audit.activity_log, its own Postgres schema - see app-maintenance's
// migrate.go, which bootstraps it). Any module can write to it via this
// same fully-qualified-table-name pattern; no shared Go code or API call
// needed between modules. Best-effort: a logging failure never fails the
// request that triggered it.
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
		`INSERT INTO audit.activity_log (actor_user_id, actor_username, module_code, action, entity_type, entity_id, detail, ip_address, request_id)
		 VALUES (NULLIF($1, '')::uuid, $2, 'kanban', $3, $4, $5, $6, $7, $8)`,
		currentUserID(r), r.Header.Get("X-Username"), action, entityType, entityID, detailJSON, clientIP(r), r.Header.Get("X-Request-Id"),
	)
	if err != nil {
		log.Printf("audit log write failed: %v", err)
	}
}
