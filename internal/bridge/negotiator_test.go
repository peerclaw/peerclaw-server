package bridge

import (
	"context"
	"testing"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
)

// stubBridge implements ProtocolBridge for testing.
type stubBridge struct {
	proto string
}

func (s *stubBridge) Protocol() string                                                       { return s.proto }
func (s *stubBridge) Send(_ context.Context, _ *envelope.Envelope) error                     { return nil }
func (s *stubBridge) Receive(_ context.Context) (<-chan *envelope.Envelope, error)            { return nil, nil }
func (s *stubBridge) Handshake(_ context.Context, _ *agentcard.Card) error                   { return nil }
func (s *stubBridge) Translate(_ context.Context, _ *envelope.Envelope, _ string) (*envelope.Envelope, error) {
	return nil, nil
}
func (s *stubBridge) Close() error { return nil }

func setupManager(protos ...string) *Manager {
	m := NewManager(nil)
	for _, p := range protos {
		m.RegisterBridge(&stubBridge{proto: p})
	}
	return m
}

func TestNegotiate_CommonProtocol(t *testing.T) {
	m := setupManager("a2a", "mcp")
	n := NewNegotiator(m, nil)

	source := &agentcard.Card{
		ID:        "agent-1",
		Protocols: []protocol.Protocol{protocol.ProtocolA2A, protocol.ProtocolMCP},
	}
	target := &agentcard.Card{
		ID:        "agent-2",
		Protocols: []protocol.Protocol{protocol.ProtocolA2A},
	}

	result, err := n.Negotiate(source, target)
	if err != nil {
		t.Fatalf("Negotiate: %v", err)
	}
	if result.Protocol != "a2a" {
		t.Errorf("Protocol = %q, want a2a", result.Protocol)
	}
	if result.NeedsTranslation {
		t.Error("should not need translation for common protocol")
	}
}

func TestNegotiate_MCPCommon(t *testing.T) {
	m := setupManager("a2a", "mcp")
	n := NewNegotiator(m, nil)

	source := &agentcard.Card{
		ID:        "agent-1",
		Protocols: []protocol.Protocol{protocol.ProtocolMCP},
	}
	target := &agentcard.Card{
		ID:        "agent-2",
		Protocols: []protocol.Protocol{protocol.ProtocolMCP},
	}

	result, err := n.Negotiate(source, target)
	if err != nil {
		t.Fatal(err)
	}
	if result.Protocol != "mcp" {
		t.Errorf("Protocol = %q", result.Protocol)
	}
	if result.NeedsTranslation {
		t.Error("should not need translation")
	}
}

func TestNegotiate_NeedsTranslation(t *testing.T) {
	m := setupManager("a2a", "mcp", "acp")
	n := NewNegotiator(m, nil)

	source := &agentcard.Card{
		ID:        "agent-1",
		Protocols: []protocol.Protocol{protocol.ProtocolA2A},
	}
	target := &agentcard.Card{
		ID:        "agent-2",
		Protocols: []protocol.Protocol{protocol.ProtocolMCP},
	}

	result, err := n.Negotiate(source, target)
	if err != nil {
		t.Fatal(err)
	}
	if !result.NeedsTranslation {
		t.Error("should need translation")
	}
	// Target protocol should be MCP (the target's protocol).
	if result.Protocol != "mcp" {
		t.Errorf("Protocol = %q, want mcp", result.Protocol)
	}
}

func TestNegotiate_ACPToA2A(t *testing.T) {
	m := setupManager("a2a", "acp")
	n := NewNegotiator(m, nil)

	source := &agentcard.Card{
		ID:        "agent-1",
		Protocols: []protocol.Protocol{protocol.ProtocolACP},
	}
	target := &agentcard.Card{
		ID:        "agent-2",
		Protocols: []protocol.Protocol{protocol.ProtocolA2A},
	}

	result, err := n.Negotiate(source, target)
	if err != nil {
		t.Fatal(err)
	}
	if !result.NeedsTranslation {
		t.Error("should need translation")
	}
	if result.Protocol != "a2a" {
		t.Errorf("Protocol = %q, want a2a", result.Protocol)
	}
}

func TestNegotiate_NoCompatiblePath(t *testing.T) {
	// Only A2A bridge available, but one agent only supports ACP.
	m := setupManager("a2a")
	n := NewNegotiator(m, nil)

	source := &agentcard.Card{
		ID:        "agent-1",
		Protocols: []protocol.Protocol{protocol.ProtocolA2A},
	}
	target := &agentcard.Card{
		ID:        "agent-2",
		Protocols: []protocol.Protocol{protocol.ProtocolACP},
	}

	_, err := n.Negotiate(source, target)
	if err == nil {
		t.Error("expected error for no compatible path")
	}
}

func TestNegotiate_NilCards(t *testing.T) {
	m := setupManager("a2a")
	n := NewNegotiator(m, nil)

	_, err := n.Negotiate(nil, &agentcard.Card{})
	if err == nil {
		t.Error("expected error for nil source")
	}
	_, err = n.Negotiate(&agentcard.Card{}, nil)
	if err == nil {
		t.Error("expected error for nil target")
	}
}

func TestNegotiate_PriorityOrder(t *testing.T) {
	// Both agents support A2A and MCP, A2A should be preferred.
	m := setupManager("a2a", "mcp")
	n := NewNegotiator(m, nil)

	source := &agentcard.Card{
		ID:        "agent-1",
		Protocols: []protocol.Protocol{protocol.ProtocolMCP, protocol.ProtocolA2A},
	}
	target := &agentcard.Card{
		ID:        "agent-2",
		Protocols: []protocol.Protocol{protocol.ProtocolMCP, protocol.ProtocolA2A},
	}

	result, err := n.Negotiate(source, target)
	if err != nil {
		t.Fatal(err)
	}
	// A2A has higher priority in protocolPriority.
	if result.Protocol != "a2a" {
		t.Errorf("Protocol = %q, want a2a (higher priority)", result.Protocol)
	}
}
