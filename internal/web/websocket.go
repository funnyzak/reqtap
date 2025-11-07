package web

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/funnyzak/reqtap/internal/logger"
)

// WebsocketHub manages live connections for request broadcasts.
type WebsocketHub struct {
	logger  logger.Logger
	clients map[*websocket.Conn]struct{}
	mu      sync.RWMutex

	upgrader websocket.Upgrader
}

// NewWebsocketHub creates a new hub.
func NewWebsocketHub(log logger.Logger) *WebsocketHub {
	return &WebsocketHub{
		logger:  log,
		clients: make(map[*websocket.Conn]struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// Upgrade upgrades the HTTP connection to WebSocket.
func (h *WebsocketHub) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	h.register(conn)
	return conn, nil
}

func (h *WebsocketHub) register(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()

	go h.readLoop(conn)
}

func (h *WebsocketHub) readLoop(conn *websocket.Conn) {
	defer h.unregister(conn)

	conn.SetReadLimit(1024)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *WebsocketHub) unregister(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()

	conn.Close()
}

// Broadcast sends payload to all active connections.
func (h *WebsocketHub) Broadcast(event interface{}) {
	h.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(h.clients))
	for conn := range h.clients {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()

	if len(conns) == 0 {
		return
	}

	payload, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("Failed to marshal websocket payload", "error", err)
		return
	}

	for _, conn := range conns {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			h.logger.Warn("Failed to write to websocket client", "error", err)
			h.unregister(conn)
		}
	}
}

// Close terminates all connections.
func (h *WebsocketHub) Close() {
	h.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(h.clients))
	for conn := range h.clients {
		conns = append(conns, conn)
	}
	h.clients = make(map[*websocket.Conn]struct{})
	h.mu.Unlock()

	for _, conn := range conns {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		conn.Close()
	}
}
