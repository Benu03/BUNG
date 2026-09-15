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
	// Same-origin only in practice - nginx is the only thing that can
	// ever reach this backend's port (not published to the host, see
	// docker-compose.yml), same trust boundary as every other module.
	// CheckOrigin is permissive here for that reason, not because origin
	// doesn't matter in general.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// hub fans out board-change notifications to every client currently
// viewing that board - in-memory only (fine for a single kanban-backend
// replica, which is what this project runs; a multi-replica deployment
// would need a shared pub/sub - e.g. Postgres LISTEN/NOTIFY - instead of
// this map, since each replica would otherwise only see its own
// connections' boards).
type hub struct {
	mu    sync.Mutex
	conns map[string]map[*websocket.Conn]bool // boardID -> set of conns
}

func newHub() *hub {
	return &hub{conns: map[string]map[*websocket.Conn]bool{}}
}

func (h *hub) register(boardID string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[boardID] == nil {
		h.conns[boardID] = map[*websocket.Conn]bool{}
	}
	h.conns[boardID][c] = true
}

func (h *hub) unregister(boardID string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns[boardID], c)
	if len(h.conns[boardID]) == 0 {
		delete(h.conns, boardID)
	}
}

// broadcast sends a small JSON event to every client currently viewing
// boardID. The payload is deliberately minimal (just a type + who did
// it) - clients just refetch the whole board on any event rather than
// reconciling a fine-grained patch, which keeps both sides simple at the
// cost of one extra GET per change (negligible for a board this size).
func (h *hub) broadcast(boardID, eventType, actorID string) {
	if boardID == "" {
		return
	}
	h.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(h.conns[boardID]))
	for c := range h.conns[boardID] {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	if len(conns) == 0 {
		return
	}

	msg, err := json.Marshal(map[string]string{"type": eventType, "actorId": actorID})
	if err != nil {
		return
	}
	for _, c := range conns {
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("ws broadcast to board %s failed: %v", boardID, err)
		}
	}
}

// boardWS upgrades to a WebSocket and registers the connection with the
// hub for boardID, requiring the same membership check as every other
// board endpoint. The client never needs to send anything meaningful -
// this is a receive-only channel from its perspective - so the read loop
// just exists to notice a closed connection and answer pings/close
// frames (gorilla's ReadMessage handles control frames internally).
// A server-initiated ping every 30s keeps the connection (and any
// in-between proxy's idle timeout) alive.
func (a *api) boardWS(w http.ResponseWriter, r *http.Request) {
	boardID := r.PathValue("id")
	userID := currentUserID(r)
	if err := a.store.requireMember(boardID, userID); err != nil {
		handleErr(w, err)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	a.hub.register(boardID, conn)
	defer a.hub.unregister(boardID, conn)

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
