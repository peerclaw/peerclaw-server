package verification

import (
	"context"
	"time"
)

// ChallengeStatus represents the status of a verification challenge.
type ChallengeStatus string

const (
	StatusPending  ChallengeStatus = "pending"
	StatusVerified ChallengeStatus = "verified"
	StatusFailed   ChallengeStatus = "failed"
	StatusExpired  ChallengeStatus = "expired"
)

// Challenge represents a verification challenge record.
type Challenge struct {
	AgentID   string          `json:"agent_id"`
	Challenge string          `json:"challenge"`
	CreatedAt time.Time       `json:"created_at"`
	ExpiresAt time.Time       `json:"expires_at"`
	Status    ChallengeStatus `json:"status"`
}

// Store defines the persistence interface for verification challenges.
type Store interface {
	// InsertChallenge creates a new verification challenge.
	InsertChallenge(ctx context.Context, ch *Challenge) error

	// GetPendingChallenge retrieves a pending (non-expired) challenge for an agent.
	GetPendingChallenge(ctx context.Context, agentID, nonce string) (*Challenge, error)

	// UpdateChallengeStatus updates the status of a challenge.
	UpdateChallengeStatus(ctx context.Context, agentID, nonce string, status ChallengeStatus) error

	// CleanExpired removes expired challenges.
	CleanExpired(ctx context.Context) error

	// Migrate creates the required tables.
	Migrate(ctx context.Context) error

	// Close releases resources.
	Close() error
}
