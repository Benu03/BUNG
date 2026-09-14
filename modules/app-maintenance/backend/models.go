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

// createUserRequest is the payload for POST /users - separate from User so
// a plaintext Password never round-trips through a User value (which is
// also what every other handler returns to clients).
type createUserRequest struct {
	Username string `json:"username"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	IsActive bool   `json:"isActive"`
	Password string `json:"password"`
}

// loginRequest is the payload for POST /auth/login.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthUser is what login/me hand back to the frontend: the user plus the
// modules their roles grant access to (what the portal renders).
type AuthUser struct {
	ID       string    `json:"id"`
	Username string    `json:"username"`
	FullName string    `json:"fullName"`
	Email    string    `json:"email"`
	Modules  []*Module `json:"modules"`
}

// SiteSettings controls content shown on the portal's public landing page.
type SiteSettings struct {
	SiteName     string    `json:"siteName"`
	Tagline      string    `json:"tagline"`
	Announcement string    `json:"announcement"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
