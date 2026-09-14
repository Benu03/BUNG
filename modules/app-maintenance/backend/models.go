package main

import "time"

// User represents an application user.
type User struct {
	ID                string    `json:"id"`
	Username          string    `json:"username"`
	FullName          string    `json:"fullName"`
	Email             string    `json:"email"`
	IsActive          bool      `json:"isActive"`
	RoleIDs           []string  `json:"roleIds"`
	PasswordChangedAt time.Time `json:"passwordChangedAt"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
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

// Role belongs to exactly one module and is assigned to users. A user can
// hold several roles within the same module (e.g. "Editor" + "Approver"
// both in Kanban) - that's just two rows in user_roles pointing at two
// roles that share a module_id, nothing special needed in this struct.
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ModuleID    string    `json:"moduleId"`
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

// changePasswordRequest is the payload for POST /auth/change-password -
// self-service, requires the current password (unlike the admin-only
// "Set Password" action in the Users tab, which doesn't).
type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// forgotPasswordRequest is the payload for POST /auth/forgot-password.
// Identifier can be a username or an email.
type forgotPasswordRequest struct {
	Identifier string `json:"identifier"`
}

// resetPasswordRequest is the payload for POST /auth/reset-password.
type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

// AuthUser is what login/me hand back to the frontend: the user plus the
// modules their roles grant access to (what the portal renders), and
// whether their password has aged past site_settings.password_expiry_days
// (the portal blocks access to everything but App Maintenance until
// they change it - see verify() in auth.go).
type AuthUser struct {
	ID                 string    `json:"id"`
	Username           string    `json:"username"`
	FullName           string    `json:"fullName"`
	Email              string    `json:"email"`
	Modules            []*Module `json:"modules"`
	MustChangePassword bool      `json:"mustChangePassword"`
}

// SiteSettings is the platform's general settings: content shown on the
// portal's public landing page, and platform-wide policy.
type SiteSettings struct {
	SiteName           string    `json:"siteName"`
	Tagline            string    `json:"tagline"`
	Announcement       string    `json:"announcement"`
	PasswordExpiryDays int       `json:"passwordExpiryDays"`
	UpdatedAt          time.Time `json:"updatedAt"`
}
