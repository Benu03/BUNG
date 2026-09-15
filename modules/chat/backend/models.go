package main

import "time"

// UserRef is a minimal, read-only view of an app-maintenance user - same
// cross-schema-lookup pattern as every other module's "add member"-style
// search (see store.go's ListAllUsers).
type UserRef struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"fullName"`
}

// FriendRequest is one direction of a friend request/relationship.
// Direction ("incoming"/"outgoing") is computed relative to the caller,
// not stored - see store.go's ListFriendRequests.
type FriendRequest struct {
	ID        string    `json:"id"`
	FromUser  UserRef   `json:"fromUser"`
	ToUser    UserRef   `json:"toUser"`
	Status    string    `json:"status"`
	Direction string    `json:"direction,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Conversation is a 1:1 thread with a friend, from the caller's point of
// view - Friend is always "the other person", never the caller.
type Conversation struct {
	ID          string    `json:"id"`
	Friend      UserRef   `json:"friend"`
	LastMessage *Message  `json:"lastMessage,omitempty"`
	UnreadCount int       `json:"unreadCount"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Message is one chat message - either plain text, a file attachment, or
// both. Attachment bytes live on disk (see storage.go), only this row's
// attachment_* columns are in Postgres, same split as my-storage/ticketing.
type Message struct {
	ID                    string     `json:"id"`
	ConversationID        string     `json:"conversationId"`
	SenderID              string     `json:"senderId"`
	Body                  string     `json:"body"`
	AttachmentFilename    *string    `json:"attachmentFilename,omitempty"`
	AttachmentContentType *string    `json:"attachmentContentType,omitempty"`
	AttachmentSize        *int64     `json:"attachmentSize,omitempty"`
	AttachmentStoragePath *string    `json:"-"` // internal only, never serialized to clients
	CreatedAt             time.Time  `json:"createdAt"`
	ReadAt                *time.Time `json:"readAt,omitempty"`
}

// Broadcast is an admin-only announcement, visible to everyone regardless
// of friend status (see migrate.go's comment on the broadcasts table).
type Broadcast struct {
	ID        string    `json:"id"`
	Sender    UserRef   `json:"sender"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type friendRequestBody struct {
	Username string `json:"username"`
}

type sendMessageRequest struct {
	Body string `json:"body"`
}

type broadcastRequest struct {
	Body string `json:"body"`
}
