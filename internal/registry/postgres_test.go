//go:build integration

package registry

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/protocol"
)

func TestPostgresStore_CRUD(t *testing.T) {
	dsn := os.Getenv("PEERCLAW_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("set PEERCLAW_TEST_PG_DSN to run PostgreSQL integration tests")
	}

	store, err := NewPostgresStore(dsn)
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	card := &agentcard.Card{
		ID:           "test-pg-1",
		Name:         "PG Test Agent",
		Description:  "A PostgreSQL test agent",
		Version:      "1.0.0",
		Capabilities: []string{"chat", "search"},
		Endpoint: agentcard.Endpoint{
			URL:       "http://localhost:3000",
			Transport: protocol.TransportHTTP,
		},
		Protocols:     []protocol.Protocol{protocol.A2A},
		Status:        agentcard.StatusOnline,
		RegisteredAt:  now,
		LastHeartbeat: now,
	}

	// Put
	if err := store.Put(ctx, card); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Get
	got, err := store.Get(ctx, card.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != card.Name {
		t.Errorf("Name = %q, want %q", got.Name, card.Name)
	}
	if len(got.Capabilities) != 2 {
		t.Errorf("Capabilities = %v, want 2 items", got.Capabilities)
	}

	// List
	result, err := store.List(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.TotalCount < 1 {
		t.Errorf("TotalCount = %d, want >= 1", result.TotalCount)
	}

	// FindByCapabilities
	agents, err := store.FindByCapabilities(ctx, []string{"chat"}, "", 10)
	if err != nil {
		t.Fatalf("FindByCapabilities: %v", err)
	}
	if len(agents) < 1 {
		t.Error("FindByCapabilities returned 0 agents")
	}

	// UpdateHeartbeat
	if err := store.UpdateHeartbeat(ctx, card.ID, agentcard.StatusBusy); err != nil {
		t.Fatalf("UpdateHeartbeat: %v", err)
	}
	got, _ = store.Get(ctx, card.ID)
	if got.Status != agentcard.StatusBusy {
		t.Errorf("Status = %q, want %q", got.Status, agentcard.StatusBusy)
	}

	// Delete
	if err := store.Delete(ctx, card.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = store.Get(ctx, card.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
