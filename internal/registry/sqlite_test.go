package registry

import (
	"context"
	"testing"
	"time"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/protocol"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func testCard() *agentcard.Card {
	now := time.Now().UTC()
	return &agentcard.Card{
		ID:           "agent-1",
		Name:         "TestAgent",
		Description:  "A test agent",
		Version:      "1.0.0",
		Capabilities: []string{"chat", "search"},
		Endpoint: agentcard.Endpoint{
			URL:       "http://localhost:3000",
			Host:      "localhost",
			Port:      3000,
			Transport: protocol.TransportHTTP,
		},
		Protocols: []protocol.Protocol{protocol.ProtocolA2A, protocol.ProtocolMCP},
		Auth: agentcard.AuthInfo{
			Type:   "bearer",
			Params: map[string]string{"token_url": "http://localhost/token"},
		},
		Metadata: map[string]string{"env": "test"},
		PeerClaw: agentcard.PeerClawExtension{
			NATType:         "none",
			RelayPreference: "direct",
			Priority:        10,
			Tags:            []string{"test"},
		},
		Status:        agentcard.StatusOnline,
		RegisteredAt:  now,
		LastHeartbeat: now,
	}
}

func TestSQLiteStore_PutAndGet(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	card := testCard()

	if err := store.Put(ctx, card); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := store.Get(ctx, card.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.ID != card.ID {
		t.Errorf("ID = %q, want %q", got.ID, card.ID)
	}
	if got.Name != card.Name {
		t.Errorf("Name = %q, want %q", got.Name, card.Name)
	}
	if len(got.Capabilities) != 2 {
		t.Errorf("Capabilities count = %d, want 2", len(got.Capabilities))
	}
	if len(got.Protocols) != 2 {
		t.Errorf("Protocols count = %d, want 2", len(got.Protocols))
	}
	if got.PeerClaw.Priority != 10 {
		t.Errorf("Priority = %d, want 10", got.PeerClaw.Priority)
	}
}

func TestSQLiteStore_Delete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	card := testCard()

	store.Put(ctx, card)

	if err := store.Delete(ctx, card.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Get(ctx, card.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestSQLiteStore_Delete_NotFound(t *testing.T) {
	store := newTestStore(t)
	err := store.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent agent, got nil")
	}
}

func TestSQLiteStore_List(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := range 3 {
		card := testCard()
		card.ID = string(rune('a'+i)) + "-agent"
		card.Name = card.ID
		store.Put(ctx, card)
	}

	result, err := store.List(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", result.TotalCount)
	}
	if len(result.Agents) != 3 {
		t.Errorf("Agents count = %d, want 3", len(result.Agents))
	}
}

func TestSQLiteStore_List_Filter(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	card1 := testCard()
	card1.ID = "agent-a2a"
	card1.Protocols = []protocol.Protocol{protocol.ProtocolA2A}
	store.Put(ctx, card1)

	card2 := testCard()
	card2.ID = "agent-mcp"
	card2.Protocols = []protocol.Protocol{protocol.ProtocolMCP}
	store.Put(ctx, card2)

	result, err := store.List(ctx, ListFilter{Protocol: "mcp"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", result.TotalCount)
	}
}

func TestSQLiteStore_UpdateHeartbeat(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	card := testCard()
	store.Put(ctx, card)

	err := store.UpdateHeartbeat(ctx, card.ID, agentcard.StatusDegraded)
	if err != nil {
		t.Fatalf("UpdateHeartbeat: %v", err)
	}

	got, _ := store.Get(ctx, card.ID)
	if got.Status != agentcard.StatusDegraded {
		t.Errorf("Status = %q, want %q", got.Status, agentcard.StatusDegraded)
	}
}

func TestSQLiteStore_FindByCapabilities(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	card1 := testCard()
	card1.ID = "search-agent"
	card1.Capabilities = []string{"search", "index"}
	store.Put(ctx, card1)

	card2 := testCard()
	card2.ID = "chat-agent"
	card2.Capabilities = []string{"chat"}
	store.Put(ctx, card2)

	agents, err := store.FindByCapabilities(ctx, []string{"search"}, "", 10)
	if err != nil {
		t.Fatalf("FindByCapabilities: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("got %d agents, want 1", len(agents))
	}
	if len(agents) > 0 && agents[0].ID != "search-agent" {
		t.Errorf("ID = %q, want %q", agents[0].ID, "search-agent")
	}
}

func TestSQLiteStore_Upsert(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	card := testCard()

	store.Put(ctx, card)

	card.Name = "UpdatedAgent"
	card.Version = "2.0.0"
	if err := store.Put(ctx, card); err != nil {
		t.Fatalf("Put (upsert): %v", err)
	}

	got, _ := store.Get(ctx, card.ID)
	if got.Name != "UpdatedAgent" {
		t.Errorf("Name = %q, want %q", got.Name, "UpdatedAgent")
	}
	if got.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", got.Version, "2.0.0")
	}
}
