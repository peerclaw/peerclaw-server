package registry

import (
	"context"
	"testing"

	"github.com/peerclaw/peerclaw-go/agentcard"
	"github.com/peerclaw/peerclaw-go/protocol"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	store := newTestStore(t)
	return NewService(store, nil)
}

func TestService_Register(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	card, err := svc.Register(ctx, RegisterRequest{
		Name:         "TestAgent",
		Description:  "A test agent",
		Version:      "1.0.0",
		Capabilities: []string{"chat"},
		Endpoint:     agentcard.Endpoint{URL: "http://localhost:3000"},
		Protocols:    []protocol.Protocol{protocol.ProtocolA2A},
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if card.ID == "" {
		t.Error("expected non-empty ID")
	}
	if card.Status != agentcard.StatusOnline {
		t.Errorf("Status = %q, want %q", card.Status, agentcard.StatusOnline)
	}
}

func TestService_Register_Validation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	tests := []struct {
		name string
		req  RegisterRequest
	}{
		{"empty name", RegisterRequest{Endpoint: agentcard.Endpoint{URL: "http://x"}, Protocols: []protocol.Protocol{protocol.ProtocolA2A}}},
		{"empty endpoint", RegisterRequest{Name: "X", Protocols: []protocol.Protocol{protocol.ProtocolA2A}}},
		{"no protocols", RegisterRequest{Name: "X", Endpoint: agentcard.Endpoint{URL: "http://x"}}},
		{"invalid protocol", RegisterRequest{Name: "X", Endpoint: agentcard.Endpoint{URL: "http://x"}, Protocols: []protocol.Protocol{"invalid"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(ctx, tt.req)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestService_Deregister(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	card, _ := svc.Register(ctx, RegisterRequest{
		Name:      "TestAgent",
		Endpoint:  agentcard.Endpoint{URL: "http://localhost:3000"},
		Protocols: []protocol.Protocol{protocol.ProtocolA2A},
	})

	if err := svc.Deregister(ctx, card.ID); err != nil {
		t.Fatalf("Deregister: %v", err)
	}

	_, err := svc.GetAgent(ctx, card.ID)
	if err == nil {
		t.Fatal("expected error after deregister")
	}
}

func TestService_Heartbeat(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	card, _ := svc.Register(ctx, RegisterRequest{
		Name:      "TestAgent",
		Endpoint:  agentcard.Endpoint{URL: "http://localhost:3000"},
		Protocols: []protocol.Protocol{protocol.ProtocolA2A},
	})

	deadline, err := svc.Heartbeat(ctx, card.ID, agentcard.StatusOnline)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if deadline.IsZero() {
		t.Error("expected non-zero deadline")
	}
}

func TestService_Discover(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	svc.Register(ctx, RegisterRequest{
		Name:         "SearchAgent",
		Capabilities: []string{"search", "index"},
		Endpoint:     agentcard.Endpoint{URL: "http://localhost:3001"},
		Protocols:    []protocol.Protocol{protocol.ProtocolA2A},
	})
	svc.Register(ctx, RegisterRequest{
		Name:         "ChatAgent",
		Capabilities: []string{"chat"},
		Endpoint:     agentcard.Endpoint{URL: "http://localhost:3002"},
		Protocols:    []protocol.Protocol{protocol.ProtocolMCP},
	})

	agents, err := svc.Discover(ctx, []string{"search"}, "", 10)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("got %d agents, want 1", len(agents))
	}
}

func TestService_Discover_NoCapabilities(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Discover(context.Background(), nil, "", 10)
	if err == nil {
		t.Error("expected error for empty capabilities")
	}
}
