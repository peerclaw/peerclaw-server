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
	store       Store
	jwt         *JWTManager
	bcryptCost  int
	adminEmails map[string]bool
	emailSender EmailSender
	logger      *slog.Logger
}

// NewService creates a new auth service.
func NewService(store Store, jwt *JWTManager, bcryptCost int, adminEmails []string, emailSender EmailSender, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if bcryptCost <= 0 {
		bcryptCost = 12
	}
	ae := make(map[string]bool, len(adminEmails))
	for _, e := range adminEmails {
		ae[strings.TrimSpace(strings.ToLower(e))] = true
	}
	return &Service{
		store:       store,
		jwt:         jwt,
		bcryptCost:  bcryptCost,
		adminEmails: ae,
		emailSender: emailSender,
		logger:      logger,
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

// ErrEmailNotVerified is returned when a user attempts to log in without verifying their email.
var ErrEmailNotVerified = fmt.Errorf("email not verified")

// Register creates a new user account.
// When emailSender is configured, returns (user, nil, nil) — the caller must verify email first.
// When emailSender is nil (dev mode), returns (user, tokens, nil) immediately.
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

	role := "user"
	if s.adminEmails[email] {
		role = "admin"
	}

	now := time.Now().UTC()
	user := &User{
		ID:            uuid.New().String(),
		Email:         email,
		PasswordHash:  string(hash),
		DisplayName:   displayName,
		Role:          role,
		EmailVerified: s.emailSender == nil, // verified immediately when no email sender
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.store.CreateUser(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	// If email sender is configured, send verification OTP instead of returning tokens.
	if s.emailSender != nil {
		if err := s.sendOTP(ctx, email, "register"); err != nil {
			s.logger.Error("failed to send verification email", "email", email, "error", err)
			// Still return the user — they can resend later.
		}
		s.logger.Info("user registered (pending verification)", "user_id", user.ID, "email", user.Email)
		return user, nil, nil
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

	if !user.EmailVerified {
		return user, nil, ErrEmailNotVerified
	}

	if s.adminEmails[user.Email] && user.Role != "admin" {
		user.Role = "admin"
		user.UpdatedAt = time.Now().UTC()
		_ = s.store.UpdateUser(ctx, user)
		s.logger.Info("user auto-promoted to admin", "user_id", user.ID, "email", user.Email)
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

// UpdateProfileRequest holds profile update parameters.
type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name"`
	Email       *string `json:"email"`
	Description *string `json:"description"`
}

// UpdateProfile updates the user's profile fields.
func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*User, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	if req.DisplayName != nil && strings.TrimSpace(*req.DisplayName) != "" {
		user.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.Email != nil {
		newEmail := strings.TrimSpace(strings.ToLower(*req.Email))
		if newEmail == "" || !strings.Contains(newEmail, "@") {
			return nil, fmt.Errorf("invalid email address")
		}
		if newEmail != user.Email {
			if existing, _ := s.store.GetUserByEmail(ctx, newEmail); existing != nil {
				return nil, fmt.Errorf("email already registered")
			}
			user.Email = newEmail
		}
	}
	if req.Description != nil {
		user.Description = *req.Description
	}
	user.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return user, nil
}

// ChangePasswordRequest holds password change parameters.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword changes the user's password after verifying the current one.
func (s *Service) ChangePassword(ctx context.Context, userID string, req ChangePasswordRequest) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}
	if len(req.NewPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	s.logger.Info("user password changed", "user_id", userID)
	return nil
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

// ListUsers returns users with optional search and role filter.
func (s *Service) ListUsers(ctx context.Context, search, role string, limit, offset int) ([]User, int, error) {
	return s.store.ListUsers(ctx, search, role, limit, offset)
}

// UpdateRole updates a user's role after validation.
func (s *Service) UpdateRole(ctx context.Context, userID, newRole string) (*User, error) {
	validRoles := map[string]bool{"user": true, "provider": true, "admin": true}
	if !validRoles[newRole] {
		return nil, fmt.Errorf("invalid role: %s (must be user, provider, or admin)", newRole)
	}

	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	user.Role = newRole
	user.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	s.logger.Info("user role updated", "user_id", userID, "new_role", newRole)
	return user, nil
}

// DeleteUser removes a user by ID.
func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	if err := s.store.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	s.logger.Info("user deleted", "user_id", userID)
	return nil
}

// CountUsers returns the total number of users.
func (s *Service) CountUsers(ctx context.Context) (int, error) {
	return s.store.CountUsers(ctx)
}

// VerifyEmailRequest holds email verification parameters.
type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// VerifyEmail verifies the OTP and activates the user account.
func (s *Service) VerifyEmail(ctx context.Context, req VerifyEmailRequest, ipAddress, userAgent string) (*User, *TokenPair, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	code := strings.TrimSpace(req.Code)
	if email == "" || code == "" {
		return nil, nil, fmt.Errorf("email and code are required")
	}

	codeHash := hashToken(code)
	ev, err := s.store.GetEmailVerification(ctx, email, codeHash, "register")
	if err != nil {
		return nil, nil, fmt.Errorf("invalid or expired verification code")
	}
	if time.Now().After(ev.ExpiresAt) {
		return nil, nil, fmt.Errorf("verification code expired")
	}

	_ = s.store.MarkEmailVerificationUsed(ctx, ev.ID)

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	if err := s.store.SetEmailVerified(ctx, user.ID); err != nil {
		return nil, nil, fmt.Errorf("set email verified: %w", err)
	}
	user.EmailVerified = true

	tokens, err := s.generateTokenPair(ctx, user, ipAddress, userAgent)
	if err != nil {
		return nil, nil, err
	}

	s.logger.Info("email verified", "user_id", user.ID, "email", email)
	return user, tokens, nil
}

// ResendVerification sends a new verification OTP for a pending user.
func (s *Service) ResendVerification(ctx context.Context, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return nil // anti-enumeration
	}
	if user.EmailVerified {
		return nil // already verified, nothing to do
	}
	return s.sendOTP(ctx, email, "register")
}

// RequestPasswordReset sends a password reset OTP. Always returns nil (anti-enumeration).
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	user, _ := s.store.GetUserByEmail(ctx, email)
	if user == nil {
		return nil // anti-enumeration: don't reveal if email exists
	}
	_ = s.sendOTP(ctx, email, "password_reset")
	return nil
}

// ResetPasswordRequest holds password reset parameters.
type ResetPasswordRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

// ResetPassword verifies the OTP and sets a new password.
func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	code := strings.TrimSpace(req.Code)
	if email == "" || code == "" || req.NewPassword == "" {
		return fmt.Errorf("email, code, and new password are required")
	}
	if len(req.NewPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	codeHash := hashToken(code)
	ev, err := s.store.GetEmailVerification(ctx, email, codeHash, "password_reset")
	if err != nil {
		return fmt.Errorf("invalid or expired reset code")
	}
	if time.Now().After(ev.ExpiresAt) {
		return fmt.Errorf("reset code expired")
	}

	_ = s.store.MarkEmailVerificationUsed(ctx, ev.ID)

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	s.logger.Info("password reset", "user_id", user.ID, "email", email)
	return nil
}

// sendOTP generates a 6-digit OTP, checks rate limits, stores the hash, and sends the email.
func (s *Service) sendOTP(ctx context.Context, email, purpose string) error {
	// Rate limit: 5 per hour.
	count, err := s.store.CountRecentVerifications(ctx, email, time.Now().Add(-1*time.Hour))
	if err != nil {
		return fmt.Errorf("count verifications: %w", err)
	}
	if count >= 5 {
		return fmt.Errorf("too many verification requests, please try again later")
	}

	// Generate 6-digit code.
	code, err := generateOTP()
	if err != nil {
		return fmt.Errorf("generate OTP: %w", err)
	}

	codeHash := hashToken(code)
	now := time.Now().UTC()
	ev := &EmailVerification{
		ID:        uuid.New().String(),
		Email:     email,
		CodeHash:  codeHash,
		Purpose:   purpose,
		ExpiresAt: now.Add(10 * time.Minute),
		CreatedAt: now,
	}
	if err := s.store.CreateEmailVerification(ctx, ev); err != nil {
		return fmt.Errorf("store verification: %w", err)
	}

	if s.emailSender != nil {
		if err := s.emailSender.SendVerificationCode(email, code, purpose); err != nil {
			return fmt.Errorf("send email: %w", err)
		}
	}
	return nil
}

// generateOTP returns a cryptographically random 6-digit numeric string.
func generateOTP() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1000000
	return fmt.Sprintf("%06d", n), nil
}

// DeleteExpiredVerifications removes expired verification records.
func (s *Service) DeleteExpiredVerifications(ctx context.Context) error {
	return s.store.DeleteExpiredVerifications(ctx)
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
