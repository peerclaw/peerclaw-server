package router

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/peerclaw/peerclaw-core/agentcard"
)

// Engine provides intelligent routing decisions based on capabilities,
// protocol compatibility, latency, and priority.
type Engine struct {
	table  *Table
	logger *slog.Logger
}

// NewEngine creates a new routing engine.
func NewEngine(table *Table, logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	return &Engine{table: table, logger: logger}
}

// ResolveOptions specifies criteria for route resolution.
type ResolveOptions struct {
	TargetID   string
	Protocol   string
	Capability string
}

// Resolve finds the best route to a target agent.
func (e *Engine) Resolve(opts ResolveOptions) (*RouteEntry, error) {
	if opts.TargetID == "" {
		return nil, fmt.Errorf("target agent ID is required")
	}

	routes := e.table.GetRoutes(opts.TargetID)
	if len(routes) == 0 {
		return nil, fmt.Errorf("no routes found for agent %s", opts.TargetID)
	}

	// Filter by protocol if specified.
	if opts.Protocol != "" {
		var filtered []RouteEntry
		for _, r := range routes {
			if r.Protocol == opts.Protocol {
				filtered = append(filtered, r)
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("no routes for agent %s with protocol %s", opts.TargetID, opts.Protocol)
		}
		routes = filtered
	}

	// Sort by priority (descending), then latency (ascending).
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Priority != routes[j].Priority {
			return routes[i].Priority > routes[j].Priority
		}
		return routes[i].LatencyMs < routes[j].LatencyMs
	})

	best := routes[0]
	e.logger.Debug("route resolved", "target", opts.TargetID, "protocol", best.Protocol, "latency_ms", best.LatencyMs)
	return &best, nil
}

// UpdateFromCard updates routing table entries from an agent card.
func (e *Engine) UpdateFromCard(card *agentcard.Card) {
	for _, p := range card.Protocols {
		e.table.AddRoute(RouteEntry{
			SourceID:  "gateway",
			TargetID:  card.ID,
			Protocol:  string(p),
			Endpoint:  card.Endpoint.URL,
			LatencyMs: 0,
			Priority:  card.PeerClaw.Priority,
		})
	}
}

// RemoveAgent removes all routes for an agent.
func (e *Engine) RemoveAgent(agentID string) {
	e.table.RemoveRoute(agentID)
}

// Table returns the underlying routing table.
func (e *Engine) Table() *Table {
	return e.table
}
