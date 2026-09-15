package main

import (
	"database/sql"
	"strconv"
)

// Result is one hit, normalized across every module's own shape so the
// frontend can render one flat list. Link is a best-effort deep link -
// see each searchX function below for which ones actually resolve to a
// specific item (via a query param the target page reads on load) versus
// just the module's root.
type Result struct {
	Type       string `json:"type"` // ticket | event | file | board | user
	ID         string `json:"id"`
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle,omitempty"`
	ModuleCode string `json:"moduleCode"`
	Link       string `json:"link"`
}

const resultLimit = 6

// Store is backed by Postgres - see db.go for why there's no schema of
// its own here.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// GrantedModules returns the set of module codes userID has any role in -
// same join app-maintenance itself uses to decide what shows on the
// portal dashboard. Search results are filtered to this set so a search
// can never leak content from a module the user doesn't have access to,
// even though the search gateway route itself (see nginx.conf) is open to
// any logged-in user like notifications.
func (s *Store) GrantedModules(userID string) (map[string]bool, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT m.code
		FROM app_maintenance.user_roles ur
		JOIN app_maintenance.roles r ON r.id = ur.role_id
		JOIN app_maintenance.modules m ON m.id = r.module_id
		WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	granted := map[string]bool{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		granted[code] = true
	}
	return granted, rows.Err()
}

func (s *Store) SearchTickets(like string) ([]Result, error) {
	rows, err := s.db.Query(`
		SELECT id, title, category, status
		FROM ticketing.tickets
		WHERE title ILIKE $1 OR description ILIKE $1 OR category ILIKE $1
		ORDER BY created_at DESC LIMIT `+strconv.Itoa(resultLimit), like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Result
	for rows.Next() {
		var id, title, category, status string
		if err := rows.Scan(&id, &title, &category, &status); err != nil {
			return nil, err
		}
		subtitle := status
		if category != "" {
			subtitle = category + " - " + status
		}
		out = append(out, Result{
			Type: "ticket", ID: id, Title: title, Subtitle: subtitle,
			ModuleCode: "ticketing", Link: "/ticketing/?ticket=" + id,
		})
	}
	return out, rows.Err()
}

// SearchEvents is scoped to userID (owner or invited attendee) - unlike
// tickets, calendar events are private, so a module grant alone isn't
// enough to see someone else's event.
func (s *Store) SearchEvents(like, userID string) ([]Result, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT e.id, e.title, e.location, e.start_at
		FROM calendar.events e
		LEFT JOIN calendar.event_attendees ea ON ea.event_id = e.id AND ea.user_id = $2
		WHERE (e.owner_id = $2 OR ea.user_id = $2)
		  AND (e.title ILIKE $1 OR e.description ILIKE $1 OR e.location ILIKE $1)
		ORDER BY e.start_at DESC LIMIT `+strconv.Itoa(resultLimit), like, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Result
	for rows.Next() {
		var id, title, location string
		var startAt any
		if err := rows.Scan(&id, &title, &location, &startAt); err != nil {
			return nil, err
		}
		out = append(out, Result{
			Type: "event", ID: id, Title: title, Subtitle: location,
			ModuleCode: "calendar", Link: "/calendar/?event=" + id,
		})
	}
	return out, rows.Err()
}

// SearchFiles is scoped to userID - my-storage is entirely private per
// owner. No deep link into a specific folder (files can be nested
// arbitrarily deep) - just the module root, with the filename as the hint.
func (s *Store) SearchFiles(like, userID string) ([]Result, error) {
	rows, err := s.db.Query(`
		SELECT id, filename, content_type
		FROM my_storage.files
		WHERE owner_id = $2 AND filename ILIKE $1
		ORDER BY created_at DESC LIMIT `+strconv.Itoa(resultLimit), like, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Result
	for rows.Next() {
		var id, filename, contentType string
		if err := rows.Scan(&id, &filename, &contentType); err != nil {
			return nil, err
		}
		out = append(out, Result{
			Type: "file", ID: id, Title: filename, Subtitle: contentType,
			ModuleCode: "my-storage", Link: "/my-storage/",
		})
	}
	return out, rows.Err()
}

// SearchBoards is scoped to userID (board membership) - kanban boards are
// private per member, same reasoning as calendar events.
func (s *Store) SearchBoards(like, userID string) ([]Result, error) {
	rows, err := s.db.Query(`
		SELECT b.id, b.name, b.description
		FROM kanban.boards b
		JOIN kanban.board_members bm ON bm.board_id = b.id
		WHERE bm.user_id = $2 AND (b.name ILIKE $1 OR b.description ILIKE $1)
		ORDER BY b.updated_at DESC LIMIT `+strconv.Itoa(resultLimit), like, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Result
	for rows.Next() {
		var id, name, description string
		if err := rows.Scan(&id, &name, &description); err != nil {
			return nil, err
		}
		out = append(out, Result{
			Type: "board", ID: id, Title: name, Subtitle: description,
			ModuleCode: "kanban", Link: "/kanban/?board=" + id,
		})
	}
	return out, rows.Err()
}

// SearchUsers is not scoped to a single user's own data (app-maintenance's
// user directory is shared, same as it is on the Users tab) - only gated
// by the caller having app-maintenance access at all (checked by the
// caller via GrantedModules before this is called).
func (s *Store) SearchUsers(like string) ([]Result, error) {
	rows, err := s.db.Query(`
		SELECT id, username, full_name
		FROM app_maintenance.users
		WHERE username ILIKE $1 OR full_name ILIKE $1
		ORDER BY username LIMIT `+strconv.Itoa(resultLimit), like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Result
	for rows.Next() {
		var id, username, fullName string
		if err := rows.Scan(&id, &username, &fullName); err != nil {
			return nil, err
		}
		title := fullName
		if title == "" {
			title = username
		}
		out = append(out, Result{
			Type: "user", ID: id, Title: title, Subtitle: "@" + username,
			ModuleCode: "app-maintenance", Link: "/app-maintenance/",
		})
	}
	return out, rows.Err()
}
