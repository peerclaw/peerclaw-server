package contacts

import (
	"context"
	"time"
)


// Contact represents a whitelist entry allowing one agent to communicate with another.
// The OwnerAgentID manages the list; ContactAgentID is the agent allowed to send messages.
type Contact struct {
	ID             string     `json:"id"`
	OwnerAgentID   string     `json:"owner_agent_id"`
	ContactAgentID string     `json:"contact_agent_id"`
	Alias          string     `json:"alias"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Store defines the persistence interface for agent contacts.
type Store interface {
	// Add inserts a new contact entry.
	Add(ctx context.Context, contact *Contact) error

	// Remove deletes a contact entry.
	Remove(ctx context.Context, ownerAgentID, contactAgentID string) error

	// IsAllowed checks whether contactAgentID is in ownerAgentID's contact list.
	IsAllowed(ctx context.Context, ownerAgentID, contactAgentID string) (bool, error)

	// ListByOwner returns all contacts for the given owner agent.
	ListByOwner(ctx context.Context, ownerAgentID string) ([]Contact, error)

	// Migrate creates the required database tables and indexes.
	Migrate(ctx context.Context) error

	// Close releases resources.
	Close() error
}
