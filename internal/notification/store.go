package notification

import (
	"context"
	"database/sql"
	"time"
)

// NotificationType represents the type of notification event.
type NotificationType string

const (
	TypeAccessRequestReceived  NotificationType = "access_request_received"
	TypeAccessRequestApproved  NotificationType = "access_request_approved"
	TypeAccessRequestRejected  NotificationType = "access_request_rejected"
	TypeContactRequestReceived NotificationType = "contact_request_received"
	TypeContactAdded           NotificationType = "contact_added"
	TypeAgentOffline           NotificationType = "agent_offline"
	TypeAgentDegraded          NotificationType = "agent_degraded"
	TypeSDKOutdated            NotificationType = "sdk_outdated"
	TypeReRegister             NotificationType = "re_register"
)

// Severity represents the severity level of a notification.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Notification represents a notification sent to a user.
type Notification struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	AgentID   string            `json:"agent_id"`
	Type      NotificationType  `json:"type"`
	Severity  Severity          `json:"severity"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Read      bool              `json:"read"`
	CreatedAt time.Time         `json:"created_at"`
}

// Store defines the persistence interface for notifications.
type Store interface {
	Create(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, id string) (*Notification, error)
	ListByUser(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, int, error)
	CountUnread(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, id, userID string) error
	MarkAllRead(ctx context.Context, userID string) error
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
	Migrate(ctx context.Context) error
	Close() error
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
