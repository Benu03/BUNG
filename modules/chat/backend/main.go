package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dataDir := getenv("DATA_DIR", "/data")

	db := openDB()
	defer db.Close()

	a := &api{store: NewStore(db), blobs: newBlobStore(dataDir), hub: newHub()}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /me", a.me)
	mux.HandleFunc("GET /is-admin", a.isAdmin)
	mux.HandleFunc("GET /users", a.listUsers) // for the "add friend" search
	mux.HandleFunc("GET /ws", a.serveWS)      // real-time updates, see hub.go

	mux.HandleFunc("GET /friends", a.listFriends)
	mux.HandleFunc("GET /friend-requests", a.listFriendRequests)
	mux.HandleFunc("POST /friend-requests", a.createFriendRequest)
	mux.HandleFunc("POST /friend-requests/{id}/accept", a.acceptFriendRequest)
	mux.HandleFunc("POST /friend-requests/{id}/decline", a.declineFriendRequest)

	mux.HandleFunc("GET /conversations", a.listConversations)
	mux.HandleFunc("POST /conversations", a.startConversation)
	mux.HandleFunc("GET /conversations/{id}/messages", a.listMessages)
	mux.HandleFunc("POST /conversations/{id}/messages", a.sendMessage)
	mux.HandleFunc("POST /conversations/{id}/attachments", a.uploadAttachment)
	mux.HandleFunc("POST /conversations/{id}/read", a.markRead)
	mux.HandleFunc("GET /messages/{messageId}/download", a.downloadAttachment)

	mux.HandleFunc("GET /broadcasts", a.listBroadcasts)
	mux.HandleFunc("POST /broadcasts", a.createBroadcast) // app-maintenance admins only, checked in-handler

	log.Printf("chat-backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
