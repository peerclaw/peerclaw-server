package adminaudit

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Service implements admin audit business logic.
type Service struct {
	store  Store
	logger *slog.Logger
}

// NewService creates a new admin audit service.
func NewService(store Store, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: store, logger: logger}
}

// Record saves an admin audit event.
func (s *Service) Record(ctx context.Context, adminUserID, action, targetType, targetID, details, ipAddress string) {
	event := &AdminAuditEvent{
		ID:          uuid.New().String(),
		AdminUserID: adminUserID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Details:     details,
		IPAddress:   ipAddress,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.store.Insert(ctx, event); err != nil {
		s.logger.Debug("failed to record admin audit event",
			"action", action,
			"admin_user_id", adminUserID,
			"error", err,
		)
	}
}

// List returns audit events with optional filtering.
func (s *Service) List(ctx context.Context, adminUserID, action, targetType string, since time.Time, limit, offset int) ([]AdminAuditEvent, int, error) {
	return s.store.List(ctx, adminUserID, action, targetType, since, limit, offset)
}
