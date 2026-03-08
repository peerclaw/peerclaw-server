package claimtoken

import (
	"context"
	"database/sql"
	"strings"
	"testing"

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
	db := newTestDB(t)
	store := NewSQLiteStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewService(store, nil)
}

func TestService_GenerateAndValidate(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	token, err := svc.Generate(ctx, "user-1", GenerateParams{
		AgentName:    "MyAgent",
		Capabilities: "chat,search",
		Protocols:    "a2a,mcp",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Code should be in "PCW-XXXX-XXXX" format.
	if !strings.HasPrefix(token.Code, "PCW-") {
		t.Errorf("Code = %q, want prefix PCW-", token.Code)
	}
	parts := strings.Split(token.Code, "-")
	if len(parts) != 3 {
		t.Errorf("Code parts = %d, want 3 (PCW-XXXX-XXXX)", len(parts))
	}
	if token.Status != StatusPending {
		t.Errorf("Status = %q, want %q", token.Status, StatusPending)
	}
	if token.AgentName != "MyAgent" {
		t.Errorf("AgentName = %q, want %q", token.AgentName, "MyAgent")
	}
	if token.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", token.UserID, "user-1")
	}

	// Validate the token.
	validated, err := svc.Validate(ctx, token.Code)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if validated.ID != token.ID {
		t.Errorf("validated ID = %q, want %q", validated.ID, token.ID)
	}

	// Claim the token.
	if err := svc.Claim(ctx, token.Code, "agent-42"); err != nil {
		t.Fatalf("Claim: %v", err)
	}

	// Validate should fail now (already claimed).
	_, err = svc.Validate(ctx, token.Code)
	if err == nil {
		t.Fatal("expected error validating claimed token, got nil")
	}
}

func TestService_ListByUser(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Generate two tokens for the same user.
	svc.Generate(ctx, "user-1", GenerateParams{AgentName: "Agent1"})
	svc.Generate(ctx, "user-1", GenerateParams{AgentName: "Agent2"})
	// Generate one for a different user.
	svc.Generate(ctx, "user-2", GenerateParams{AgentName: "Agent3"})

	tokens, err := svc.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("ListByUser count = %d, want 2", len(tokens))
	}
}
