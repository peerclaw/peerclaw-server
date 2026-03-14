package notification

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/coder/websocket"
)

// DashboardHub manages WebSocket connections for dashboard notification push.
// It is separate from the agent signaling hub (JWT auth vs Ed25519, keyed by userID).
type DashboardHub struct {
	mu     sync.RWMutex
	conns  map[string]*websocket.Conn // userID -> conn
	logger *slog.Logger
}

// NewDashboardHub creates a new DashboardHub.
func NewDashboardHub(logger *slog.Logger) *DashboardHub {
	if logger == nil {
		logger = slog.Default()
	}
	return &DashboardHub{
		conns:  make(map[string]*websocket.Conn),
		logger: logger,
	}
}

// AddConn registers a WebSocket connection for a user.
// If an existing connection exists, it is replaced.
func (h *DashboardHub) AddConn(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if old, ok := h.conns[userID]; ok {
		_ = old.Close(websocket.StatusGoingAway, "replaced by new connection")
	}
	h.conns[userID] = conn
}

// RemoveConn removes the WebSocket connection for a user.
func (h *DashboardHub) RemoveConn(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing, ok := h.conns[userID]; ok && existing == conn {
		delete(h.conns, userID)
	}
}

// Push sends a notification to the user's WebSocket connection if connected.
func (h *DashboardHub) Push(n *Notification) {
	h.mu.RLock()
	conn, ok := h.conns[n.UserID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	data, err := json.Marshal(n)
	if err != nil {
		h.logger.Debug("failed to marshal notification for ws push", "error", err)
		return
	}

	if err := conn.Write(nil, websocket.MessageText, data); err != nil {
		h.logger.Debug("failed to push notification via ws", "user_id", n.UserID, "error", err)
	}
}

// CloseAll closes all active WebSocket connections.
func (h *DashboardHub) CloseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for userID, conn := range h.conns {
		_ = conn.Close(websocket.StatusGoingAway, "server shutting down")
		delete(h.conns, userID)
	}
}
