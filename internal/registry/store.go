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
	Verified    bool
	MinScore    float64
	Search      string
	SortBy      string // "reputation", "name", "registered_at"
	OwnerUserID string // Filter by owner user ID.
	Category    string // Filter by category slug.
	PlaygroundOnly     bool   // Only return agents with playground_enabled=true
	PublicOnly         bool   // Only return agents with visibility='public'
	IncludeOwnerUserID string // When PublicOnly is false, also include agents owned by this user ID
}

// ListResult holds a page of agents and pagination info.
type ListResult struct {
	Agents        []*agentcard.Card `json:"agents"`
	NextPageToken string            `json:"next_page_token,omitempty"`
	TotalCount    int               `json:"total_count"`
}

// AccessFlags holds security-critical flags for agent access control.
type AccessFlags struct {
	PlaygroundEnabled bool   `json:"playground_enabled"`
	Visibility        string `json:"visibility"` // "public" or "private"
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

	// UpdateMetadata merges the provided metadata keys into the agent's existing metadata.
	UpdateMetadata(ctx context.Context, id string, metadata map[string]string) error

	// FindByCapabilities returns agents that match any of the given capabilities.
	FindByCapabilities(ctx context.Context, capabilities []string, protocol string, maxResults int) ([]*agentcard.Card, error)

	// ListByOwner returns agents owned by a specific user.
	ListByOwner(ctx context.Context, userID string, filter ListFilter) (*ListResult, error)

	// GetDB returns the underlying *sql.DB for shared use by other modules.
	GetDB() interface{}

	// GetAccessFlags returns access control flags for an agent.
	GetAccessFlags(ctx context.Context, id string) (*AccessFlags, error)

	// GetAccessFlagsBatch returns access control flags for multiple agents.
	GetAccessFlagsBatch(ctx context.Context, ids []string) (map[string]*AccessFlags, error)

	// SetAccessFlags updates access control flags for an agent.
	SetAccessFlags(ctx context.Context, id string, flags *AccessFlags) error

	// Close releases resources.
	Close() error
}
