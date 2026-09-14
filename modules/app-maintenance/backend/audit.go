package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// AuditEntry is one row of the shared, cross-module audit trail. It lives
// in its own Postgres schema ("audit", not any module's own schema - see
// auditSchemaSQL in migrate.go) so any module's backend can write to it via
// the fully-qualified table name (audit.activity_log) without needing
// "audit" in its own search_path.
//
// Only app-maintenance writes to it today (see writeAudit below, used from
// its own handlers), but the schema is deliberately module-agnostic
// (ModuleCode says which module an entry came from) so another module can
// start writing entries the same way later - see README.
type AuditEntry struct {
	ID            string         `json:"id"`
	OccurredAt    time.Time      `json:"occurredAt"`
	ActorUserID   string         `json:"actorUserId"`
	ActorUsername string         `json:"actorUsername"`
	ModuleCode    string         `json:"moduleCode"`
	Action        string         `json:"action"`
	EntityType    string         `json:"entityType"`
	EntityID      string         `json:"entityId"`
	Detail        map[string]any `json:"detail,omitempty"`
	IPAddress     string         `json:"ipAddress"`
}

func (s *Store) InsertAuditLog(e *AuditEntry) error {
	var detailJSON []byte
	if e.Detail != nil {
		var err error
		detailJSON, err = json.Marshal(e.Detail)
		if err != nil {
			return err
		}
	}
	_, err := s.db.Exec(
		`INSERT INTO audit.activity_log (actor_user_id, actor_username, module_code, action, entity_type, entity_id, detail, ip_address)
		 VALUES (NULLIF($1, '')::uuid, $2, $3, $4, $5, $6, $7, $8)`,
		e.ActorUserID, e.ActorUsername, e.ModuleCode, e.Action, e.EntityType, e.EntityID, detailJSON, e.IPAddress,
	)
	return err
}

func (s *Store) ListAuditLog(limit, offset int) ([]*AuditEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, occurred_at, coalesce(actor_user_id::text, ''), actor_username,
		       module_code, action, entity_type, entity_id, detail, ip_address
		FROM audit.activity_log
		ORDER BY occurred_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []*AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		var detailJSON []byte
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.ActorUserID, &e.ActorUsername,
			&e.ModuleCode, &e.Action, &e.EntityType, &e.EntityID, &detailJSON, &e.IPAddress); err != nil {
			return nil, err
		}
		if len(detailJSON) > 0 {
			_ = json.Unmarshal(detailJSON, &e.Detail)
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}

// clientIP prefers the value nginx forwards (see auth-common.conf and the
// public auth/ location block), falling back to the raw connection.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

// writeAudit is the convenience path used by handlers that run behind
// nginx's auth_request gate, where X-User-Id/X-Username are already
// populated by the gateway - no need to look the actor up again.
// Best-effort: a logging failure is logged itself but never fails the
// request that triggered it.
func writeAudit(s *Store, r *http.Request, action, entityType, entityID string, detail map[string]any) {
	e := &AuditEntry{
		ActorUserID:   r.Header.Get("X-User-Id"),
		ActorUsername: r.Header.Get("X-Username"),
		ModuleCode:    "app-maintenance",
		Action:        action,
		EntityType:    entityType,
		EntityID:      entityID,
		Detail:        detail,
		IPAddress:     clientIP(r),
	}
	if err := s.InsertAuditLog(e); err != nil {
		log.Printf("audit log write failed: %v", err)
	}
}
