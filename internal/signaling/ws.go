package signaling

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/peerclaw/peerclaw-go/signaling"
	"nhooyr.io/websocket"
)

// Hub manages WebSocket connections for signaling between agents.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]*websocket.Conn // agentID -> connection
	logger *slog.Logger
}

// NewHub creates a new signaling hub.
func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		conns:  make(map[string]*websocket.Conn),
		logger: logger,
	}
}

// HandleConnect upgrades an HTTP request to a WebSocket connection for signaling.
// The agent_id query parameter identifies the connecting agent.
func (h *Hub) HandleConnect(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "missing agent_id parameter", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.logger.Error("websocket accept failed", "error", err)
		return
	}

	h.mu.Lock()
	// Close existing connection for this agent if any.
	if old, exists := h.conns[agentID]; exists {
		old.Close(websocket.StatusGoingAway, "replaced by new connection")
	}
	h.conns[agentID] = conn
	h.mu.Unlock()

	h.logger.Info("agent connected to signaling", "agent_id", agentID)

	defer func() {
		h.mu.Lock()
		if h.conns[agentID] == conn {
			delete(h.conns, agentID)
		}
		h.mu.Unlock()
		conn.Close(websocket.StatusNormalClosure, "")
		h.logger.Info("agent disconnected from signaling", "agent_id", agentID)
	}()

	// Read loop: forward messages to target agents.
	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				h.logger.Debug("websocket closed", "agent_id", agentID)
			} else {
				h.logger.Error("websocket read error", "agent_id", agentID, "error", err)
			}
			return
		}

		var msg signaling.SignalMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			h.logger.Warn("invalid signal message", "agent_id", agentID, "error", err)
			continue
		}

		msg.From = agentID
		h.Forward(r.Context(), msg)
	}
}

// Forward sends a signal message to the target agent.
func (h *Hub) Forward(ctx context.Context, msg signaling.SignalMessage) {
	h.mu.RLock()
	target, ok := h.conns[msg.To]
	h.mu.RUnlock()

	if !ok {
		h.logger.Warn("target agent not connected", "from", msg.From, "to", msg.To)
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("marshal signal message", "error", err)
		return
	}

	if err := target.Write(ctx, websocket.MessageText, data); err != nil {
		h.logger.Error("forward signal message", "from", msg.From, "to", msg.To, "error", err)
	}
}

// ConnectedAgents returns the number of currently connected agents.
func (h *Hub) ConnectedAgents() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}
