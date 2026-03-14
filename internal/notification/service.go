package notification

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"
)

// Emitter is a callback invoked after a notification is created.
// It decouples the notification service from WebSocket/email delivery.
type Emitter func(n *Notification)

// Service provides notification business logic.
type Service struct {
	store   Store
	emitter Emitter
	logger  *slog.Logger
}

// NewService creates a new notification service.
func NewService(store Store, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: store, logger: logger}
}

// SetEmitter sets the callback invoked after each notification is created.
func (s *Service) SetEmitter(e Emitter) {
	s.emitter = e
}

// Notify creates a notification and invokes the emitter callback.
func (s *Service) Notify(ctx context.Context, userID, agentID string, ntype NotificationType, severity Severity, title, body string, metadata map[string]string) (*Notification, error) {
	n := &Notification{
		ID:        generateID(),
		UserID:    userID,
		AgentID:   agentID,
		Type:      ntype,
		Severity:  severity,
		Title:     title,
		Body:      body,
		Metadata:  metadata,
		Read:      false,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.store.Create(ctx, n); err != nil {
		return nil, err
	}

	s.logger.Debug("notification created",
		"id", n.ID,
		"user_id", userID,
		"type", ntype,
		"severity", severity,
	)

	if s.emitter != nil {
		s.emitter(n)
	}

	return n, nil
}

// List returns notifications for a user with pagination.
func (s *Service) List(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, int, error) {
	return s.store.ListByUser(ctx, userID, unreadOnly, limit, offset)
}

// CountUnread returns the number of unread notifications for a user.
func (s *Service) CountUnread(ctx context.Context, userID string) (int, error) {
	return s.store.CountUnread(ctx, userID)
}

// MarkRead marks a single notification as read.
func (s *Service) MarkRead(ctx context.Context, id, userID string) error {
	return s.store.MarkRead(ctx, id, userID)
}

// MarkAllRead marks all notifications as read for a user.
func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.store.MarkAllRead(ctx, userID)
}

// Prune deletes notifications older than the given cutoff time.
func (s *Service) Prune(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.store.DeleteOlderThan(ctx, cutoff)
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
