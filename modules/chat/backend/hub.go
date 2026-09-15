package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// Same trust boundary as every other module's WebSocket (kanban) -
	// nginx is the only thing that can ever reach this backend's port.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// hub fans out chat events to every connection a user currently has open
// (they might have more than one tab) - keyed by user id rather than by
// conversation/board like kanban's hub, since a user needs live updates
// across all their conversations plus friend-request/broadcast events, not
// just one at a time. In-memory only - see kanban's hub.go for the same
// "fine for one replica" caveat.
type hub struct {
	mu    sync.Mutex
	conns map[string]map[*websocket.Conn]bool // userID -> set of conns
}

func newHub() *hub {
	return &hub{conns: map[string]map[*websocket.Conn]bool{}}
}

func (h *hub) register(userID string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[userID] == nil {
		h.conns[userID] = map[*websocket.Conn]bool{}
	}
	h.conns[userID][c] = true
}

func (h *hub) unregister(userID string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns[userID], c)
	if len(h.conns[userID]) == 0 {
		delete(h.conns, userID)
	}
}

// isOnline reports whether userID currently has at least one open
// connection - i.e. they have the chat page open right now, in some tab.
// Used to decide whether a new message also needs a row in
// notifications.inbox (see notify.go): if they're online they'll get it
// live over this same socket, so a separate notification would just be
// noise; if they're not, this is the only way they'll ever hear about it
// short of reopening chat themselves.
func (h *hub) isOnline(userID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.conns[userID]) > 0
}

func (h *hub) send(userID string, payload any) {
	if userID == "" {
		return
	}
	h.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(h.conns[userID]))
	for c := range h.conns[userID] {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	if len(conns) == 0 {
		return
	}

	msg, err := json.Marshal(payload)
	if err != nil {
		return
	}
	for _, c := range conns {
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("ws send to %s failed: %v", userID, err)
		}
	}
}

// broadcastAll sends to every currently-connected user (used for admin
// broadcasts) - offline users just see it next time they load /broadcasts.
func (h *hub) broadcastAll(payload any) {
	h.mu.Lock()
	userIDs := make([]string, 0, len(h.conns))
	for userID := range h.conns {
		userIDs = append(userIDs, userID)
	}
	h.mu.Unlock()
	for _, userID := range userIDs {
		h.send(userID, payload)
	}
}

// serveWS upgrades to a WebSocket and registers it under the caller's own
// user id (nginx/auth-common.conf already verified who they are). Same
// receive-only-from-the-client shape as kanban's boardWS: the read loop
// just exists to notice disconnects and answer pings/close frames, with a
// server-initiated ping every 30s to survive idle proxies.
func (a *api) serveWS(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	a.hub.register(userID, conn)
	defer a.hub.unregister(userID, conn)

	done := make(chan struct{})
	defer close(done)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
