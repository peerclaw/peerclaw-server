package userauth

import (
	"context"
	"time"
)

// User represents a registered user.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description"`
	Role         string    `json:"role"` // "user", "provider", "admin"
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session represents an active refresh token session.
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	RefreshToken string    `json:"-"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserAPIKey represents a user-generated API key.
type UserAPIKey struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Name      string     `json:"name"`
	KeyHash   string     `json:"-"`
	Prefix    string     `json:"prefix"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Revoked   bool       `json:"revoked"`
}

// Store defines the persistence interface for user authentication data.
type Store interface {
	// CreateUser inserts a new user.
	CreateUser(ctx context.Context, user *User) error

	// GetUserByID retrieves a user by ID.
	GetUserByID(ctx context.Context, id string) (*User, error)

	// GetUserByEmail retrieves a user by email.
	GetUserByEmail(ctx context.Context, email string) (*User, error)

	// UpdateUser updates user fields.
	UpdateUser(ctx context.Context, user *User) error

	// CreateSession inserts a new session.
	CreateSession(ctx context.Context, session *Session) error

	// GetSessionByToken retrieves a session by refresh token hash.
	GetSessionByToken(ctx context.Context, tokenHash string) (*Session, error)

	// DeleteSession removes a session.
	DeleteSession(ctx context.Context, id string) error

	// DeleteExpiredSessions removes all expired sessions.
	DeleteExpiredSessions(ctx context.Context) error

	// CreateAPIKey inserts a new API key record.
	CreateAPIKey(ctx context.Context, key *UserAPIKey) error

	// ListAPIKeys returns all API keys for a user.
	ListAPIKeys(ctx context.Context, userID string) ([]UserAPIKey, error)

	// GetAPIKeyByHash retrieves an API key by its hash.
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*UserAPIKey, error)

	// RevokeAPIKey marks an API key as revoked.
	RevokeAPIKey(ctx context.Context, keyID, userID string) error

	// UpdateAPIKeyLastUsed updates the last_used timestamp.
	UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error

	// ListUsers returns users with optional search and role filter, plus total count.
	ListUsers(ctx context.Context, search, role string, limit, offset int) ([]User, int, error)

	// DeleteUser removes a user by ID.
	DeleteUser(ctx context.Context, id string) error

	// CountUsers returns the total number of users.
	CountUsers(ctx context.Context) (int, error)

	// Migrate creates the required tables.
	Migrate(ctx context.Context) error

	// Close releases resources.
	Close() error
}
