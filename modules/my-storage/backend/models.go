package main

import "time"

// Folder is a directory a user has created to organize their files.
// ParentID is empty/"" for a top-level ("root") folder.
type Folder struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"ownerId"`
	ParentID  string    `json:"parentId,omitempty"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// FileMeta is one uploaded file's metadata. The bytes themselves live on
// disk (see storage.go), under StoragePath, inside the module's own Docker
// volume - only this row is in Postgres. FolderID is empty/"" for a file
// at the root (not inside any folder).
type FileMeta struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"ownerId"`
	FolderID    string    `json:"folderId,omitempty"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	StoragePath string    `json:"-"` // internal only, never serialized to clients
	CreatedAt   time.Time `json:"createdAt"`
}

// BreadcrumbEntry is one folder in the path from root down to (and
// including) the current folder.
type BreadcrumbEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// BrowseResponse is what GET /browse returns: everything needed to render
// one folder's contents in a single request.
type BrowseResponse struct {
	Folder     *Folder           `json:"folder"` // nil at root
	Breadcrumb []BreadcrumbEntry `json:"breadcrumb"`
	Folders    []*Folder         `json:"folders"`
	Files      []*FileMeta       `json:"files"`
}

type createFolderRequest struct {
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
}
