package useracl

import (
	"context"
	"database/sql"
	"time"
)

// AccessRequest represents a user's request to access an agent.
type AccessRequest struct {
	ID           string     `json:"id"`
	AgentID      string     `json:"agent_id"`
	UserID       string     `json:"user_id"`
	Status       string     `json:"status"` // pending, approved, rejected
	Message      string     `json:"message"`
	RejectReason string     `json:"reject_reason,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Store defines the persistence interface for user access control.
type Store interface {
	Create(ctx context.Context, req *AccessRequest) error
	GetByID(ctx context.Context, id string) (*AccessRequest, error)
	GetByAgentAndUser(ctx context.Context, agentID, userID string) (*AccessRequest, error)
	ListByAgent(ctx context.Context, agentID string, status string) ([]AccessRequest, error)
	ListByUser(ctx context.Context, userID string) ([]AccessRequest, error)
	UpdateStatus(ctx context.Context, id, status, rejectReason string, expiresAt *time.Time) error
	IsAllowed(ctx context.Context, agentID, userID string) (bool, error)
	Delete(ctx context.Context, id string) error
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
