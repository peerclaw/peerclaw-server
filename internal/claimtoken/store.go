package claimtoken

import (
	"context"
	"time"
)

// ClaimToken represents a one-time-use token that binds a user account to an agent's public key.
// Agent metadata (name, capabilities, protocols) is captured at generation time so the
// CLI claim command only needs the token code — no additional parameters.
type ClaimToken struct {
	ID           string     `json:"id"`
	Code         string     `json:"code"`           // "PCW-XXXX-XXXX" format
	UserID       string     `json:"user_id"`         // the user who generated this token
	Status       string     `json:"status"`          // "pending", "claimed", "expired"
	AgentID      string     `json:"agent_id"`        // filled after claim
	AgentName    string     `json:"agent_name"`      // pre-configured agent name
	Capabilities string     `json:"capabilities"`    // comma-separated capability list
	Protocols    string     `json:"protocols"`       // comma-separated protocol list
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	ClaimedAt    *time.Time `json:"claimed_at,omitempty"`
}

const (
	StatusPending = "pending"
	StatusClaimed = "claimed"
	StatusExpired = "expired"
)

// Store defines the persistence interface for claim tokens.
type Store interface {
	// Create inserts a new claim token.
	Create(ctx context.Context, token *ClaimToken) error

	// GetByCode retrieves a claim token by its short code.
	GetByCode(ctx context.Context, code string) (*ClaimToken, error)

	// MarkClaimed marks a token as claimed by the given agent.
	MarkClaimed(ctx context.Context, code, agentID string) error

	// ListByUser returns all claim tokens for a user.
	ListByUser(ctx context.Context, userID string) ([]ClaimToken, error)

	// DeleteExpired removes expired, unclaimed tokens.
	DeleteExpired(ctx context.Context) (int64, error)

	// Migrate creates the required database tables and indexes.
	Migrate(ctx context.Context) error

	// Close releases resources.
	Close() error
}
