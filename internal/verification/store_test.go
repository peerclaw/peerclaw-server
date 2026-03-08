package verification

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

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	db := newTestDB(t)
	store := NewSQLiteStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store
}

func TestStore_InsertAndGetPendingChallenge(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC()
	ch := &Challenge{
		AgentID:   "agent-1",
		Challenge: "abc123nonce",
		CreatedAt: now,
		ExpiresAt: now.Add(5 * time.Minute),
		Status:    StatusPending,
	}

	if err := store.InsertChallenge(ctx, ch); err != nil {
		t.Fatalf("InsertChallenge: %v", err)
	}

	// Retrieve the pending challenge.
	got, err := store.GetPendingChallenge(ctx, "agent-1", "abc123nonce")
	if err != nil {
		t.Fatalf("GetPendingChallenge: %v", err)
	}
	if got.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", got.AgentID, "agent-1")
	}
	if got.Challenge != "abc123nonce" {
		t.Errorf("Challenge = %q, want %q", got.Challenge, "abc123nonce")
	}
	if got.Status != StatusPending {
		t.Errorf("Status = %q, want %q", got.Status, StatusPending)
	}
}

func TestStore_UpdateChallengeStatus(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC()
	ch := &Challenge{
		AgentID:   "agent-2",
		Challenge: "nonce456",
		CreatedAt: now,
		ExpiresAt: now.Add(5 * time.Minute),
		Status:    StatusPending,
	}
	store.InsertChallenge(ctx, ch)

	// Update status to verified.
	if err := store.UpdateChallengeStatus(ctx, "agent-2", "nonce456", StatusVerified); err != nil {
		t.Fatalf("UpdateChallengeStatus: %v", err)
	}

	// Should no longer be retrievable as pending.
	_, err := store.GetPendingChallenge(ctx, "agent-2", "nonce456")
	if err == nil {
		t.Fatal("expected error for verified challenge lookup as pending, got nil")
	}
}

func TestStore_ExpiredChallengeNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC()
	ch := &Challenge{
		AgentID:   "agent-3",
		Challenge: "expired-nonce",
		CreatedAt: now.Add(-10 * time.Minute),
		ExpiresAt: now.Add(-5 * time.Minute), // Already expired.
		Status:    StatusPending,
	}
	store.InsertChallenge(ctx, ch)

	// Should not be retrievable as pending (expired).
	_, err := store.GetPendingChallenge(ctx, "agent-3", "expired-nonce")
	if err == nil {
		t.Fatal("expected error for expired challenge, got nil")
	}
}

func TestStore_CleanExpired(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC()

	// Insert an expired challenge.
	expired := &Challenge{
		AgentID:   "agent-4",
		Challenge: "old-nonce",
		CreatedAt: now.Add(-1 * time.Hour),
		ExpiresAt: now.Add(-30 * time.Minute),
		Status:    StatusPending,
	}
	store.InsertChallenge(ctx, expired)

	// Insert a valid challenge.
	valid := &Challenge{
		AgentID:   "agent-5",
		Challenge: "fresh-nonce",
		CreatedAt: now,
		ExpiresAt: now.Add(5 * time.Minute),
		Status:    StatusPending,
	}
	store.InsertChallenge(ctx, valid)

	// Clean expired.
	if err := store.CleanExpired(ctx); err != nil {
		t.Fatalf("CleanExpired: %v", err)
	}

	// Valid challenge should still be there.
	got, err := store.GetPendingChallenge(ctx, "agent-5", "fresh-nonce")
	if err != nil {
		t.Fatalf("GetPendingChallenge after cleanup: %v", err)
	}
	if got.Challenge != "fresh-nonce" {
		t.Errorf("Challenge = %q, want %q", got.Challenge, "fresh-nonce")
	}
}
