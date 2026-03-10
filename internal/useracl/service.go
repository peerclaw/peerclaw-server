package useracl

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Service implements user ACL business logic.
type Service struct {
	store  Store
	logger *slog.Logger
}

// NewService creates a new user ACL service.
func NewService(store Store, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: store, logger: logger}
}

// SubmitRequest creates a new access request.
func (s *Service) SubmitRequest(ctx context.Context, agentID, userID, message string) (*AccessRequest, error) {
	if agentID == "" || userID == "" {
		return nil, fmt.Errorf("agent ID and user ID are required")
	}

	// Check for existing pending request.
	existing, _ := s.store.GetByAgentAndUser(ctx, agentID, userID)
	if existing != nil && existing.Status == "pending" {
		return nil, fmt.Errorf("a pending request already exists")
	}
	// If previously rejected, allow re-request.
	if existing != nil && existing.Status == "approved" {
		return nil, fmt.Errorf("access already approved")
	}

	now := time.Now().UTC()
	req := &AccessRequest{
		ID:        uuid.New().String(),
		AgentID:   agentID,
		UserID:    userID,
		Status:    "pending",
		Message:   message,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Create(ctx, req); err != nil {
		return nil, fmt.Errorf("create access request: %w", err)
	}

	s.logger.Info("access request submitted",
		"id", req.ID,
		"agent_id", agentID,
		"user_id", userID,
	)
	return req, nil
}

// Approve approves an access request with optional expiry.
func (s *Service) Approve(ctx context.Context, id string, expiresAt *time.Time) error {
	if err := s.store.UpdateStatus(ctx, id, "approved", "", expiresAt); err != nil {
		return fmt.Errorf("approve access request: %w", err)
	}
	s.logger.Info("access request approved", "id", id)
	return nil
}

// Reject rejects an access request with a reason.
func (s *Service) Reject(ctx context.Context, id, reason string) error {
	if err := s.store.UpdateStatus(ctx, id, "rejected", reason, nil); err != nil {
		return fmt.Errorf("reject access request: %w", err)
	}
	s.logger.Info("access request rejected", "id", id, "reason", reason)
	return nil
}

// Revoke deletes an approved access request (revokes access).
func (s *Service) Revoke(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("revoke access: %w", err)
	}
	s.logger.Info("access revoked", "id", id)
	return nil
}

// IsAllowed checks if a user has approved, non-expired access to an agent.
func (s *Service) IsAllowed(ctx context.Context, agentID, userID string) (bool, error) {
	return s.store.IsAllowed(ctx, agentID, userID)
}

// ListByAgent returns access requests for an agent, optionally filtered by status.
func (s *Service) ListByAgent(ctx context.Context, agentID, status string) ([]AccessRequest, error) {
	return s.store.ListByAgent(ctx, agentID, status)
}

// ListByUser returns all access requests for a user.
func (s *Service) ListByUser(ctx context.Context, userID string) ([]AccessRequest, error) {
	return s.store.ListByUser(ctx, userID)
}

// GetByID returns a single access request.
func (s *Service) GetByID(ctx context.Context, id string) (*AccessRequest, error) {
	return s.store.GetByID(ctx, id)
}

// GetByAgentAndUser returns the current access request for a specific user+agent pair.
func (s *Service) GetByAgentAndUser(ctx context.Context, agentID, userID string) (*AccessRequest, error) {
	return s.store.GetByAgentAndUser(ctx, agentID, userID)
}
