package contactreq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-server/internal/contacts"
)

// Service implements contact request business logic.
type Service struct {
	store    Store
	contacts *contacts.Service
	logger   *slog.Logger
}

// NewService creates a new contact request service.
func NewService(store Store, contacts *contacts.Service, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: store, contacts: contacts, logger: logger}
}

// Submit creates a new contact request from one agent to another.
func (s *Service) Submit(ctx context.Context, fromAgentID, toAgentID, message string) (*ContactRequest, error) {
	if fromAgentID == "" || toAgentID == "" {
		return nil, fmt.Errorf("from and to agent IDs are required")
	}
	if fromAgentID == toAgentID {
		return nil, fmt.Errorf("cannot send contact request to self")
	}

	// Check for existing pending request.
	existing, _ := s.store.GetByAgents(ctx, fromAgentID, toAgentID)
	if existing != nil && existing.Status == "pending" {
		return nil, fmt.Errorf("a pending contact request already exists")
	}
	if existing != nil && existing.Status == "approved" {
		return nil, fmt.Errorf("contact request already approved")
	}

	now := time.Now().UTC()
	req := &ContactRequest{
		ID:          uuid.New().String(),
		FromAgentID: fromAgentID,
		ToAgentID:   toAgentID,
		Status:      "pending",
		Message:     message,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Create(ctx, req); err != nil {
		return nil, fmt.Errorf("create contact request: %w", err)
	}

	s.logger.Info("contact request submitted",
		"id", req.ID,
		"from", fromAgentID,
		"to", toAgentID,
	)
	return req, nil
}

// Approve approves a contact request and adds both agents as contacts of each other.
// Returns the approved request so callers can notify both agents.
func (s *Service) Approve(ctx context.Context, id string) (*ContactRequest, error) {
	req, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get contact request: %w", err)
	}
	if req == nil {
		return nil, fmt.Errorf("contact request not found")
	}
	if req.Status != "pending" {
		return nil, fmt.Errorf("contact request is not pending (status: %s)", req.Status)
	}

	if err := s.store.UpdateStatus(ctx, id, "approved", ""); err != nil {
		return nil, fmt.Errorf("approve contact request: %w", err)
	}

	// Bidirectional contact addition.
	if s.contacts != nil {
		if _, err := s.contacts.Add(ctx, req.ToAgentID, req.FromAgentID, "", nil); err != nil {
			s.logger.Warn("failed to add contact (to→from)", "error", err)
		}
		if _, err := s.contacts.Add(ctx, req.FromAgentID, req.ToAgentID, "", nil); err != nil {
			s.logger.Warn("failed to add contact (from→to)", "error", err)
		}
	}

	s.logger.Info("contact request approved", "id", id, "from", req.FromAgentID, "to", req.ToAgentID)
	return req, nil
}

// Reject rejects a contact request with a reason.
func (s *Service) Reject(ctx context.Context, id, reason string) error {
	req, err := s.store.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get contact request: %w", err)
	}
	if req == nil {
		return fmt.Errorf("contact request not found")
	}
	if req.Status != "pending" {
		return fmt.Errorf("contact request is not pending (status: %s)", req.Status)
	}

	if err := s.store.UpdateStatus(ctx, id, "rejected", reason); err != nil {
		return fmt.Errorf("reject contact request: %w", err)
	}

	s.logger.Info("contact request rejected", "id", id, "reason", reason)
	return nil
}

// ListPending returns pending contact requests for the target agent.
func (s *Service) ListPending(ctx context.Context, toAgentID string) ([]ContactRequest, error) {
	return s.store.ListByTarget(ctx, toAgentID, "pending")
}

// ListIncoming returns all contact requests for the target agent, optionally filtered by status.
func (s *Service) ListIncoming(ctx context.Context, toAgentID, status string) ([]ContactRequest, error) {
	return s.store.ListByTarget(ctx, toAgentID, status)
}

// ListSent returns contact requests sent by the given agent, optionally filtered by status.
func (s *Service) ListSent(ctx context.Context, fromAgentID, status string) ([]ContactRequest, error) {
	return s.store.ListBySender(ctx, fromAgentID, status)
}

// GetByID returns a single contact request.
func (s *Service) GetByID(ctx context.Context, id string) (*ContactRequest, error) {
	return s.store.GetByID(ctx, id)
}
