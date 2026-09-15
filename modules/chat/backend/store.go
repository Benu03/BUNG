package main

import (
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrForbidden = errors.New("forbidden")
var ErrConflict = errors.New("conflict")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ---- users (cross-schema) ----

// ListAllUsers is a cross-schema read of app-maintenance's user directory,
// used to power the "add friend" search - same pattern as kanban's
// ListAllUsers.
func (s *Store) ListAllUsers() ([]*UserRef, error) {
	rows, err := s.db.Query(`SELECT id, username, full_name FROM app_maintenance.users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*UserRef{}
	for rows.Next() {
		var u UserRef
		if err := rows.Scan(&u.ID, &u.Username, &u.FullName); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (s *Store) getUserRef(id string) (*UserRef, error) {
	var u UserRef
	err := s.db.QueryRow(`SELECT id, username, full_name FROM app_maintenance.users WHERE id = $1`, id).
		Scan(&u.ID, &u.Username, &u.FullName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

// IsAppMaintenanceAdmin reports whether userID holds any role in the
// app-maintenance module - same cross-schema join pattern as
// search-backend's GrantedModules. Used to gate broadcasts.
func (s *Store) IsAppMaintenanceAdmin(userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM app_maintenance.user_roles ur
			JOIN app_maintenance.roles r ON r.id = ur.role_id
			JOIN app_maintenance.modules m ON m.id = r.module_id
			WHERE ur.user_id = $1 AND m.code = 'app-maintenance' AND r.is_active = true AND m.is_active = true
		)`, userID).Scan(&exists)
	return exists, err
}

// ---- friend requests ----

// SendFriendRequest looks up toUsername and creates a pending request from
// fromUserID. If toUsername already sent fromUserID a pending request,
// that one is accepted instead of creating a crossing duplicate - a nicer
// UX than making both people notice and cancel the other's request.
func (s *Store) SendFriendRequest(fromUserID, toUsername string) (*FriendRequest, error) {
	var toUserID string
	if err := s.db.QueryRow(`SELECT id FROM app_maintenance.users WHERE username = $1`, toUsername).Scan(&toUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if toUserID == fromUserID {
		return nil, ErrConflict
	}

	// Reverse pending request already exists - accept it instead.
	var reverseID string
	err := s.db.QueryRow(
		`SELECT id FROM friend_requests WHERE from_user_id = $1 AND to_user_id = $2 AND status = 'pending'`,
		toUserID, fromUserID,
	).Scan(&reverseID)
	if err == nil {
		return s.AcceptFriendRequest(fromUserID, reverseID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var id string
	var createdAt, updatedAt time.Time
	err = s.db.QueryRow(
		`INSERT INTO friend_requests (from_user_id, to_user_id, status) VALUES ($1, $2, 'pending')
		 ON CONFLICT (from_user_id, to_user_id) DO UPDATE SET status = 'pending', updated_at = now()
		 RETURNING id, created_at, updated_at`,
		fromUserID, toUserID,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	fromUser, err := s.getUserRef(fromUserID)
	if err != nil {
		return nil, err
	}
	toUser, err := s.getUserRef(toUserID)
	if err != nil {
		return nil, err
	}
	return &FriendRequest{ID: id, FromUser: *fromUser, ToUser: *toUser, Status: "pending", CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

// ListFriendRequests returns every pending request involving userID,
// tagged with Direction ("incoming" = userID needs to respond, "outgoing"
// = userID is waiting on the other person).
func (s *Store) ListFriendRequests(userID string) ([]*FriendRequest, error) {
	rows, err := s.db.Query(
		`SELECT id, from_user_id, to_user_id, status, created_at, updated_at
		 FROM friend_requests WHERE (from_user_id = $1 OR to_user_id = $1) AND status = 'pending'
		 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := []*FriendRequest{}
	for rows.Next() {
		var fr FriendRequest
		var fromID, toID string
		if err := rows.Scan(&fr.ID, &fromID, &toID, &fr.Status, &fr.CreatedAt, &fr.UpdatedAt); err != nil {
			return nil, err
		}
		fromUser, err := s.getUserRef(fromID)
		if err != nil {
			return nil, err
		}
		toUser, err := s.getUserRef(toID)
		if err != nil {
			return nil, err
		}
		fr.FromUser = *fromUser
		fr.ToUser = *toUser
		if fromID == userID {
			fr.Direction = "outgoing"
		} else {
			fr.Direction = "incoming"
		}
		requests = append(requests, &fr)
	}
	return requests, rows.Err()
}

// AcceptFriendRequest is only callable by the request's recipient.
// Creating the underlying conversation happens lazily on first message
// (see GetOrCreateConversation), not here.
func (s *Store) AcceptFriendRequest(userID, requestID string) (*FriendRequest, error) {
	var fromID, toID string
	if err := s.db.QueryRow(`SELECT from_user_id, to_user_id FROM friend_requests WHERE id = $1 AND status = 'pending'`, requestID).
		Scan(&fromID, &toID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if toID != userID {
		return nil, ErrForbidden
	}

	var createdAt, updatedAt time.Time
	if err := s.db.QueryRow(
		`UPDATE friend_requests SET status = 'accepted', updated_at = now() WHERE id = $1 RETURNING created_at, updated_at`,
		requestID,
	).Scan(&createdAt, &updatedAt); err != nil {
		return nil, err
	}

	fromUser, err := s.getUserRef(fromID)
	if err != nil {
		return nil, err
	}
	toUser, err := s.getUserRef(toID)
	if err != nil {
		return nil, err
	}
	return &FriendRequest{ID: requestID, FromUser: *fromUser, ToUser: *toUser, Status: "accepted", CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func (s *Store) DeclineFriendRequest(userID, requestID string) error {
	res, err := s.db.Exec(
		`UPDATE friend_requests SET status = 'declined', updated_at = now() WHERE id = $1 AND to_user_id = $2 AND status = 'pending'`,
		requestID, userID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// AreFriends reports whether an accepted friend_requests row exists
// between the two users, in either direction.
func (s *Store) AreFriends(userA, userB string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(
		`SELECT EXISTS(
			SELECT 1 FROM friend_requests
			WHERE status = 'accepted' AND (
				(from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1)
			)
		)`, userA, userB).Scan(&exists)
	return exists, err
}

func (s *Store) ListFriends(userID string) ([]*UserRef, error) {
	rows, err := s.db.Query(
		`SELECT CASE WHEN from_user_id = $1 THEN to_user_id ELSE from_user_id END AS friend_id
		 FROM friend_requests WHERE status = 'accepted' AND (from_user_id = $1 OR to_user_id = $1)`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friendIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		friendIDs = append(friendIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	friends := []*UserRef{}
	for _, id := range friendIDs {
		u, err := s.getUserRef(id)
		if err != nil {
			return nil, err
		}
		friends = append(friends, u)
	}
	return friends, nil
}

// ---- conversations ----

// conversationKey orders a pair so (userA, userB) always maps to the same
// row regardless of who's asking - conversations.user_a_id/user_b_id are
// always stored in this order.
func conversationKey(userA, userB string) (a, b string) {
	if userA < userB {
		return userA, userB
	}
	return userB, userA
}

// GetOrCreateConversation requires the two users to already be friends
// (checked by the caller via AreFriends) - it just lazily creates the
// conversation row on first use.
func (s *Store) GetOrCreateConversation(userA, userB string) (string, error) {
	a, b := conversationKey(userA, userB)
	var id string
	err := s.db.QueryRow(`SELECT id FROM conversations WHERE user_a_id = $1 AND user_b_id = $2`, a, b).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	err = s.db.QueryRow(
		`INSERT INTO conversations (user_a_id, user_b_id) VALUES ($1, $2)
		 ON CONFLICT (user_a_id, user_b_id) DO UPDATE SET user_a_id = EXCLUDED.user_a_id
		 RETURNING id`,
		a, b,
	).Scan(&id)
	return id, err
}

// conversationOtherUser resolves who userID is talking to in
// conversationID, and confirms userID is actually part of it (returning
// ErrNotFound otherwise - same "not found" for "not yours" reasoning used
// throughout the app).
func (s *Store) conversationOtherUser(conversationID, userID string) (string, error) {
	var a, b string
	err := s.db.QueryRow(`SELECT user_a_id, user_b_id FROM conversations WHERE id = $1`, conversationID).Scan(&a, &b)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	switch userID {
	case a:
		return b, nil
	case b:
		return a, nil
	default:
		return "", ErrNotFound
	}
}

func (s *Store) ListConversations(userID string) ([]*Conversation, error) {
	rows, err := s.db.Query(
		`SELECT id, user_a_id, user_b_id, created_at FROM conversations WHERE user_a_id = $1 OR user_b_id = $1 ORDER BY created_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type row struct{ id, a, b string; createdAt time.Time }
	var raw []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.a, &r.b, &r.createdAt); err != nil {
			return nil, err
		}
		raw = append(raw, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	conversations := []*Conversation{}
	for _, r := range raw {
		otherID := r.a
		if userID == r.a {
			otherID = r.b
		}
		friend, err := s.getUserRef(otherID)
		if err != nil {
			return nil, err
		}
		last, err := s.lastMessage(r.id)
		if err != nil {
			return nil, err
		}
		unread, err := s.unreadCount(r.id, userID)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, &Conversation{
			ID: r.id, Friend: *friend, LastMessage: last, UnreadCount: unread, CreatedAt: r.createdAt,
		})
	}
	return conversations, nil
}

const messageColumns = `id, conversation_id, sender_id, body, attachment_filename, attachment_content_type, attachment_size, attachment_storage_path, created_at, read_at`

func scanMessage(row interface{ Scan(...any) error }) (*Message, error) {
	var m Message
	var filename, contentType, storagePath sql.NullString
	var size sql.NullInt64
	var readAt sql.NullTime
	if err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &filename, &contentType, &size, &storagePath, &m.CreatedAt, &readAt); err != nil {
		return nil, err
	}
	if filename.Valid {
		m.AttachmentFilename = &filename.String
	}
	if contentType.Valid {
		m.AttachmentContentType = &contentType.String
	}
	if size.Valid {
		m.AttachmentSize = &size.Int64
	}
	if storagePath.Valid {
		m.AttachmentStoragePath = &storagePath.String
	}
	if readAt.Valid {
		m.ReadAt = &readAt.Time
	}
	return &m, nil
}

func (s *Store) lastMessage(conversationID string) (*Message, error) {
	row := s.db.QueryRow(`SELECT `+messageColumns+` FROM messages WHERE conversation_id = $1 ORDER BY created_at DESC LIMIT 1`, conversationID)
	m, err := scanMessage(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (s *Store) unreadCount(conversationID, userID string) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT count(*) FROM messages WHERE conversation_id = $1 AND sender_id != $2 AND read_at IS NULL`,
		conversationID, userID,
	).Scan(&count)
	return count, err
}

// ListMessages returns a conversation's messages oldest-first, provided
// userID is actually part of it.
func (s *Store) ListMessages(userID, conversationID string, limit int) ([]*Message, error) {
	if _, err := s.conversationOtherUser(conversationID, userID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		`SELECT `+messageColumns+` FROM messages WHERE conversation_id = $1 ORDER BY created_at DESC LIMIT $2`,
		conversationID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialized (not `var messages []*Message`) so a conversation with
	// zero messages yet serializes as `[]`, not `null` - the frontend
	// calls .map()/.length on this without a null-guard, same as every
	// other list endpoint here.
	messages := []*Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Reverse to oldest-first for display.
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func (s *Store) SendMessage(userID, conversationID, body string) (*Message, error) {
	if _, err := s.conversationOtherUser(conversationID, userID); err != nil {
		return nil, err
	}
	row := s.db.QueryRow(
		`INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3) RETURNING `+messageColumns,
		conversationID, userID, body,
	)
	return scanMessage(row)
}

// CreateAttachmentMessage reserves a message row (storage_path filled
// with a placeholder until the blob is written - see
// UpdateMessageAttachment), same two-step pattern as my-storage/ticketing.
func (s *Store) CreateAttachmentMessage(userID, conversationID, filename, contentType string) (*Message, error) {
	if _, err := s.conversationOtherUser(conversationID, userID); err != nil {
		return nil, err
	}
	row := s.db.QueryRow(
		`INSERT INTO messages (conversation_id, sender_id, attachment_filename, attachment_content_type, attachment_storage_path)
		 VALUES ($1, $2, $3, $4, 'pending') RETURNING `+messageColumns,
		conversationID, userID, filename, contentType,
	)
	return scanMessage(row)
}

func (s *Store) UpdateMessageAttachment(id, storagePath string, size int64) error {
	_, err := s.db.Exec(`UPDATE messages SET attachment_storage_path = $1, attachment_size = $2 WHERE id = $3`, storagePath, size, id)
	return err
}

func (s *Store) GetMessage(id string) (*Message, error) {
	row := s.db.QueryRow(`SELECT `+messageColumns+` FROM messages WHERE id = $1`, id)
	m, err := scanMessage(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return m, err
}

func (s *Store) MarkRead(userID, conversationID string) error {
	if _, err := s.conversationOtherUser(conversationID, userID); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`UPDATE messages SET read_at = now() WHERE conversation_id = $1 AND sender_id != $2 AND read_at IS NULL`,
		conversationID, userID,
	)
	return err
}

// ---- broadcasts ----

func (s *Store) ListBroadcasts(limit int) ([]*Broadcast, error) {
	rows, err := s.db.Query(`SELECT id, sender_id, body, created_at FROM broadcasts ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	broadcasts := []*Broadcast{}
	for rows.Next() {
		var b Broadcast
		var senderID string
		if err := rows.Scan(&b.ID, &senderID, &b.Body, &b.CreatedAt); err != nil {
			return nil, err
		}
		sender, err := s.getUserRef(senderID)
		if err != nil {
			return nil, err
		}
		b.Sender = *sender
		broadcasts = append(broadcasts, &b)
	}
	return broadcasts, rows.Err()
}

func (s *Store) CreateBroadcast(senderID, body string) (*Broadcast, error) {
	var b Broadcast
	if err := s.db.QueryRow(
		`INSERT INTO broadcasts (sender_id, body) VALUES ($1, $2) RETURNING id, created_at`,
		senderID, body,
	).Scan(&b.ID, &b.CreatedAt); err != nil {
		return nil, err
	}
	sender, err := s.getUserRef(senderID)
	if err != nil {
		return nil, err
	}
	b.Sender = *sender
	b.Body = body
	return &b, nil
}
