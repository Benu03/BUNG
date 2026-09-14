package main

import "time"

// User represents an application user.
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	FullName  string    `json:"fullName"`
	Email     string    `json:"email"`
	IsActive  bool      `json:"isActive"`
	RoleIDs   []string  `json:"roleIds"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Module represents a registered application module (e.g. hr, finance,
// app-maintenance itself). Roles are granted access to one or more modules.
type Module struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"` // matches the folder name under /modules and the nginx path prefix
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Role groups a set of module permissions and is assigned to users.
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ModuleIDs   []string  `json:"moduleIds"` // modules this role can access
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
