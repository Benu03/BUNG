package main

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound is returned by Store methods when the requested row doesn't
// exist, so handlers can tell that apart from a real (5xx) failure.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when a create/update would violate a uniqueness
// constraint (username or email already taken).
var ErrConflict = errors.New("conflict")

// isUniqueViolation maps a Postgres unique_violation (SQLSTATE 23505) onto
// ErrConflict, so handlers can return 409 instead of a generic 500.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Store is backed by Postgres, scoped to this module's schema via the
// connection's search_path (see db.go).
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ---- users ----

func (s *Store) ListUsers() ([]*User, error) {
	rows, err := s.db.Query(`SELECT id, username, full_name, email, is_active, password_changed_at, created_at, updated_at FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.FullName, &u.Email, &u.IsActive, &u.PasswordChangedAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, u := range users {
		roleIDs, err := s.userRoleIDs(u.ID)
		if err != nil {
			return nil, err
		}
		u.RoleIDs = roleIDs
	}
	return users, nil
}

func (s *Store) GetUser(id string) (*User, error) {
	var u User
	err := s.db.QueryRow(`SELECT id, username, full_name, email, is_active, password_changed_at, created_at, updated_at FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Username, &u.FullName, &u.Email, &u.IsActive, &u.PasswordChangedAt, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	roleIDs, err := s.userRoleIDs(u.ID)
	if err != nil {
		return nil, err
	}
	u.RoleIDs = roleIDs
	return &u, nil
}

func (s *Store) CreateUser(in *User, passwordHash string) (*User, error) {
	err := s.db.QueryRow(
		`INSERT INTO users (username, full_name, email, is_active, password_hash) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, password_changed_at, created_at, updated_at`,
		in.Username, in.FullName, in.Email, in.IsActive, passwordHash,
	).Scan(&in.ID, &in.PasswordChangedAt, &in.CreatedAt, &in.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	in.RoleIDs = []string{}
	return in, nil
}

// GetUserByUsername is used by login - it also returns the password hash,
// which GetUser deliberately never exposes.
func (s *Store) GetUserByUsername(username string) (*User, string, error) {
	return s.getUserByColumn("username", username)
}

// GetUserByIdentifier is used by forgot-password - identifier can be either
// a username or an email.
func (s *Store) GetUserByIdentifier(identifier string) (*User, string, error) {
	return s.getUserByColumn("username_or_email", identifier)
}

func (s *Store) getUserByColumn(mode, value string) (*User, string, error) {
	query := `SELECT id, username, full_name, email, is_active, password_hash, password_changed_at, created_at, updated_at FROM users WHERE `
	if mode == "username" {
		query += `username = $1`
	} else {
		query += `username = $1 OR email = $1`
	}
	var u User
	var passwordHash string
	err := s.db.QueryRow(query, value).
		Scan(&u.ID, &u.Username, &u.FullName, &u.Email, &u.IsActive, &passwordHash, &u.PasswordChangedAt, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	return &u, passwordHash, nil
}

// SetUserPassword replaces a user's password hash and resets the
// password-age clock (password_changed_at) - used by the admin "Set
// Password" action, self-service change-password, and forgot-password reset.
func (s *Store) SetUserPassword(id string, passwordHash string) error {
	res, err := s.db.Exec(`UPDATE users SET password_hash = $1, password_changed_at = now(), updated_at = now() WHERE id = $2`, passwordHash, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// UserModules returns the distinct, active modules a user can access via
// any of their active roles (each role belongs to exactly one module) -
// this is what gets baked into the session JWT at login, and what the
// portal renders. Both the role AND the module must be active - a user
// keeps their role assignment either way (deactivating is reversible,
// deleting isn't), it just stops granting access while inactive.
func (s *Store) UserModules(userID string) ([]*Module, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT m.id, m.code, m.name, m.description, m.is_active, m.created_at, m.updated_at
		FROM modules m
		JOIN roles r ON r.module_id = m.id
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1 AND m.is_active = true AND r.is_active = true
		ORDER BY m.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	modules := []*Module{}
	for rows.Next() {
		var m Module
		if err := rows.Scan(&m.ID, &m.Code, &m.Name, &m.Description, &m.IsActive, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		modules = append(modules, &m)
	}
	return modules, rows.Err()
}

func (s *Store) UpdateUser(id string, in *User) (*User, error) {
	res, err := s.db.Exec(
		`UPDATE users SET username = $1, full_name = $2, email = $3, is_active = $4, updated_at = now() WHERE id = $5`,
		in.Username, in.FullName, in.Email, in.IsActive, id,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.GetUser(id)
}

func (s *Store) DeleteUser(id string) error {
	res, err := s.db.Exec(`DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetUserRoles(id string, roleIDs []string) (*User, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var exists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, id).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	if _, err := tx.Exec(`DELETE FROM user_roles WHERE user_id = $1`, id); err != nil {
		return nil, err
	}
	for _, roleID := range roleIDs {
		if _, err := tx.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, roleID); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(`UPDATE users SET updated_at = now() WHERE id = $1`, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetUser(id)
}

func (s *Store) userRoleIDs(userID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT role_id FROM user_roles WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ---- roles ----

func (s *Store) ListRoles() ([]*Role, error) {
	rows, err := s.db.Query(`SELECT id, module_id, name, description, is_active, created_at, updated_at FROM roles ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := []*Role{}
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.ModuleID, &r.Name, &r.Description, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, &r)
	}
	return roles, rows.Err()
}

func (s *Store) GetRole(id string) (*Role, error) {
	var r Role
	err := s.db.QueryRow(`SELECT id, module_id, name, description, is_active, created_at, updated_at FROM roles WHERE id = $1`, id).
		Scan(&r.ID, &r.ModuleID, &r.Name, &r.Description, &r.IsActive, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) CreateRole(in *Role) (*Role, error) {
	err := s.db.QueryRow(
		`INSERT INTO roles (module_id, name, description, is_active) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		in.ModuleID, in.Name, in.Description, in.IsActive,
	).Scan(&in.ID, &in.CreatedAt, &in.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return in, nil
}

func (s *Store) UpdateRole(id string, in *Role) (*Role, error) {
	res, err := s.db.Exec(
		`UPDATE roles SET module_id = $1, name = $2, description = $3, is_active = $4, updated_at = now() WHERE id = $5`,
		in.ModuleID, in.Name, in.Description, in.IsActive, id,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.GetRole(id)
}

func (s *Store) DeleteRole(id string) error {
	res, err := s.db.Exec(`DELETE FROM roles WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- modules ----

func (s *Store) ListModules() ([]*Module, error) {
	rows, err := s.db.Query(`SELECT id, code, name, description, is_active, created_at, updated_at FROM modules ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []*Module
	for rows.Next() {
		var m Module
		if err := rows.Scan(&m.ID, &m.Code, &m.Name, &m.Description, &m.IsActive, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		modules = append(modules, &m)
	}
	return modules, rows.Err()
}

func (s *Store) GetModule(id string) (*Module, error) {
	var m Module
	err := s.db.QueryRow(`SELECT id, code, name, description, is_active, created_at, updated_at FROM modules WHERE id = $1`, id).
		Scan(&m.ID, &m.Code, &m.Name, &m.Description, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Store) CreateModule(in *Module) (*Module, error) {
	err := s.db.QueryRow(
		`INSERT INTO modules (code, name, description, is_active) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		in.Code, in.Name, in.Description, in.IsActive,
	).Scan(&in.ID, &in.CreatedAt, &in.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return in, nil
}

func (s *Store) UpdateModule(id string, in *Module) (*Module, error) {
	res, err := s.db.Exec(
		`UPDATE modules SET code = $1, name = $2, description = $3, is_active = $4, updated_at = now() WHERE id = $5`,
		in.Code, in.Name, in.Description, in.IsActive, id,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.GetModule(id)
}

func (s *Store) DeleteModule(id string) error {
	res, err := s.db.Exec(`DELETE FROM modules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- site settings ----

func (s *Store) GetSiteSettings() (*SiteSettings, error) {
	var st SiteSettings
	err := s.db.QueryRow(`SELECT site_name, tagline, announcement, password_expiry_days, updated_at FROM site_settings WHERE id = 'default'`).
		Scan(&st.SiteName, &st.Tagline, &st.Announcement, &st.PasswordExpiryDays, &st.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Store) UpdateSiteSettings(in *SiteSettings) (*SiteSettings, error) {
	_, err := s.db.Exec(
		`UPDATE site_settings SET site_name = $1, tagline = $2, announcement = $3, password_expiry_days = $4, updated_at = now() WHERE id = 'default'`,
		in.SiteName, in.Tagline, in.Announcement, in.PasswordExpiryDays,
	)
	if err != nil {
		return nil, err
	}
	return s.GetSiteSettings()
}

// ---- password resets ----

// CreatePasswordReset stores a (hashed) reset token for userID, valid until
// expiresAt.
func (s *Store) CreatePasswordReset(userID, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

// ConsumePasswordReset atomically validates a (hashed) reset token - must
// exist, be unexpired, and not already used - marks it used, and returns
// the user it belongs to. Using it twice (e.g. a replayed request) fails
// the second time.
func (s *Store) ConsumePasswordReset(tokenHash string) (string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var id, userID string
	err = tx.QueryRow(
		`SELECT id, user_id FROM password_resets
		 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()`,
		tokenHash,
	).Scan(&id, &userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}

	if _, err := tx.Exec(`UPDATE password_resets SET used_at = now() WHERE id = $1`, id); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return userID, nil
}
