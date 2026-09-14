package main

import "time"

// FileMeta is one uploaded file's metadata. The bytes themselves live on
// disk (see storage.go), under StoragePath, inside the module's own Docker
// volume - only this row is in Postgres.
type FileMeta struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"ownerId"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	StoragePath string    `json:"-"` // internal only, never serialized to clients
	CreatedAt   time.Time `json:"createdAt"`
}
