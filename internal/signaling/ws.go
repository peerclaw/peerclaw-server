package signaling

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/peerclaw/peerclaw-core/signaling"
	"github.com/peerclaw/peerclaw-server/internal/audit"
	"github.com/peerclaw/peerclaw-server/internal/identity"
	"github.com/peerclaw/peerclaw-server/internal/observability"
	"nhooyr.io/websocket"
)

// TURNConfig holds TURN server credentials for ICE negotiation.
type TURNConfig struct {
	URLs       []string
	Username   string
	Credential string
}

// authFrame is the authentication message sent by the client after WebSocket upgrade.
type authFrame struct {
	AgentID   string `json:"agent_id"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
	PublicKey string `json:"public_key"`
}

// ContactsChecker checks whether signaling between two agents is allowed.
type ContactsChecker interface {
	IsAllowed(ctx context.Context, fromAgentID, toAgentID string) (bool, error)
}

// Hub manages WebSocket connections for signaling between agents.
type Hub struct {
	mu           sync.RWMutex
	conns        map[string]*websocket.Conn // agentID -> connection
	logger       *slog.Logger
	turnConfig   *TURNConfig
	maxConns     int // 0 means unlimited
	audit        *audit.Logger
	metrics      *observability.Metrics
	broker       Broker
	verifier     *identity.Verifier
	contacts     ContactsChecker
	authRequired bool
	authTimeout  time.Duration // default 5s
}

// NewHub creates a new signaling hub.
// turnCfg may be nil if no TURN servers are configured.
// maxConns limits the total number of WebSocket connections (0 = unlimited).
func NewHub(logger *slog.Logger, turnCfg *TURNConfig, maxConns int) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		conns:        make(map[string]*websocket.Conn),
		logger:       logger,
		turnConfig:   turnCfg,
		maxConns:     maxConns,
		authRequired: true,
		authTimeout:  5 * time.Second,
	}
}

// SetVerifier sets the identity verifier for WebSocket authentication.
func (h *Hub) SetVerifier(v *identity.Verifier) {
	h.verifier = v
}

// SetAuthRequired enables mandatory authentication for WebSocket connections.
func (h *Hub) SetAuthRequired(required bool) {
	h.authRequired = required
}

// SetAudit sets the audit logger for recording signaling events.
func (h *Hub) SetAudit(a *audit.Logger) {
	h.audit = a
}

// SetMetrics sets the metrics instruments for observability.
func (h *Hub) SetMetrics(m *observability.Metrics) {
	h.metrics = m
}

// SetBroker sets the signaling broker for message distribution.
// When set, Forward() delegates to the broker instead of delivering directly.
func (h *Hub) SetBroker(b Broker) {
	h.broker = b
}

// SetContacts sets the contacts checker for signaling whitelist enforcement.
func (h *Hub) SetContacts(c ContactsChecker) {
	h.contacts = c
}

// DeliverLocal delivers a signal message to a locally connected agent.
// This is called by brokers to deliver messages that should be sent on this node.
func (h *Hub) DeliverLocal(ctx context.Context, msg signaling.SignalMessage) {
	h.mu.RLock()
	target, ok := h.conns[msg.To]
	h.mu.RUnlock()

	if !ok {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("marshal signal message for local delivery", "error", err)
		return
	}

	if err := target.Write(ctx, websocket.MessageText, data); err != nil {
		h.logger.Error("local deliver signal message", "from", msg.From, "to", msg.To, "error", err)
		return
	}
	if h.metrics != nil {
		h.metrics.SignalingMessagesTotal.Add(ctx, 1)
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

	// Check connection limit before accepting.
	if h.maxConns > 0 {
		h.mu.RLock()
		_, alreadyConnected := h.conns[agentID]
		currentCount := len(h.conns)
		h.mu.RUnlock()

		if !alreadyConnected && currentCount >= h.maxConns {
			h.logger.Warn("WebSocket connection limit reached",
				"max", h.maxConns,
				"agent_id", agentID,
			)
			http.Error(w, "too many connections", http.StatusServiceUnavailable)
			return
		}
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err == nil {
		// Limit individual message size to 64KB.
		conn.SetReadLimit(64 * 1024)
	}
	if err != nil {
		h.logger.Error("websocket accept failed", "error", err)
		return
	}

	// Authenticate the WebSocket connection.
	if h.authRequired && h.verifier != nil {
		authedID, authErr := h.authenticateConn(r.Context(), conn, agentID)
		if authErr != nil {
			h.logger.Warn("websocket auth failed", "agent_id", agentID, "error", authErr)
			_ = conn.Close(websocket.StatusPolicyViolation, "authentication failed")
			return
		}
		agentID = authedID
	}

	h.mu.Lock()
	// Close existing connection for this agent if any.
	if old, exists := h.conns[agentID]; exists {
		_ = old.Close(websocket.StatusGoingAway, "replaced by new connection")
	}
	h.conns[agentID] = conn
	h.mu.Unlock()

	h.logger.Info("agent connected to signaling", "agent_id", agentID)
	if h.audit != nil {
		h.audit.LogSignalingConnect(r.Context(), agentID, r.RemoteAddr)
	}
	if h.metrics != nil {
		h.metrics.SignalingConnections.Add(r.Context(), 1)
	}

	// Push ICE server configuration if TURN is configured.
	if h.turnConfig != nil && len(h.turnConfig.URLs) > 0 {
		configMsg := signaling.SignalMessage{
			Type: signaling.MessageTypeConfig,
			ICEServers: []signaling.ICEServerConfig{
				{
					URLs:       h.turnConfig.URLs,
					Username:   h.turnConfig.Username,
					Credential: h.turnConfig.Credential,
				},
			},
		}
		data, err := json.Marshal(configMsg)
		if err != nil {
			h.logger.Error("marshal config message", "error", err)
		} else if err := conn.Write(r.Context(), websocket.MessageText, data); err != nil {
			h.logger.Error("send config message", "agent_id", agentID, "error", err)
		} else {
			h.logger.Debug("sent ICE server config", "agent_id", agentID)
		}
	}

	defer func() {
		h.mu.Lock()
		if h.conns[agentID] == conn {
			delete(h.conns, agentID)
		}
		h.mu.Unlock()
		_ = conn.Close(websocket.StatusNormalClosure, "")
		h.logger.Info("agent disconnected from signaling", "agent_id", agentID)
		if h.audit != nil {
			h.audit.LogSignalingDisconnect(r.Context(), agentID)
		}
		if h.metrics != nil {
			h.metrics.SignalingConnections.Add(r.Context(), -1)
		}
	}()

	// Rate limiter: 10 messages/sec, burst 20.
	msgLimiter := newMessageLimiter(10, 20)

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

		// Enforce message rate limit.
		if !msgLimiter.allow() {
			h.logger.Warn("websocket rate limit exceeded, closing", "agent_id", agentID)
			_ = conn.Close(websocket.StatusPolicyViolation, "rate limit exceeded")
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

// authenticateConn waits for an auth frame from the client within the auth timeout.
func (h *Hub) authenticateConn(ctx context.Context, conn *websocket.Conn, expectedAgentID string) (string, error) {
	authCtx, cancel := context.WithTimeout(ctx, h.authTimeout)
	defer cancel()

	_, data, err := conn.Read(authCtx)
	if err != nil {
		return "", fmt.Errorf("read auth frame: %w", err)
	}

	var frame authFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		return "", fmt.Errorf("parse auth frame: %w", err)
	}

	if frame.AgentID == "" || frame.PublicKey == "" || frame.Signature == "" {
		return "", fmt.Errorf("incomplete auth frame")
	}

	if frame.AgentID != expectedAgentID {
		return "", fmt.Errorf("agent_id mismatch: query=%s, frame=%s", expectedAgentID, frame.AgentID)
	}

	// Verify timestamp is within 30 seconds to prevent replay.
	now := time.Now().Unix()
	if abs64(now-frame.Timestamp) > 30 {
		return "", fmt.Errorf("auth frame timestamp too old")
	}

	// Verify signature over "agent_id:timestamp".
	payload := []byte(fmt.Sprintf("%s:%d", frame.AgentID, frame.Timestamp))
	if err := h.verifier.VerifySignature(frame.PublicKey, payload, frame.Signature); err != nil {
		return "", fmt.Errorf("signature verification: %w", err)
	}

	return frame.AgentID, nil
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// isSignalingType returns true for message types that establish P2P connections.
func isSignalingType(t signaling.MessageType) bool {
	return t == signaling.MessageTypeOffer ||
		t == signaling.MessageTypeAnswer ||
		t == signaling.MessageTypeICECandidate
}

// Forward sends a signal message to the target agent.
// When a broker is set, messages are published through the broker for
// cross-node delivery. Otherwise, messages are delivered locally.
func (h *Hub) Forward(ctx context.Context, msg signaling.SignalMessage) {
	// Whitelist check for signaling messages (offer/answer/ICE).
	if h.contacts != nil && isSignalingType(msg.Type) {
		allowed, err := h.contacts.IsAllowed(ctx, msg.From, msg.To)
		if err != nil {
			h.logger.Error("contacts check failed", "from", msg.From, "to", msg.To, "error", err)
			return
		}
		if !allowed {
			h.logger.Warn("signaling blocked: not in contacts", "from", msg.From, "to", msg.To, "type", msg.Type)
			return
		}
	}

	if h.broker != nil {
		if err := h.broker.Publish(ctx, msg); err != nil {
			h.logger.Error("broker publish failed", "from", msg.From, "to", msg.To, "error", err)
		}
		return
	}

	// Direct local delivery (no broker).
	h.DeliverLocal(ctx, msg)
}

// DeliverEnvelope sends a bridge_message to a connected agent via WebSocket.
func (h *Hub) DeliverEnvelope(ctx context.Context, agentID string, envPayload json.RawMessage) error {
	h.mu.RLock()
	conn, ok := h.conns[agentID]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent %s not connected", agentID)
	}

	msg := signaling.SignalMessage{
		Type:    signaling.MessageTypeBridgeMessage,
		To:      agentID,
		Payload: envPayload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal bridge message: %w", err)
	}

	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		return fmt.Errorf("deliver bridge message: %w", err)
	}

	h.logger.Debug("delivered bridge message", "agent_id", agentID)
	return nil
}

// ConnectedAgents returns the number of currently connected agents.
func (h *Hub) ConnectedAgents() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

// HasAgent returns true if the given agent ID has an active WebSocket connection.
func (h *Hub) HasAgent(agentID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.conns[agentID]
	return ok
}

// CloseAll gracefully closes all WebSocket connections.
func (h *Hub) CloseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, conn := range h.conns {
		_ = conn.Close(websocket.StatusGoingAway, "server shutting down")
		delete(h.conns, id)
	}
}

// messageLimiter is a simple token bucket rate limiter for per-connection use.
type messageLimiter struct {
	tokens   float64
	maxBurst float64
	rate     float64 // tokens per second
	lastTime time.Time
}

func newMessageLimiter(rate float64, burst int) *messageLimiter {
	return &messageLimiter{
		tokens:   float64(burst),
		maxBurst: float64(burst),
		rate:     rate,
		lastTime: time.Now(),
	}
}

func (l *messageLimiter) allow() bool {
	now := time.Now()
	elapsed := now.Sub(l.lastTime).Seconds()
	l.lastTime = now
	l.tokens += elapsed * l.rate
	if l.tokens > l.maxBurst {
		l.tokens = l.maxBurst
	}
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}
