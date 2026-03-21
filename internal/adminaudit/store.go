package adminaudit

import (
	"context"
	"time"
)

// AdminAuditEvent represents an admin action log entry.
type AdminAuditEvent struct {
	ID          string    `json:"id"`
	AdminUserID string    `json:"admin_user_id"`
	Action      string    `json:"action"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	Details     string    `json:"details,omitempty"`
	IPAddress   string    `json:"ip_address,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Store defines the persistence interface for admin audit events.
type Store interface {
	// Insert records a new audit event.
	Insert(ctx context.Context, event *AdminAuditEvent) error

	// List returns audit events with optional filtering.
	List(ctx context.Context, adminUserID, action, targetType string, since time.Time, limit, offset int) ([]AdminAuditEvent, int, error)

	// Migrate creates the required tables.
	Migrate(ctx context.Context) error

	// Close releases resources.
	Close() error
}
