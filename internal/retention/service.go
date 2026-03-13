package retention

import (
	"context"
	"log/slog"
	"time"

	"github.com/peerclaw/peerclaw-server/internal/invocation"
	"github.com/peerclaw/peerclaw-server/internal/reputation"
	"github.com/peerclaw/peerclaw-server/internal/review"
)

// Config holds retention policy settings.
type Config struct {
	ReputationEventsDays int
	InvocationsDays      int
	AbuseReportsDays     int
}

// PruneResult holds the number of rows deleted from each table.
type PruneResult struct {
	ReputationEvents int64
	Invocations      int64
	AbuseReports     int64
}

// Service coordinates data retention cleanup across stores.
type Service struct {
	repStore reputation.Store
	invStore invocation.Store
	revStore review.Store
	config   Config
	logger   *slog.Logger
}

// NewService creates a new retention service. Any store may be nil.
func NewService(repStore reputation.Store, invStore invocation.Store, revStore review.Store, cfg Config, logger *slog.Logger) *Service {
	return &Service{
		repStore: repStore,
		invStore: invStore,
		revStore: revStore,
		config:   cfg,
		logger:   logger,
	}
}

// RunOnce executes a single cleanup pass, deleting expired data from all configured stores.
func (s *Service) RunOnce(ctx context.Context) (*PruneResult, error) {
	now := time.Now().UTC()
	result := &PruneResult{}

	if s.repStore != nil && s.config.ReputationEventsDays > 0 {
		cutoff := now.AddDate(0, 0, -s.config.ReputationEventsDays)
		n, err := s.repStore.PruneEvents(ctx, cutoff)
		if err != nil {
			s.logger.Error("retention: failed to prune reputation events", "error", err)
			return result, err
		}
		result.ReputationEvents = n
	}

	if s.invStore != nil && s.config.InvocationsDays > 0 {
		cutoff := now.AddDate(0, 0, -s.config.InvocationsDays)
		n, err := s.invStore.PruneInvocations(ctx, cutoff)
		if err != nil {
			s.logger.Error("retention: failed to prune invocations", "error", err)
			return result, err
		}
		result.Invocations = n
	}

	if s.revStore != nil && s.config.AbuseReportsDays > 0 {
		cutoff := now.AddDate(0, 0, -s.config.AbuseReportsDays)
		n, err := s.revStore.PruneResolvedReports(ctx, cutoff)
		if err != nil {
			s.logger.Error("retention: failed to prune abuse reports", "error", err)
			return result, err
		}
		result.AbuseReports = n
	}

	if result.ReputationEvents > 0 || result.Invocations > 0 || result.AbuseReports > 0 {
		s.logger.Info("retention cleanup completed",
			"reputation_events", result.ReputationEvents,
			"invocations", result.Invocations,
			"abuse_reports", result.AbuseReports,
		)
	}

	return result, nil
}
