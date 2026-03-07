package review

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-server/internal/reputation"
)

// Service implements review and catalog business logic.
type Service struct {
	store      Store
	reputation *reputation.Engine
	logger     *slog.Logger
}

// NewService creates a new review service.
// The reputation engine may be nil if reputation tracking is not enabled.
func NewService(store Store, rep *reputation.Engine, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		store:      store,
		reputation: rep,
		logger:     logger,
	}
}

// SubmitReview creates or updates a review for an agent by a user.
func (s *Service) SubmitReview(ctx context.Context, agentID, userID string, rating int, comment string) (*Review, error) {
	if rating < 1 || rating > 5 {
		return nil, fmt.Errorf("rating must be between 1 and 5")
	}

	now := time.Now().UTC()
	review := &Review{
		ID:        uuid.New().String(),
		AgentID:   agentID,
		UserID:    userID,
		Rating:    rating,
		Comment:   comment,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.UpsertReview(ctx, review); err != nil {
		return nil, fmt.Errorf("upsert review: %w", err)
	}

	// Record reputation event if engine is available.
	if s.reputation != nil {
		if rating >= 4 {
			if err := s.reputation.RecordEvent(ctx, agentID, reputation.EventReviewPositive, ""); err != nil {
				s.logger.Warn("failed to record positive review event", "error", err, "agent_id", agentID)
			}
		} else if rating <= 2 {
			if err := s.reputation.RecordEvent(ctx, agentID, reputation.EventReviewNegative, ""); err != nil {
				s.logger.Warn("failed to record negative review event", "error", err, "agent_id", agentID)
			}
		}
	}

	s.logger.Info("review submitted", "agent_id", agentID, "user_id", userID, "rating", rating)
	return review, nil
}

// DeleteReview removes a review for an agent by a user.
func (s *Service) DeleteReview(ctx context.Context, agentID, userID string) error {
	return s.store.DeleteReview(ctx, agentID, userID)
}

// ListReviews returns reviews for an agent with pagination.
func (s *Service) ListReviews(ctx context.Context, agentID string, limit, offset int) ([]Review, int, error) {
	return s.store.ListReviews(ctx, agentID, limit, offset)
}

// GetSummary returns aggregate review statistics for an agent.
func (s *Service) GetSummary(ctx context.Context, agentID string) (*ReviewSummary, error) {
	return s.store.GetReviewSummary(ctx, agentID)
}

// SubmitReport creates a new abuse report.
func (s *Service) SubmitReport(ctx context.Context, reporterID, targetType, targetID, reason, details string) error {
	report := &AbuseReport{
		ID:         uuid.New().String(),
		ReporterID: reporterID,
		TargetType: targetType,
		TargetID:   targetID,
		Reason:     reason,
		Details:    details,
		Status:     "pending",
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.store.CreateReport(ctx, report); err != nil {
		return fmt.Errorf("create abuse report: %w", err)
	}

	s.logger.Info("abuse report submitted",
		"report_id", report.ID,
		"reporter_id", reporterID,
		"target_type", targetType,
		"target_id", targetID,
	)
	return nil
}

// ListCategories returns all categories.
func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	return s.store.ListCategories(ctx)
}

// ListReports returns abuse reports with optional status filter.
func (s *Service) ListReports(ctx context.Context, status string, limit, offset int) ([]AbuseReport, int, error) {
	return s.store.ListReports(ctx, status, limit, offset)
}

// GetReport retrieves a single abuse report by ID.
func (s *Service) GetReport(ctx context.Context, id string) (*AbuseReport, error) {
	return s.store.GetReport(ctx, id)
}

// UpdateReportStatus updates the status of an abuse report.
func (s *Service) UpdateReportStatus(ctx context.Context, id, status string) error {
	validStatuses := map[string]bool{"pending": true, "reviewed": true, "dismissed": true, "actioned": true}
	if !validStatuses[status] {
		return fmt.Errorf("invalid report status: %s", status)
	}
	return s.store.UpdateReportStatus(ctx, id, status)
}

// DeleteReport removes an abuse report by ID.
func (s *Service) DeleteReport(ctx context.Context, id string) error {
	return s.store.DeleteReport(ctx, id)
}

// CreateCategory creates a new category.
func (s *Service) CreateCategory(ctx context.Context, category *Category) error {
	if category.ID == "" {
		category.ID = uuid.New().String()
	}
	return s.store.CreateCategory(ctx, category)
}

// UpdateCategory updates an existing category.
func (s *Service) UpdateCategory(ctx context.Context, category *Category) error {
	return s.store.UpdateCategory(ctx, category)
}

// DeleteCategory removes a category by ID.
func (s *Service) DeleteCategory(ctx context.Context, id string) error {
	return s.store.DeleteCategory(ctx, id)
}

// CountReviews returns the total number of reviews.
func (s *Service) CountReviews(ctx context.Context) (int, error) {
	return s.store.CountReviews(ctx)
}

// CountReports returns the number of abuse reports, optionally filtered by status.
func (s *Service) CountReports(ctx context.Context, status string) (int, error) {
	return s.store.CountReports(ctx, status)
}
