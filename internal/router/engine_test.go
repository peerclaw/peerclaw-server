package router

import (
	"testing"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/protocol"
)

func TestTable_AddAndGetRoutes(t *testing.T) {
	table := NewTable()
	table.AddRoute(RouteEntry{
		SourceID:  "gateway",
		TargetID:  "agent-1",
		Protocol:  "a2a",
		Endpoint:  "http://localhost:3000",
		LatencyMs: 5,
		Priority:  10,
	})

	routes := table.GetRoutes("agent-1")
	if len(routes) != 1 {
		t.Fatalf("got %d routes, want 1", len(routes))
	}
	if routes[0].Endpoint != "http://localhost:3000" {
		t.Errorf("Endpoint = %q, want %q", routes[0].Endpoint, "http://localhost:3000")
	}
}

func TestTable_AddRoute_Update(t *testing.T) {
	table := NewTable()
	table.AddRoute(RouteEntry{
		SourceID: "gateway", TargetID: "agent-1", Protocol: "a2a",
		Endpoint: "http://old", LatencyMs: 10,
	})
	table.AddRoute(RouteEntry{
		SourceID: "gateway", TargetID: "agent-1", Protocol: "a2a",
		Endpoint: "http://new", LatencyMs: 5,
	})

	routes := table.GetRoutes("agent-1")
	if len(routes) != 1 {
		t.Fatalf("got %d routes, want 1 (should update in place)", len(routes))
	}
	if routes[0].Endpoint != "http://new" {
		t.Errorf("Endpoint = %q, want %q", routes[0].Endpoint, "http://new")
	}
}

func TestTable_RemoveRoute(t *testing.T) {
	table := NewTable()
	table.AddRoute(RouteEntry{SourceID: "gateway", TargetID: "agent-1", Protocol: "a2a"})
	table.RemoveRoute("agent-1")

	routes := table.GetRoutes("agent-1")
	if len(routes) != 0 {
		t.Errorf("got %d routes after remove, want 0", len(routes))
	}
}

func TestTable_AllRoutes(t *testing.T) {
	table := NewTable()
	table.AddRoute(RouteEntry{SourceID: "gateway", TargetID: "agent-1", Protocol: "a2a"})
	table.AddRoute(RouteEntry{SourceID: "gateway", TargetID: "agent-2", Protocol: "mcp"})

	all := table.AllRoutes()
	if len(all) != 2 {
		t.Errorf("got %d routes, want 2", len(all))
	}
}

func TestTable_Watch(t *testing.T) {
	table := NewTable()
	ch := table.Watch()

	table.AddRoute(RouteEntry{SourceID: "gateway", TargetID: "agent-1", Protocol: "a2a"})

	select {
	case update := <-ch:
		if update.Type != UpdateAdd {
			t.Errorf("Type = %d, want %d (UpdateAdd)", update.Type, UpdateAdd)
		}
	default:
		t.Error("expected route update, got none")
	}

	table.Unwatch(ch)
}

func TestEngine_Resolve(t *testing.T) {
	table := NewTable()
	engine := NewEngine(table, nil)

	table.AddRoute(RouteEntry{
		SourceID: "gateway", TargetID: "agent-1", Protocol: "a2a",
		Endpoint: "http://slow", LatencyMs: 100, Priority: 1,
	})
	table.AddRoute(RouteEntry{
		SourceID: "gateway", TargetID: "agent-1", Protocol: "mcp",
		Endpoint: "http://fast", LatencyMs: 5, Priority: 10,
	})

	route, err := engine.Resolve(ResolveOptions{TargetID: "agent-1"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// Should pick the higher-priority route.
	if route.Protocol != "mcp" {
		t.Errorf("Protocol = %q, want %q (higher priority)", route.Protocol, "mcp")
	}
}

func TestEngine_Resolve_WithProtocolFilter(t *testing.T) {
	table := NewTable()
	engine := NewEngine(table, nil)

	table.AddRoute(RouteEntry{
		SourceID: "gateway", TargetID: "agent-1", Protocol: "a2a",
		Endpoint: "http://a2a", Priority: 1,
	})
	table.AddRoute(RouteEntry{
		SourceID: "gateway", TargetID: "agent-1", Protocol: "mcp",
		Endpoint: "http://mcp", Priority: 10,
	})

	route, err := engine.Resolve(ResolveOptions{TargetID: "agent-1", Protocol: "a2a"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if route.Protocol != "a2a" {
		t.Errorf("Protocol = %q, want %q", route.Protocol, "a2a")
	}
}

func TestEngine_Resolve_NotFound(t *testing.T) {
	engine := NewEngine(NewTable(), nil)
	_, err := engine.Resolve(ResolveOptions{TargetID: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent target")
	}
}

func TestEngine_Resolve_NoTargetID(t *testing.T) {
	engine := NewEngine(NewTable(), nil)
	_, err := engine.Resolve(ResolveOptions{})
	if err == nil {
		t.Error("expected error for empty target ID")
	}
}

func TestEngine_UpdateFromCard(t *testing.T) {
	table := NewTable()
	engine := NewEngine(table, nil)

	card := &agentcard.Card{
		ID: "agent-1",
		Endpoint: agentcard.Endpoint{
			URL: "http://localhost:3000",
		},
		Protocols: []protocol.Protocol{protocol.ProtocolA2A, protocol.ProtocolMCP},
		PeerClaw:  agentcard.PeerClawExtension{Priority: 5},
	}

	engine.UpdateFromCard(card)

	routes := table.GetRoutes("agent-1")
	if len(routes) != 2 {
		t.Fatalf("got %d routes, want 2", len(routes))
	}
}

func TestEngine_RemoveAgent(t *testing.T) {
	table := NewTable()
	engine := NewEngine(table, nil)

	table.AddRoute(RouteEntry{SourceID: "gateway", TargetID: "agent-1", Protocol: "a2a"})
	engine.RemoveAgent("agent-1")

	routes := table.GetRoutes("agent-1")
	if len(routes) != 0 {
		t.Errorf("got %d routes after remove, want 0", len(routes))
	}
}
