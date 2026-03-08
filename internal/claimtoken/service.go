package claimtoken

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// charset for code generation: A-Z, 0-9 minus easily confused chars (0/O, 1/I/L).
const charset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

const (
	codePrefix = "PCW"
	codeTTL    = 30 * time.Minute
)

// Service implements claim token business logic.
type Service struct {
	store  Store
	logger *slog.Logger
}

// NewService creates a new claim token service.
func NewService(store Store, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: store, logger: logger}
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

// Generate creates a new claim token for the given user.
func (s *Service) Generate(ctx context.Context, userID string) (*ClaimToken, error) {
	code, err := generateCode()
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	now := time.Now().UTC()
	token := &ClaimToken{
		ID:        uuid.New().String(),
		Code:      code,
		UserID:    userID,
		Status:    StatusPending,
		CreatedAt: now,
		ExpiresAt: now.Add(codeTTL),
	}

	if err := s.store.Create(ctx, token); err != nil {
		return nil, fmt.Errorf("create claim token: %w", err)
	}

	s.logger.Info("claim token generated", "code", code, "user_id", userID)
	return token, nil
}

// Validate checks that a claim token exists, is pending, and has not expired.
// Returns the token (including UserID) on success.
func (s *Service) Validate(ctx context.Context, code string) (*ClaimToken, error) {
	token, err := s.store.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("get claim token: %w", err)
	}

	if token.Status != StatusPending {
		return nil, fmt.Errorf("token already claimed")
	}

	if time.Now().UTC().After(token.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	return token, nil
}

// Claim marks a token as claimed by the given agent.
func (s *Service) Claim(ctx context.Context, code, agentID string) error {
	if err := s.store.MarkClaimed(ctx, code, agentID); err != nil {
		return fmt.Errorf("mark claimed: %w", err)
	}
	s.logger.Info("claim token claimed", "code", code, "agent_id", agentID)
	return nil
}

// ListByUser returns all claim tokens for a user.
func (s *Service) ListByUser(ctx context.Context, userID string) ([]ClaimToken, error) {
	return s.store.ListByUser(ctx, userID)
}

// CleanupExpired removes expired, unclaimed tokens.
func (s *Service) CleanupExpired(ctx context.Context) (int64, error) {
	return s.store.DeleteExpired(ctx)
}

// generateCode produces a "PCW-XXXX-XXXX" format code using crypto/rand.
func generateCode() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	chars := make([]byte, 8)
	for i := range chars {
		chars[i] = charset[int(b[i])%len(charset)]
	}

	var sb strings.Builder
	sb.WriteString(codePrefix)
	sb.WriteByte('-')
	sb.Write(chars[:4])
	sb.WriteByte('-')
	sb.Write(chars[4:])
	return sb.String(), nil
}
