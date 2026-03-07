package registry

import (
	"context"

	"github.com/peerclaw/peerclaw-core/agentcard"
)

// ListFilter specifies criteria for listing agents.
type ListFilter struct {
	Protocol   string
	Capability string
	Status     agentcard.AgentStatus
	PageSize   int
	PageToken  string
	// Public directory filters.
	Verified bool
	MinScore float64
	Search   string
	SortBy   string // "reputation", "name", "registered_at"
}

// ListResult holds a page of agents and pagination info.
type ListResult struct {
	Agents        []*agentcard.Card
	NextPageToken string
	TotalCount    int
}

// Store defines the persistence interface for agent registration data.
type Store interface {
	// Put inserts or updates an agent card.
	Put(ctx context.Context, card *agentcard.Card) error

	// Get retrieves an agent card by ID.
	Get(ctx context.Context, id string) (*agentcard.Card, error)

	// Delete removes an agent card by ID.
	Delete(ctx context.Context, id string) error

	// List returns agents matching the filter criteria.
	List(ctx context.Context, filter ListFilter) (*ListResult, error)

	// UpdateHeartbeat updates the heartbeat timestamp and status of an agent.
	UpdateHeartbeat(ctx context.Context, id string, status agentcard.AgentStatus) error

	// FindByCapabilities returns agents that match any of the given capabilities.
	FindByCapabilities(ctx context.Context, capabilities []string, protocol string, maxResults int) ([]*agentcard.Card, error)

	// GetDB returns the underlying *sql.DB for shared use by other modules.
	GetDB() interface{}

	// Close releases resources.
	Close() error
}
