package contactreq

import (
	"context"
	"database/sql"
	"time"
)

// ContactRequest represents an agent-to-agent contact request.
type ContactRequest struct {
	ID           string    `json:"id"`
	FromAgentID  string    `json:"from_agent_id"`
	ToAgentID    string    `json:"to_agent_id"`
	Status       string    `json:"status"` // pending, approved, rejected
	Message      string    `json:"message"`
	RejectReason string    `json:"reject_reason,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Store defines the persistence interface for agent contact requests.
type Store interface {
	Create(ctx context.Context, req *ContactRequest) error
	GetByID(ctx context.Context, id string) (*ContactRequest, error)
	GetByAgents(ctx context.Context, fromAgentID, toAgentID string) (*ContactRequest, error)
	ListByTarget(ctx context.Context, toAgentID string, status string) ([]ContactRequest, error)
	ListBySender(ctx context.Context, fromAgentID string, status string) ([]ContactRequest, error)
	UpdateStatus(ctx context.Context, id, status, rejectReason string) error
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
