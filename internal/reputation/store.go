package reputation

import (
	"context"
	"time"
)

// EventType represents the type of reputation event.
type EventType string

const (
	EventRegistration      EventType = "registration"
	EventHeartbeatSuccess  EventType = "heartbeat_success"
	EventHeartbeatMiss     EventType = "heartbeat_miss"
	EventVerificationPass  EventType = "verification_pass"
	EventVerificationFail  EventType = "verification_fail"
	EventBridgeSuccess     EventType = "bridge_success"
	EventBridgeError       EventType = "bridge_error"
	EventBridgeTimeout     EventType = "bridge_timeout"
	EventReviewPositive    EventType = "review_positive"
	EventReviewNegative    EventType = "review_negative"
)

// Event represents a single reputation event record.
type Event struct {
	ID         int64     `json:"id"`
	AgentID    string    `json:"agent_id"`
	EventType  EventType `json:"event_type"`
	Weight     float64   `json:"weight"`
	ScoreAfter float64   `json:"score_after"`
	Metadata   string    `json:"metadata,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// Store defines the persistence interface for reputation data.
type Store interface {
	// InsertEvent records a reputation event.
	InsertEvent(ctx context.Context, event *Event) error

	// GetScore returns the current reputation score for an agent.
	// Returns defaultScore if no events exist.
	GetScore(ctx context.Context, agentID string) (float64, int64, error)

	// ListEvents returns reputation events for an agent, ordered by most recent first.
	ListEvents(ctx context.Context, agentID string, limit int) ([]Event, error)

	// UpdateAgentReputation updates the cached reputation score on the agents table.
	UpdateAgentReputation(ctx context.Context, agentID string, score float64, eventCount int64) error

	// SetAgentVerified marks an agent as verified.
	SetAgentVerified(ctx context.Context, agentID string) error

	// UnsetAgentVerified removes the verified status from an agent.
	UnsetAgentVerified(ctx context.Context, agentID string) error

	// ListStaleOnlineAgents returns IDs of agents whose status is online but
	// whose last heartbeat is older than the given timeout.
	ListStaleOnlineAgents(ctx context.Context, timeout time.Duration) ([]string, error)

	// Migrate creates the required tables and columns.
	Migrate(ctx context.Context) error

	// Close releases resources.
	Close() error
}
