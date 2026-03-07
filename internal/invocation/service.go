package invocation

import (
	"context"
	"log/slog"
	"time"
)

// Service implements invocation business logic.
type Service struct {
	store  Store
	logger *slog.Logger
}

// NewService creates a new invocation service.
func NewService(store Store, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: store, logger: logger}
}

// Record saves an invocation record.
func (s *Service) Record(ctx context.Context, record *InvocationRecord) error {
	return s.store.Insert(ctx, record)
}

// GetByID retrieves an invocation by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*InvocationRecord, error) {
	return s.store.GetByID(ctx, id)
}

// ListByUser returns invocations for a user.
func (s *Service) ListByUser(ctx context.Context, userID string, limit, offset int) ([]InvocationRecord, int, error) {
	return s.store.ListByUser(ctx, userID, limit, offset)
}

// ListByAgent returns invocations for an agent.
func (s *Service) ListByAgent(ctx context.Context, agentID string, limit, offset int) ([]InvocationRecord, int, error) {
	return s.store.ListByAgent(ctx, agentID, limit, offset)
}

// AgentStats returns stats for an agent.
func (s *Service) AgentStats(ctx context.Context, agentID string, since time.Time) (*AgentInvocationStats, error) {
	return s.store.AgentStats(ctx, agentID, since)
}

// AgentTimeSeries returns time series data for an agent.
func (s *Service) AgentTimeSeries(ctx context.Context, agentID string, since time.Time, bucketMinutes int) ([]TimeSeriesPoint, error) {
	return s.store.AgentTimeSeries(ctx, agentID, since, bucketMinutes)
}

// ProviderDashboardStats returns aggregated stats for a provider.
func (s *Service) ProviderDashboardStats(ctx context.Context, ownerUserID string) (*AgentInvocationStats, error) {
	return s.store.ProviderDashboardStats(ctx, ownerUserID)
}

// ListAll returns all invocations with optional agent/user filters.
func (s *Service) ListAll(ctx context.Context, agentID, userID string, limit, offset int) ([]InvocationRecord, int, error) {
	return s.store.ListAll(ctx, agentID, userID, limit, offset)
}

// GlobalStats returns aggregated stats across all agents.
func (s *Service) GlobalStats(ctx context.Context, since time.Time) (*AgentInvocationStats, error) {
	return s.store.GlobalStats(ctx, since)
}

// GlobalTimeSeries returns time-bucketed invocation data across all agents.
func (s *Service) GlobalTimeSeries(ctx context.Context, since time.Time, bucketMinutes int) ([]TimeSeriesPoint, error) {
	return s.store.GlobalTimeSeries(ctx, since, bucketMinutes)
}

// TopAgents returns the top agents by call count.
func (s *Service) TopAgents(ctx context.Context, since time.Time, limit int) ([]AgentCallStats, error) {
	return s.store.TopAgents(ctx, since, limit)
}

// CountInvocations returns the total number of invocations.
func (s *Service) CountInvocations(ctx context.Context) (int, error) {
	return s.store.CountInvocations(ctx)
}
