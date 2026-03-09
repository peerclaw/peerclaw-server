package signaling

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/peerclaw/peerclaw-core/signaling"
	"nhooyr.io/websocket"
)

// ---------- helpers ----------

// dialHub upgrades an HTTP test server connection for the given agentID.
// It returns the client-side websocket.Conn that can send/receive messages.
func dialHub(t *testing.T, srv *httptest.Server, agentID string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?agent_id=" + agentID
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", agentID, err)
	}
	return conn
}

// readSignalMessage reads a single SignalMessage from the websocket connection.
func readSignalMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) signaling.SignalMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	var msg signaling.SignalMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	return msg
}

// newTestServer creates an httptest.Server that routes /ws to the hub's HandleConnect.
func newTestServer(hub *Hub) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleConnect)
	return httptest.NewServer(mux)
}

// waitForAgent polls until the hub reports the agent as connected (or times out).
func waitForAgent(t *testing.T, hub *Hub, agentID string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if hub.HasAgent(agentID) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("agent %q did not connect within %v", agentID, timeout)
}

// waitForAgentGone polls until the hub no longer reports the agent as connected.
func waitForAgentGone(t *testing.T, hub *Hub, agentID string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !hub.HasAgent(agentID) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("agent %q still connected after %v", agentID, timeout)
}

// ---------- tests ----------

func TestNewHub(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		hub := NewHub(nil, nil, 0)
		if hub == nil {
			t.Fatal("NewHub returned nil")
		}
		if hub.conns == nil {
			t.Fatal("conns map not initialised")
		}
		if hub.maxConns != 0 {
			t.Errorf("maxConns = %d, want 0", hub.maxConns)
		}
		if hub.logger == nil {
			t.Fatal("logger should default to slog.Default()")
		}
		if hub.authTimeout != 5*time.Second {
			t.Errorf("authTimeout = %v, want 5s", hub.authTimeout)
		}
	})

	t.Run("with_turn_config", func(t *testing.T) {
		turn := &TURNConfig{
			URLs:       []string{"turn:example.com:3478"},
			Username:   "user",
			Credential: "pass",
		}
		hub := NewHub(slog.Default(), turn, 100)
		if hub.turnConfig != turn {
			t.Fatal("TURN config not set")
		}
		if hub.maxConns != 100 {
			t.Errorf("maxConns = %d, want 100", hub.maxConns)
		}
	})

	t.Run("with_custom_logger", func(t *testing.T) {
		logger := slog.Default()
		hub := NewHub(logger, nil, 0)
		if hub.logger != logger {
			t.Fatal("custom logger not used")
		}
	})
}

func TestHubRegisterAndUnregister(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	srv := newTestServer(hub)
	defer srv.Close()

	// Register by connecting.
	conn := dialHub(t, srv, "agent-A")
	waitForAgent(t, hub, "agent-A", 2*time.Second)

	if !hub.HasAgent("agent-A") {
		t.Fatal("agent-A should be registered")
	}
	if hub.ConnectedAgents() != 1 {
		t.Errorf("ConnectedAgents = %d, want 1", hub.ConnectedAgents())
	}

	// Unregister by closing the client connection.
	_ = conn.Close(websocket.StatusNormalClosure, "bye")
	waitForAgentGone(t, hub, "agent-A", 2*time.Second)

	if hub.HasAgent("agent-A") {
		t.Fatal("agent-A should be unregistered after close")
	}
	if hub.ConnectedAgents() != 0 {
		t.Errorf("ConnectedAgents = %d, want 0", hub.ConnectedAgents())
	}
}

func TestHubHasAgent(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	srv := newTestServer(hub)
	defer srv.Close()

	// Unknown agent before any connection.
	if hub.HasAgent("nonexistent") {
		t.Fatal("HasAgent should return false for unknown agent")
	}

	conn := dialHub(t, srv, "agent-X")
	defer conn.Close(websocket.StatusNormalClosure, "")
	waitForAgent(t, hub, "agent-X", 2*time.Second)

	if !hub.HasAgent("agent-X") {
		t.Fatal("HasAgent should return true for connected agent")
	}

	// A different agent is still unknown.
	if hub.HasAgent("agent-Y") {
		t.Fatal("HasAgent should return false for a different agent")
	}
}

func TestHubConnectedAgents(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	srv := newTestServer(hub)
	defer srv.Close()

	if hub.ConnectedAgents() != 0 {
		t.Fatalf("ConnectedAgents = %d, want 0 initially", hub.ConnectedAgents())
	}

	const n = 5
	conns := make([]*websocket.Conn, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("agent-%d", i)
		conns[i] = dialHub(t, srv, id)
		waitForAgent(t, hub, id, 2*time.Second)
	}

	if hub.ConnectedAgents() != n {
		t.Errorf("ConnectedAgents = %d, want %d", hub.ConnectedAgents(), n)
	}

	// Close half and verify count.
	for i := 0; i < n/2; i++ {
		id := fmt.Sprintf("agent-%d", i)
		_ = conns[i].Close(websocket.StatusNormalClosure, "")
		waitForAgentGone(t, hub, id, 2*time.Second)
	}

	expected := n - n/2
	if hub.ConnectedAgents() != expected {
		t.Errorf("ConnectedAgents = %d, want %d after closing half", hub.ConnectedAgents(), expected)
	}

	// Cleanup remaining.
	for i := n / 2; i < n; i++ {
		_ = conns[i].Close(websocket.StatusNormalClosure, "")
	}
}

func TestHubForward(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	srv := newTestServer(hub)
	defer srv.Close()

	// Connect sender and receiver.
	sender := dialHub(t, srv, "sender")
	defer sender.Close(websocket.StatusNormalClosure, "")
	receiver := dialHub(t, srv, "receiver")
	defer receiver.Close(websocket.StatusNormalClosure, "")

	waitForAgent(t, hub, "sender", 2*time.Second)
	waitForAgent(t, hub, "receiver", 2*time.Second)

	// The sender writes a signal message to the hub (via the websocket).
	msg := signaling.SignalMessage{
		Type: signaling.MessageTypeOffer,
		From: "sender",
		To:   "receiver",
		SDP:  "test-sdp-offer",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := sender.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("sender write: %v", err)
	}

	// The receiver should get the message.
	got := readSignalMessage(t, receiver, 2*time.Second)
	if got.Type != signaling.MessageTypeOffer {
		t.Errorf("Type = %q, want %q", got.Type, signaling.MessageTypeOffer)
	}
	if got.From != "sender" {
		t.Errorf("From = %q, want %q", got.From, "sender")
	}
	if got.To != "receiver" {
		t.Errorf("To = %q, want %q", got.To, "receiver")
	}
	if got.SDP != "test-sdp-offer" {
		t.Errorf("SDP = %q, want %q", got.SDP, "test-sdp-offer")
	}
}

func TestHubForwardUnknown(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	srv := newTestServer(hub)
	defer srv.Close()

	// Connect only the sender; no receiver is connected.
	sender := dialHub(t, srv, "sender")
	defer sender.Close(websocket.StatusNormalClosure, "")
	waitForAgent(t, hub, "sender", 2*time.Second)

	// Forward a message to an unknown agent directly via Hub.Forward.
	// This should not panic or return an error — it silently drops.
	msg := signaling.SignalMessage{
		Type: signaling.MessageTypeOffer,
		From: "sender",
		To:   "unknown-agent",
		SDP:  "test-sdp",
	}
	hub.Forward(context.Background(), msg)

	// Also verify DeliverLocal doesn't panic for unknown agent.
	hub.DeliverLocal(context.Background(), msg)

	// Verify the hub is still healthy by checking the sender is still connected.
	if !hub.HasAgent("sender") {
		t.Fatal("sender should still be connected after forwarding to unknown agent")
	}
}

func TestHubMaxConnections(t *testing.T) {
	const maxConns = 3
	hub := NewHub(slog.Default(), nil, maxConns)
	srv := newTestServer(hub)
	defer srv.Close()

	// Fill up to the limit.
	conns := make([]*websocket.Conn, maxConns)
	for i := 0; i < maxConns; i++ {
		id := fmt.Sprintf("agent-%d", i)
		conns[i] = dialHub(t, srv, id)
		waitForAgent(t, hub, id, 2*time.Second)
	}

	if hub.ConnectedAgents() != maxConns {
		t.Fatalf("ConnectedAgents = %d, want %d", hub.ConnectedAgents(), maxConns)
	}

	// The next connection should be rejected (HTTP 503).
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?agent_id=agent-overflow"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err := websocket.Dial(ctx, url, nil)
	if err == nil {
		t.Fatal("expected connection to be rejected when at max capacity")
	}
	// The error message should indicate a non-101 status.
	if !strings.Contains(err.Error(), "503") {
		t.Logf("rejection error (might not contain 503 explicitly): %v", err)
	}

	// Disconnect one agent and try again — it should succeed now.
	_ = conns[0].Close(websocket.StatusNormalClosure, "")
	waitForAgentGone(t, hub, "agent-0", 2*time.Second)

	replacement := dialHub(t, srv, "agent-replacement")
	defer replacement.Close(websocket.StatusNormalClosure, "")
	waitForAgent(t, hub, "agent-replacement", 2*time.Second)

	if hub.ConnectedAgents() != maxConns {
		t.Errorf("ConnectedAgents = %d, want %d after replacement", hub.ConnectedAgents(), maxConns)
	}

	// Cleanup.
	for i := 1; i < maxConns; i++ {
		_ = conns[i].Close(websocket.StatusNormalClosure, "")
	}
}

func TestHubMaxConnections_ReconnectSameAgent(t *testing.T) {
	// When an already-connected agent reconnects, it should be allowed even
	// when at the connection limit (the old connection gets replaced).
	const maxConns = 2
	hub := NewHub(slog.Default(), nil, maxConns)
	srv := newTestServer(hub)

	conn1 := dialHub(t, srv, "agent-A")
	waitForAgent(t, hub, "agent-A", 2*time.Second)

	conn2 := dialHub(t, srv, "agent-B")
	waitForAgent(t, hub, "agent-B", 2*time.Second)

	if hub.ConnectedAgents() != 2 {
		t.Fatalf("ConnectedAgents = %d, want 2", hub.ConnectedAgents())
	}

	// agent-A reconnects — this should replace the existing connection, not be rejected.
	conn1New := dialHub(t, srv, "agent-A")

	// Give the hub a moment to process the replacement.
	time.Sleep(50 * time.Millisecond)

	if !hub.HasAgent("agent-A") {
		t.Fatal("agent-A should still be connected after reconnect")
	}
	if hub.ConnectedAgents() != 2 {
		t.Errorf("ConnectedAgents = %d, want 2 after reconnect", hub.ConnectedAgents())
	}

	// Cleanup: close everything and shut down the server to avoid
	// waiting for the old conn1 read goroutine to time out.
	_ = conn1.Close(websocket.StatusNormalClosure, "")
	_ = conn2.Close(websocket.StatusNormalClosure, "")
	_ = conn1New.Close(websocket.StatusNormalClosure, "")
	srv.Close()
}

func TestLocalBrokerPublish(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	broker := NewLocalBroker(hub)
	hub.SetBroker(broker)

	srv := newTestServer(hub)
	defer srv.Close()

	// Connect a receiver.
	receiver := dialHub(t, srv, "target-agent")
	defer receiver.Close(websocket.StatusNormalClosure, "")
	waitForAgent(t, hub, "target-agent", 2*time.Second)

	// Publish a message through the broker.
	msg := signaling.SignalMessage{
		Type: signaling.MessageTypeAnswer,
		From: "remote-agent",
		To:   "target-agent",
		SDP:  "answer-sdp",
	}
	if err := broker.Publish(context.Background(), msg); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// The receiver should get the message.
	got := readSignalMessage(t, receiver, 2*time.Second)
	if got.Type != signaling.MessageTypeAnswer {
		t.Errorf("Type = %q, want %q", got.Type, signaling.MessageTypeAnswer)
	}
	if got.From != "remote-agent" {
		t.Errorf("From = %q, want %q", got.From, "remote-agent")
	}
	if got.SDP != "answer-sdp" {
		t.Errorf("SDP = %q, want %q", got.SDP, "answer-sdp")
	}
}

func TestLocalBrokerSubscribe(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	broker := NewLocalBroker(hub)

	ch, err := broker.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if ch != nil {
		t.Error("LocalBroker.Subscribe should return nil channel")
	}
}

func TestLocalBrokerClose(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	broker := NewLocalBroker(hub)

	if err := broker.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestHubForwardViaBroker(t *testing.T) {
	// Verify that when a broker is set, Forward() delegates to it.
	hub := NewHub(slog.Default(), nil, 0)

	var published []signaling.SignalMessage
	var mu sync.Mutex
	broker := &recordingBroker{
		onPublish: func(msg signaling.SignalMessage) {
			mu.Lock()
			published = append(published, msg)
			mu.Unlock()
		},
	}
	hub.SetBroker(broker)

	msg := signaling.SignalMessage{
		Type: signaling.MessageTypeICECandidate,
		From: "agent-1",
		To:   "agent-2",
		Candidate: "candidate:1 1 udp 2122260223 10.0.0.1 12345 typ host",
	}
	hub.Forward(context.Background(), msg)

	mu.Lock()
	defer mu.Unlock()
	if len(published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(published))
	}
	if published[0].From != "agent-1" {
		t.Errorf("From = %q, want agent-1", published[0].From)
	}
	if published[0].To != "agent-2" {
		t.Errorf("To = %q, want agent-2", published[0].To)
	}
}

func TestHubDeliverEnvelope(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	srv := newTestServer(hub)
	defer srv.Close()

	receiver := dialHub(t, srv, "envelope-agent")
	defer receiver.Close(websocket.StatusNormalClosure, "")
	waitForAgent(t, hub, "envelope-agent", 2*time.Second)

	payload := json.RawMessage(`{"task_id":"t1","data":"hello"}`)
	err := hub.DeliverEnvelope(context.Background(), "envelope-agent", payload)
	if err != nil {
		t.Fatalf("DeliverEnvelope: %v", err)
	}

	got := readSignalMessage(t, receiver, 2*time.Second)
	if got.Type != signaling.MessageTypeBridgeMessage {
		t.Errorf("Type = %q, want %q", got.Type, signaling.MessageTypeBridgeMessage)
	}
	if got.To != "envelope-agent" {
		t.Errorf("To = %q, want envelope-agent", got.To)
	}
	if string(got.Payload) != string(payload) {
		t.Errorf("Payload = %s, want %s", got.Payload, payload)
	}
}

func TestHubDeliverEnvelope_NotConnected(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)

	err := hub.DeliverEnvelope(context.Background(), "nobody", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error when delivering to disconnected agent")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %q, expected to contain 'not connected'", err.Error())
	}
}

func TestHubCloseAll(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)

	// Verify CloseAll removes all entries from the connection map.
	// We avoid creating multiple real WebSocket connections because the
	// nhooyr.io/websocket v1 library has a 5-second per-connection close
	// handshake timeout that would make the test very slow.
	srv := newTestServer(hub)
	defer srv.Close()

	conn := dialHub(t, srv, "agent-close")
	waitForAgent(t, hub, "agent-close", 2*time.Second)

	if hub.ConnectedAgents() != 1 {
		t.Fatalf("ConnectedAgents = %d, want 1", hub.ConnectedAgents())
	}

	// CloseAll should clear the map synchronously.
	hub.CloseAll()

	if hub.ConnectedAgents() != 0 {
		t.Errorf("ConnectedAgents = %d, want 0 after CloseAll", hub.ConnectedAgents())
	}
	if hub.HasAgent("agent-close") {
		t.Error("HasAgent should return false after CloseAll")
	}

	// Close client side so the server handler goroutine exits promptly.
	_ = conn.Close(websocket.StatusNormalClosure, "")
}

func TestHubTURNConfigDelivery(t *testing.T) {
	turn := &TURNConfig{
		URLs:       []string{"turn:my-turn.example.com:3478"},
		Username:   "turnuser",
		Credential: "turnpass",
	}
	hub := NewHub(slog.Default(), turn, 0)
	srv := newTestServer(hub)
	defer srv.Close()

	conn := dialHub(t, srv, "turn-agent")
	defer conn.Close(websocket.StatusNormalClosure, "")

	// The first message should be the ICE config.
	got := readSignalMessage(t, conn, 2*time.Second)
	if got.Type != signaling.MessageTypeConfig {
		t.Fatalf("Type = %q, want %q", got.Type, signaling.MessageTypeConfig)
	}
	if len(got.ICEServers) != 1 {
		t.Fatalf("ICEServers length = %d, want 1", len(got.ICEServers))
	}
	ice := got.ICEServers[0]
	if len(ice.URLs) != 1 || ice.URLs[0] != "turn:my-turn.example.com:3478" {
		t.Errorf("ICEServers URLs = %v, want [turn:my-turn.example.com:3478]", ice.URLs)
	}
	if ice.Username != "turnuser" {
		t.Errorf("Username = %q, want turnuser", ice.Username)
	}
	if ice.Credential != "turnpass" {
		t.Errorf("Credential = %q, want turnpass", ice.Credential)
	}
}

func TestHubHandleConnect_MissingAgentID(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	srv := newTestServer(hub)
	defer srv.Close()

	// Attempt to connect without agent_id parameter.
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err := websocket.Dial(ctx, url, nil)
	if err == nil {
		t.Fatal("expected error when connecting without agent_id")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Logf("rejection error: %v", err)
	}
}

func TestHubConcurrentAccess(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	srv := newTestServer(hub)
	defer srv.Close()

	// Concurrently connect many agents and verify correctness.
	const n = 10
	var wg sync.WaitGroup
	conns := make([]*websocket.Conn, n)
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("concurrent-agent-%d", idx)
			url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?agent_id=" + id
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			c, _, err := websocket.Dial(ctx, url, nil)
			if err != nil {
				errs[idx] = err
				return
			}
			conns[idx] = c
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("agent concurrent-agent-%d failed to connect: %v", i, err)
		}
	}

	// Wait for all to register.
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("concurrent-agent-%d", i)
		waitForAgent(t, hub, id, 2*time.Second)
	}

	if hub.ConnectedAgents() != n {
		t.Errorf("ConnectedAgents = %d, want %d", hub.ConnectedAgents(), n)
	}

	// Concurrent HasAgent calls.
	var wg2 sync.WaitGroup
	for i := 0; i < n; i++ {
		wg2.Add(1)
		go func(idx int) {
			defer wg2.Done()
			id := fmt.Sprintf("concurrent-agent-%d", idx)
			if !hub.HasAgent(id) {
				t.Errorf("HasAgent(%q) = false, want true", id)
			}
		}(i)
	}
	wg2.Wait()

	// Cleanup.
	for _, c := range conns {
		if c != nil {
			_ = c.Close(websocket.StatusNormalClosure, "")
		}
	}
}

func TestMessageLimiter(t *testing.T) {
	// Burst of 5, rate 10/s.
	lim := newMessageLimiter(10, 5)

	// Should allow the full burst.
	for i := 0; i < 5; i++ {
		if !lim.allow() {
			t.Errorf("allow() = false at burst position %d", i)
		}
	}

	// Next should be denied (burst exhausted).
	if lim.allow() {
		t.Error("allow() = true after burst exhausted, want false")
	}

	// After waiting, tokens should replenish.
	time.Sleep(150 * time.Millisecond) // ~1.5 tokens at 10/s
	if !lim.allow() {
		t.Error("allow() = false after replenishment")
	}
}

func TestHubForward_ContactsWhitelistBlocks(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	srv := newTestServer(hub)
	defer srv.Close()

	// Set up a contacts checker that blocks all signaling.
	hub.SetContacts(&mockContacts{allowed: false})

	sender := dialHub(t, srv, "sender")
	defer sender.Close(websocket.StatusNormalClosure, "")
	receiver := dialHub(t, srv, "receiver")
	defer receiver.Close(websocket.StatusNormalClosure, "")

	waitForAgent(t, hub, "sender", 2*time.Second)
	waitForAgent(t, hub, "receiver", 2*time.Second)

	// Send an offer from sender to receiver (via the websocket).
	msg := signaling.SignalMessage{
		Type: signaling.MessageTypeOffer,
		From: "sender",
		To:   "receiver",
		SDP:  "test-sdp-offer",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := sender.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("sender write: %v", err)
	}

	// The receiver should NOT get the message (blocked by contacts).
	readCtx, readCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer readCancel()
	_, _, readErr := receiver.Read(readCtx)
	if readErr == nil {
		t.Error("receiver should not receive message when contacts check blocks it")
	}
}

func TestHubForward_ContactsWhitelistAllows(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	srv := newTestServer(hub)
	defer srv.Close()

	// Set up a contacts checker that allows all signaling.
	hub.SetContacts(&mockContacts{allowed: true})

	sender := dialHub(t, srv, "sender2")
	defer sender.Close(websocket.StatusNormalClosure, "")
	receiver := dialHub(t, srv, "receiver2")
	defer receiver.Close(websocket.StatusNormalClosure, "")

	waitForAgent(t, hub, "sender2", 2*time.Second)
	waitForAgent(t, hub, "receiver2", 2*time.Second)

	msg := signaling.SignalMessage{
		Type: signaling.MessageTypeOffer,
		From: "sender2",
		To:   "receiver2",
		SDP:  "test-sdp-offer",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := sender.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("sender write: %v", err)
	}

	// The receiver should get the message.
	got := readSignalMessage(t, receiver, 2*time.Second)
	if got.Type != signaling.MessageTypeOffer {
		t.Errorf("Type = %q, want %q", got.Type, signaling.MessageTypeOffer)
	}
	if got.SDP != "test-sdp-offer" {
		t.Errorf("SDP = %q, want %q", got.SDP, "test-sdp-offer")
	}
}

func TestHubForward_ContactsWhitelistSkipsNonSignaling(t *testing.T) {
	hub := NewHub(slog.Default(), nil, 0)
	srv := newTestServer(hub)
	defer srv.Close()

	// Contacts checker that blocks everything — but bridge_message should bypass it.
	hub.SetContacts(&mockContacts{allowed: false})

	sender := dialHub(t, srv, "sender3")
	defer sender.Close(websocket.StatusNormalClosure, "")
	receiver := dialHub(t, srv, "receiver3")
	defer receiver.Close(websocket.StatusNormalClosure, "")

	waitForAgent(t, hub, "sender3", 2*time.Second)
	waitForAgent(t, hub, "receiver3", 2*time.Second)

	// Send a bridge_message (not a signaling type) — should NOT be blocked.
	msg := signaling.SignalMessage{
		Type:    signaling.MessageTypeBridgeMessage,
		From:    "sender3",
		To:      "receiver3",
		Payload: json.RawMessage(`{"data":"hello"}`),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := sender.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("sender write: %v", err)
	}

	// The receiver SHOULD get this message since bridge_message is not a signaling type.
	got := readSignalMessage(t, receiver, 2*time.Second)
	if got.Type != signaling.MessageTypeBridgeMessage {
		t.Errorf("Type = %q, want %q", got.Type, signaling.MessageTypeBridgeMessage)
	}
}

// ---------- test doubles ----------

// mockContacts implements ContactsChecker for testing.
type mockContacts struct {
	allowed bool
}

func (m *mockContacts) IsAllowed(_ context.Context, _, _ string) (bool, error) {
	return m.allowed, nil
}

// recordingBroker is a Broker that records published messages.
type recordingBroker struct {
	onPublish func(signaling.SignalMessage)
}

func (b *recordingBroker) Publish(_ context.Context, msg signaling.SignalMessage) error {
	if b.onPublish != nil {
		b.onPublish(msg)
	}
	return nil
}

func (b *recordingBroker) Subscribe(_ context.Context) (<-chan signaling.SignalMessage, error) {
	return nil, nil
}

func (b *recordingBroker) Close() error {
	return nil
}
