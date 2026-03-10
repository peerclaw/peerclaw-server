package contacts

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Service implements contact whitelist business logic.
type Service struct {
	store  Store
	logger *slog.Logger
}

// NewService creates a new contacts service.
func NewService(store Store, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: store, logger: logger}
}

// NewStore creates a Store based on the database driver and a shared *sql.DB.
func NewStore(driver string, db *sql.DB) Store {
	switch driver {
	case "postgres":
		return NewPostgresStore(db)
	default:
		return NewSQLiteStore(db)
	}
}

// Add adds a contact to the owner's whitelist.
func (s *Service) Add(ctx context.Context, ownerAgentID, contactAgentID, alias string, expiresAt *time.Time) (*Contact, error) {
	if ownerAgentID == "" || contactAgentID == "" {
		return nil, fmt.Errorf("owner and contact agent IDs are required")
	}
	if ownerAgentID == contactAgentID {
		return nil, fmt.Errorf("cannot add self as contact")
	}

	contact := &Contact{
		ID:             uuid.New().String(),
		OwnerAgentID:   ownerAgentID,
		ContactAgentID: contactAgentID,
		Alias:          alias,
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now().UTC(),
	}

	if err := s.store.Add(ctx, contact); err != nil {
		return nil, fmt.Errorf("add contact: %w", err)
	}

	s.logger.Info("contact added",
		"owner", ownerAgentID,
		"contact", contactAgentID,
		"alias", alias,
	)
	return contact, nil
}

// Remove removes a contact from the owner's whitelist.
func (s *Service) Remove(ctx context.Context, ownerAgentID, contactAgentID string) error {
	if err := s.store.Remove(ctx, ownerAgentID, contactAgentID); err != nil {
		return fmt.Errorf("remove contact: %w", err)
	}
	s.logger.Info("contact removed", "owner", ownerAgentID, "contact", contactAgentID)
	return nil
}

// IsAllowed checks if fromAgentID is in toAgentID's contact list.
// The receiver (toAgentID) manages the whitelist; fromAgentID must be listed to send.
func (s *Service) IsAllowed(ctx context.Context, fromAgentID, toAgentID string) (bool, error) {
	return s.store.IsAllowed(ctx, toAgentID, fromAgentID)
}

// ListByOwner returns all contacts for the given owner agent.
func (s *Service) ListByOwner(ctx context.Context, ownerAgentID string) ([]Contact, error) {
	return s.store.ListByOwner(ctx, ownerAgentID)
}
