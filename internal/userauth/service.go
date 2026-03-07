package userauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service implements user authentication business logic.
type Service struct {
	store      Store
	jwt        *JWTManager
	bcryptCost int
	logger     *slog.Logger
}

// NewService creates a new auth service.
func NewService(store Store, jwt *JWTManager, bcryptCost int, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if bcryptCost <= 0 {
		bcryptCost = 12
	}
	return &Service{
		store:      store,
		jwt:        jwt,
		bcryptCost: bcryptCost,
		logger:     logger,
	}
}

// RegisterRequest holds registration parameters.
type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// LoginRequest holds login parameters.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register creates a new user account.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*User, *TokenPair, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, nil, fmt.Errorf("invalid email address")
	}
	if len(req.Password) < 8 {
		return nil, nil, fmt.Errorf("password must be at least 8 characters")
	}

	// Check if email already exists.
	if existing, _ := s.store.GetUserByEmail(ctx, email); existing != nil {
		return nil, nil, fmt.Errorf("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = strings.Split(email, "@")[0]
	}

	now := time.Now().UTC()
	user := &User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  displayName,
		Role:         "user",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.CreateUser(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	tokens, err := s.generateTokenPair(ctx, user, "", "")
	if err != nil {
		return nil, nil, err
	}

	s.logger.Info("user registered", "user_id", user.ID, "email", user.Email)
	return user, tokens, nil
}

// Login authenticates a user and returns a token pair.
func (s *Service) Login(ctx context.Context, req LoginRequest, ipAddress, userAgent string) (*User, *TokenPair, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return nil, nil, fmt.Errorf("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, nil, fmt.Errorf("invalid email or password")
	}

	tokens, err := s.generateTokenPair(ctx, user, ipAddress, userAgent)
	if err != nil {
		return nil, nil, err
	}

	s.logger.Info("user logged in", "user_id", user.ID)
	return user, tokens, nil
}

// RefreshToken generates a new access token from a valid refresh token.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	tokenHash := hashToken(refreshToken)
	session, err := s.store.GetSessionByToken(ctx, tokenHash)
	if err != nil || session == nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.store.DeleteSession(ctx, session.ID)
		return nil, fmt.Errorf("refresh token expired")
	}

	user, err := s.store.GetUserByID(ctx, session.UserID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Delete old session and create a new one (rotation).
	_ = s.store.DeleteSession(ctx, session.ID)

	tokens, err := s.generateTokenPair(ctx, user, session.IPAddress, session.UserAgent)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// Logout invalidates a refresh token.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	session, err := s.store.GetSessionByToken(ctx, tokenHash)
	if err != nil || session == nil {
		return nil // idempotent
	}
	return s.store.DeleteSession(ctx, session.ID)
}

// GetUser retrieves a user by ID.
func (s *Service) GetUser(ctx context.Context, userID string) (*User, error) {
	return s.store.GetUserByID(ctx, userID)
}

// UpdateProfile updates the user's display name.
func (s *Service) UpdateProfile(ctx context.Context, userID, displayName string) (*User, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	if displayName != "" {
		user.DisplayName = strings.TrimSpace(displayName)
	}
	user.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return user, nil
}

// GenerateAPIKey creates a new API key for a user.
func (s *Service) GenerateAPIKey(ctx context.Context, userID, name string) (*UserAPIKey, string, error) {
	if name == "" {
		return nil, "", fmt.Errorf("API key name is required")
	}

	// Generate a random key.
	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		return nil, "", fmt.Errorf("generate key: %w", err)
	}
	keyStr := "pck_" + hex.EncodeToString(rawKey)
	keyHash := hashToken(keyStr)

	now := time.Now().UTC()
	apiKey := &UserAPIKey{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      name,
		KeyHash:   keyHash,
		Prefix:    keyStr[:12],
		CreatedAt: now,
	}

	if err := s.store.CreateAPIKey(ctx, apiKey); err != nil {
		return nil, "", fmt.Errorf("create API key: %w", err)
	}

	s.logger.Info("API key created", "user_id", userID, "key_id", apiKey.ID)
	return apiKey, keyStr, nil
}

// ListAPIKeys returns all API keys for a user.
func (s *Service) ListAPIKeys(ctx context.Context, userID string) ([]UserAPIKey, error) {
	return s.store.ListAPIKeys(ctx, userID)
}

// RevokeAPIKey revokes an API key.
func (s *Service) RevokeAPIKey(ctx context.Context, keyID, userID string) error {
	return s.store.RevokeAPIKey(ctx, keyID, userID)
}

// ValidateAPIKey validates a user API key and returns the user.
func (s *Service) ValidateAPIKey(ctx context.Context, key string) (*User, error) {
	keyHash := hashToken(key)
	apiKey, err := s.store.GetAPIKeyByHash(ctx, keyHash)
	if err != nil || apiKey == nil {
		return nil, fmt.Errorf("invalid API key")
	}
	if apiKey.Revoked {
		return nil, fmt.Errorf("API key revoked")
	}
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, fmt.Errorf("API key expired")
	}

	_ = s.store.UpdateAPIKeyLastUsed(ctx, apiKey.ID)

	return s.store.GetUserByID(ctx, apiKey.UserID)
}

// JWTManager returns the JWT manager.
func (s *Service) JWTManager() *JWTManager {
	return s.jwt
}

// generateTokenPair creates a new access+refresh token pair and persists the session.
func (s *Service) generateTokenPair(ctx context.Context, user *User, ipAddress, userAgent string) (*TokenPair, error) {
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// Generate random refresh token.
	rawRefresh := make([]byte, 32)
	if _, err := rand.Read(rawRefresh); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshToken := hex.EncodeToString(rawRefresh)
	refreshHash := hashToken(refreshToken)

	session := &Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		RefreshToken: refreshHash,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().UTC().Add(s.jwt.RefreshTTL()),
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwt.AccessTTL().Seconds()),
	}, nil
}

// hashToken returns a SHA-256 hex digest of the token.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
