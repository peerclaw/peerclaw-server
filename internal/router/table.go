package router

import (
	"sync"
	"time"
)

// RouteEntry represents a single route in the routing table.
type RouteEntry struct {
	SourceID  string    `json:"source_id"`
	TargetID  string    `json:"target_id"`
	Protocol  string    `json:"protocol"`
	Endpoint  string    `json:"endpoint"`
	LatencyMs int       `json:"latency_ms"`
	Priority  int       `json:"priority"`
	TTL       int       `json:"ttl,omitempty"`
	Signature string    `json:"signature,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateType represents the kind of route change.
type UpdateType int

const (
	UpdateAdd    UpdateType = 1
	UpdateRemove UpdateType = 2
	UpdateModify UpdateType = 3
)

// RouteUpdate represents a change to the routing table.
type RouteUpdate struct {
	Type      UpdateType
	Route     RouteEntry
	Timestamp time.Time
}

// Table maintains the routing table for all registered agents.
type Table struct {
	mu        sync.RWMutex
	routes    map[string][]RouteEntry // keyed by target agent ID
	updatedAt time.Time
	watchers  []chan RouteUpdate
}

// NewTable creates a new empty routing table.
func NewTable() *Table {
	return &Table{
		routes:    make(map[string][]RouteEntry),
		updatedAt: time.Now(),
	}
}

// AddRoute adds or updates a route entry.
func (t *Table) AddRoute(entry RouteEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()

	entry.UpdatedAt = time.Now()
	routes := t.routes[entry.TargetID]

	// Check for existing route with same source+protocol and update it.
	for i, r := range routes {
		if r.SourceID == entry.SourceID && r.Protocol == entry.Protocol {
			routes[i] = entry
			t.updatedAt = time.Now()
			t.notify(RouteUpdate{Type: UpdateModify, Route: entry, Timestamp: time.Now()})
			return
		}
	}

	t.routes[entry.TargetID] = append(routes, entry)
	t.updatedAt = time.Now()
	t.notify(RouteUpdate{Type: UpdateAdd, Route: entry, Timestamp: time.Now()})
}

// RemoveRoute removes all routes for a given target agent.
func (t *Table) RemoveRoute(targetID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	routes, exists := t.routes[targetID]
	if !exists {
		return
	}
	delete(t.routes, targetID)
	t.updatedAt = time.Now()

	for _, r := range routes {
		t.notify(RouteUpdate{Type: UpdateRemove, Route: r, Timestamp: time.Now()})
	}
}

// GetRoutes returns all routes for a specific target agent.
func (t *Table) GetRoutes(targetID string) []RouteEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	routes := t.routes[targetID]
	result := make([]RouteEntry, len(routes))
	copy(result, routes)
	return result
}

// AllRoutes returns a snapshot of the entire routing table.
func (t *Table) AllRoutes() []RouteEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var all []RouteEntry
	for _, routes := range t.routes {
		all = append(all, routes...)
	}
	return all
}

// UpdatedAt returns when the table was last modified.
func (t *Table) UpdatedAt() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.updatedAt
}

// Watch returns a channel that receives route updates.
func (t *Table) Watch() <-chan RouteUpdate {
	t.mu.Lock()
	defer t.mu.Unlock()

	ch := make(chan RouteUpdate, 64)
	t.watchers = append(t.watchers, ch)
	return ch
}

// Unwatch removes a watcher channel.
func (t *Table) Unwatch(ch <-chan RouteUpdate) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, w := range t.watchers {
		if w == ch {
			t.watchers = append(t.watchers[:i], t.watchers[i+1:]...)
			close(w)
			return
		}
	}
}

// notify sends an update to all watchers (must be called with lock held).
func (t *Table) notify(update RouteUpdate) {
	for _, ch := range t.watchers {
		select {
		case ch <- update:
		default:
			// Drop update if watcher is too slow.
		}
	}
}
