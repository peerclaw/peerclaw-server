package userauth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	return newTestServiceWithEmail(t, nil)
}

func newTestServiceWithEmail(t *testing.T, emailSender EmailSender) *Service {
	t.Helper()
	db := newTestDB(t)
	store := NewSQLiteStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	jwt := NewJWTManager("test-secret-key-for-tests", 15*time.Minute, 7*24*time.Hour)
	// Use low bcrypt cost (4) to keep tests fast.
	return NewService(store, jwt, 4, nil, emailSender, nil)
}

// mockEmailSender captures sent codes for testing.
type mockEmailSender struct {
	codes []struct{ to, code, purpose string }
}

func (m *mockEmailSender) SendVerificationCode(to, code, purpose string) error {
	m.codes = append(m.codes, struct{ to, code, purpose string }{to, code, purpose})
	return nil
}

func (m *mockEmailSender) lastCode() string {
	if len(m.codes) == 0 {
		return ""
	}
	return m.codes[len(m.codes)-1].code
}

func TestService_RegisterAndLogin(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Register a new user.
	user, tokens, err := svc.Register(ctx, RegisterRequest{
		Email:       "alice@example.com",
		Password:    "securepassword",
		DisplayName: "Alice",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "alice@example.com")
	}
	if user.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q, want %q", user.DisplayName, "Alice")
	}
	if user.Role != "user" {
		t.Errorf("Role = %q, want %q", user.Role, "user")
	}
	if tokens.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
	if tokens.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}

	// Login with the same credentials.
	loggedIn, loginTokens, err := svc.Login(ctx, LoginRequest{
		Email:    "alice@example.com",
		Password: "securepassword",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if loggedIn.ID != user.ID {
		t.Errorf("Login user ID = %q, want %q", loggedIn.ID, user.ID)
	}
	if loginTokens.AccessToken == "" {
		t.Error("Login AccessToken is empty")
	}

	// Login with wrong password should fail.
	_, _, err = svc.Login(ctx, LoginRequest{
		Email:    "alice@example.com",
		Password: "wrongpassword",
	}, "", "")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestService_Register_Validation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Empty email should fail.
	_, _, err := svc.Register(ctx, RegisterRequest{
		Email:    "",
		Password: "securepassword",
	})
	if err == nil {
		t.Fatal("expected error for empty email")
	}

	// Short password should fail.
	_, _, err = svc.Register(ctx, RegisterRequest{
		Email:    "bob@example.com",
		Password: "short",
	})
	if err == nil {
		t.Fatal("expected error for short password")
	}

	// Duplicate email should fail.
	_, _, err = svc.Register(ctx, RegisterRequest{
		Email:    "alice@example.com",
		Password: "securepassword",
	})
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, _, err = svc.Register(ctx, RegisterRequest{
		Email:    "alice@example.com",
		Password: "anotherpassword",
	})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	mgr := NewJWTManager("my-secret", 15*time.Minute, 24*time.Hour)

	token, err := mgr.GenerateAccessToken("user-123", "admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}

	claims, err := mgr.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}

	// Invalid token should fail.
	_, err = mgr.ValidateAccessToken("garbage.token.here")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}

	// Token signed with different secret should fail.
	otherMgr := NewJWTManager("different-secret", 15*time.Minute, 24*time.Hour)
	_, err = otherMgr.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestRegister_RequiresVerification(t *testing.T) {
	mock := &mockEmailSender{}
	svc := newTestServiceWithEmail(t, mock)
	ctx := context.Background()

	user, tokens, err := svc.Register(ctx, RegisterRequest{
		Email:    "alice@example.com",
		Password: "securepassword",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if tokens != nil {
		t.Fatal("expected nil tokens when email sender is configured")
	}
	if user.EmailVerified {
		t.Fatal("expected EmailVerified=false")
	}
	if len(mock.codes) == 0 {
		t.Fatal("expected email sender to be called")
	}
	if mock.codes[0].purpose != "register" {
		t.Errorf("purpose = %q, want %q", mock.codes[0].purpose, "register")
	}
}

func TestVerifyEmail_Success(t *testing.T) {
	mock := &mockEmailSender{}
	svc := newTestServiceWithEmail(t, mock)
	ctx := context.Background()

	// Register (requires verification).
	_, _, _ = svc.Register(ctx, RegisterRequest{
		Email:    "alice@example.com",
		Password: "securepassword",
	})
	code := mock.lastCode()
	if code == "" {
		t.Fatal("no code captured")
	}

	// Verify with correct code.
	user, tokens, err := svc.VerifyEmail(ctx, VerifyEmailRequest{
		Email: "alice@example.com",
		Code:  code,
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	if !user.EmailVerified {
		t.Fatal("expected EmailVerified=true")
	}
	if tokens == nil || tokens.AccessToken == "" {
		t.Fatal("expected tokens after verification")
	}
}

func TestVerifyEmail_WrongCode(t *testing.T) {
	mock := &mockEmailSender{}
	svc := newTestServiceWithEmail(t, mock)
	ctx := context.Background()

	_, _, _ = svc.Register(ctx, RegisterRequest{
		Email:    "alice@example.com",
		Password: "securepassword",
	})

	_, _, err := svc.VerifyEmail(ctx, VerifyEmailRequest{
		Email: "alice@example.com",
		Code:  "000000",
	}, "", "")
	if err == nil {
		t.Fatal("expected error for wrong code")
	}
}

func TestLogin_UnverifiedEmail(t *testing.T) {
	mock := &mockEmailSender{}
	svc := newTestServiceWithEmail(t, mock)
	ctx := context.Background()

	_, _, _ = svc.Register(ctx, RegisterRequest{
		Email:    "alice@example.com",
		Password: "securepassword",
	})

	// Login should fail with ErrEmailNotVerified.
	_, _, err := svc.Login(ctx, LoginRequest{
		Email:    "alice@example.com",
		Password: "securepassword",
	}, "", "")
	if err != ErrEmailNotVerified {
		t.Fatalf("expected ErrEmailNotVerified, got %v", err)
	}
}

func TestResendVerification_RateLimit(t *testing.T) {
	mock := &mockEmailSender{}
	svc := newTestServiceWithEmail(t, mock)
	ctx := context.Background()

	_, _, _ = svc.Register(ctx, RegisterRequest{
		Email:    "alice@example.com",
		Password: "securepassword",
	})

	// Register already sent 1, send 4 more = 5 total.
	for i := 0; i < 4; i++ {
		if err := svc.ResendVerification(ctx, "alice@example.com"); err != nil {
			t.Fatalf("ResendVerification %d: %v", i, err)
		}
	}

	// 6th should fail due to rate limit.
	err := svc.ResendVerification(ctx, "alice@example.com")
	if err == nil {
		t.Fatal("expected rate limit error")
	}
}

func TestRequestPasswordReset_AntiEnumeration(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Should return nil even for non-existent email.
	err := svc.RequestPasswordReset(ctx, "nobody@example.com")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestResetPassword_Success(t *testing.T) {
	mock := &mockEmailSender{}
	svc := newTestServiceWithEmail(t, mock)
	ctx := context.Background()

	// Register and verify user first.
	_, _, _ = svc.Register(ctx, RegisterRequest{
		Email:    "alice@example.com",
		Password: "securepassword",
	})
	regCode := mock.lastCode()
	_, _, _ = svc.VerifyEmail(ctx, VerifyEmailRequest{
		Email: "alice@example.com",
		Code:  regCode,
	}, "", "")

	// Request password reset.
	_ = svc.RequestPasswordReset(ctx, "alice@example.com")
	resetCode := mock.lastCode()
	if resetCode == "" {
		t.Fatal("no reset code captured")
	}

	// Reset password.
	err := svc.ResetPassword(ctx, ResetPasswordRequest{
		Email:       "alice@example.com",
		Code:        resetCode,
		NewPassword: "newsecurepassword",
	})
	if err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	// Login with new password should work.
	_, tokens, err := svc.Login(ctx, LoginRequest{
		Email:    "alice@example.com",
		Password: "newsecurepassword",
	}, "", "")
	if err != nil {
		t.Fatalf("Login after reset: %v", err)
	}
	if tokens == nil {
		t.Fatal("expected tokens")
	}

	// Old password should fail.
	_, _, err = svc.Login(ctx, LoginRequest{
		Email:    "alice@example.com",
		Password: "securepassword",
	}, "", "")
	if err == nil {
		t.Fatal("expected error for old password")
	}
}

func TestResetPassword_WrongCode(t *testing.T) {
	mock := &mockEmailSender{}
	svc := newTestServiceWithEmail(t, mock)
	ctx := context.Background()

	_, _, _ = svc.Register(ctx, RegisterRequest{
		Email:    "alice@example.com",
		Password: "securepassword",
	})
	regCode := mock.lastCode()
	_, _, _ = svc.VerifyEmail(ctx, VerifyEmailRequest{
		Email: "alice@example.com",
		Code:  regCode,
	}, "", "")

	_ = svc.RequestPasswordReset(ctx, "alice@example.com")

	err := svc.ResetPassword(ctx, ResetPasswordRequest{
		Email:       "alice@example.com",
		Code:        "000000",
		NewPassword: "newsecurepassword",
	})
	if err == nil {
		t.Fatal("expected error for wrong reset code")
	}
}
