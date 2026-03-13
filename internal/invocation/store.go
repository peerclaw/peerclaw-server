package invocation

import (
	"context"
	"time"
)

// InvocationRecord represents a single agent invocation.
type InvocationRecord struct {
	ID           string    `json:"id"`
	AgentID      string    `json:"agent_id"`
	UserID       string    `json:"user_id,omitempty"`
	Protocol     string    `json:"protocol"`
	RequestBody  string    `json:"-"` // stored but not returned by default
	ResponseBody string    `json:"-"` // stored but not returned by default
	StatusCode   int       `json:"status_code"`
	DurationMs   int64     `json:"duration_ms"`
	Error        string    `json:"error,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// AgentInvocationStats holds aggregated invocation statistics.
type AgentInvocationStats struct {
	TotalCalls    int64   `json:"total_calls"`
	SuccessCalls  int64   `json:"success_calls"`
	ErrorCalls    int64   `json:"error_calls"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
	P95DurationMs float64 `json:"p95_duration_ms"`
}

// TimeSeriesPoint is a single data point in a time series.
type TimeSeriesPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	TotalCalls   int64     `json:"total_calls"`
	SuccessCalls int64     `json:"success_calls"`
	ErrorCalls   int64     `json:"error_calls"`
	AvgDurationMs float64  `json:"avg_duration_ms"`
}

// AgentCallStats holds per-agent call statistics for admin top-agents report.
type AgentCallStats struct {
	AgentID       string  `json:"agent_id"`
	AgentName     string  `json:"agent_name"`
	TotalCalls    int64   `json:"total_calls"`
	SuccessCalls  int64   `json:"success_calls"`
	ErrorCalls    int64   `json:"error_calls"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
}

// Store defines the persistence interface for invocation records.
type Store interface {
	// Insert records a new invocation.
	Insert(ctx context.Context, record *InvocationRecord) error

	// GetByID retrieves an invocation by ID.
	GetByID(ctx context.Context, id string) (*InvocationRecord, error)

	// ListByUser returns invocations for a user, ordered by most recent first.
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]InvocationRecord, int, error)

	// ListByAgent returns invocations for an agent.
	ListByAgent(ctx context.Context, agentID string, limit, offset int) ([]InvocationRecord, int, error)

	// AgentStats returns aggregated stats for an agent.
	AgentStats(ctx context.Context, agentID string, since time.Time) (*AgentInvocationStats, error)

	// AgentTimeSeries returns time-bucketed invocation data.
	AgentTimeSeries(ctx context.Context, agentID string, since time.Time, bucketMinutes int) ([]TimeSeriesPoint, error)

	// ProviderDashboardStats returns aggregated stats for all agents owned by a user.
	ProviderDashboardStats(ctx context.Context, ownerUserID string) (*AgentInvocationStats, error)

	// ListAll returns all invocations with optional agent/user filters.
	ListAll(ctx context.Context, agentID, userID string, limit, offset int) ([]InvocationRecord, int, error)

	// GlobalStats returns aggregated stats across all agents since the given time.
	GlobalStats(ctx context.Context, since time.Time) (*AgentInvocationStats, error)

	// GlobalTimeSeries returns time-bucketed invocation data across all agents.
	GlobalTimeSeries(ctx context.Context, since time.Time, bucketMinutes int) ([]TimeSeriesPoint, error)

	// TopAgents returns the top agents by call count since the given time.
	TopAgents(ctx context.Context, since time.Time, limit int) ([]AgentCallStats, error)

	// PruneInvocations deletes invocation records older than the given time.
	// Returns the number of deleted rows.
	PruneInvocations(ctx context.Context, olderThan time.Time) (int64, error)

	// CountInvocations returns the total number of invocations.
	CountInvocations(ctx context.Context) (int, error)

	// Migrate creates the required tables.
	Migrate(ctx context.Context) error

	// Close releases resources.
	Close() error
}
