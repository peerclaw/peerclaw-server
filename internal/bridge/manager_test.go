package bridge

import (
	"context"
	"testing"

	"github.com/peerclaw/peerclaw-go/agentcard"
	"github.com/peerclaw/peerclaw-go/envelope"
	"github.com/peerclaw/peerclaw-go/protocol"
)

// mockBridge implements ProtocolBridge for testing.
type mockBridge struct {
	proto  string
	sent   []*envelope.Envelope
	closed bool
}

func (m *mockBridge) Protocol() string { return m.proto }
func (m *mockBridge) Send(_ context.Context, env *envelope.Envelope) error {
	m.sent = append(m.sent, env)
	return nil
}
func (m *mockBridge) Receive(_ context.Context) (<-chan *envelope.Envelope, error) {
	return make(chan *envelope.Envelope), nil
}
func (m *mockBridge) Handshake(_ context.Context, _ *agentcard.Card) error { return nil }
func (m *mockBridge) Translate(_ context.Context, env *envelope.Envelope, target string) (*envelope.Envelope, error) {
	out := *env
	out.Protocol = protocol.Protocol(target)
	return &out, nil
}
func (m *mockBridge) Close() error { m.closed = true; return nil }

func TestManager_RegisterAndGet(t *testing.T) {
	mgr := NewManager(nil)
	mock := &mockBridge{proto: "a2a"}

	if err := mgr.RegisterBridge(mock); err != nil {
		t.Fatalf("RegisterBridge: %v", err)
	}

	b, err := mgr.GetBridge("a2a")
	if err != nil {
		t.Fatalf("GetBridge: %v", err)
	}
	if b.Protocol() != "a2a" {
		t.Errorf("Protocol = %q, want %q", b.Protocol(), "a2a")
	}
}

func TestManager_RegisterDuplicate(t *testing.T) {
	mgr := NewManager(nil)
	mgr.RegisterBridge(&mockBridge{proto: "a2a"})

	err := mgr.RegisterBridge(&mockBridge{proto: "a2a"})
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestManager_GetBridge_NotFound(t *testing.T) {
	mgr := NewManager(nil)
	_, err := mgr.GetBridge("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent bridge")
	}
}

func TestManager_Send(t *testing.T) {
	mgr := NewManager(nil)
	mock := &mockBridge{proto: "a2a"}
	mgr.RegisterBridge(mock)

	env := envelope.New("src", "dst", protocol.ProtocolA2A, []byte("{}"))
	if err := mgr.Send(context.Background(), env); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(mock.sent) != 1 {
		t.Errorf("sent count = %d, want 1", len(mock.sent))
	}
}

func TestManager_Translate(t *testing.T) {
	mgr := NewManager(nil)
	mgr.RegisterBridge(&mockBridge{proto: "a2a"})

	env := envelope.New("src", "dst", protocol.ProtocolA2A, []byte("{}"))
	translated, err := mgr.Translate(context.Background(), env, "mcp")
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if translated.Protocol != protocol.ProtocolMCP {
		t.Errorf("Protocol = %q, want %q", translated.Protocol, protocol.ProtocolMCP)
	}
}

func TestManager_Handshake(t *testing.T) {
	mgr := NewManager(nil)
	mgr.RegisterBridge(&mockBridge{proto: "a2a"})

	card := &agentcard.Card{
		ID:        "agent-1",
		Protocols: []protocol.Protocol{protocol.ProtocolA2A},
	}
	if err := mgr.Handshake(context.Background(), card); err != nil {
		t.Fatalf("Handshake: %v", err)
	}
}

func TestManager_Handshake_NoProtocol(t *testing.T) {
	mgr := NewManager(nil)
	card := &agentcard.Card{ID: "agent-1"}
	err := mgr.Handshake(context.Background(), card)
	if err == nil {
		t.Error("expected error for agent with no protocols")
	}
}

func TestManager_ListBridges(t *testing.T) {
	mgr := NewManager(nil)
	mgr.RegisterBridge(&mockBridge{proto: "a2a"})
	mgr.RegisterBridge(&mockBridge{proto: "mcp"})

	infos := mgr.ListBridges()
	if len(infos) != 2 {
		t.Errorf("got %d bridges, want 2", len(infos))
	}
}

func TestManager_Close(t *testing.T) {
	mgr := NewManager(nil)
	mock := &mockBridge{proto: "a2a"}
	mgr.RegisterBridge(mock)

	if err := mgr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !mock.closed {
		t.Error("expected bridge to be closed")
	}
}
