package useracl

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
		t.Fatalf("open db: %v", err)
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

func newTestService(t *testing.T) (*Service, *SQLiteStore) {
	t.Helper()
	store := newTestStore(t)
	svc := NewService(store, nil)
	return svc, store
}

func TestStore_CreateAndGetByID(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	req := &AccessRequest{
		ID:        "req-1",
		AgentID:   "agent-1",
		UserID:    "user-1",
		Status:    "pending",
		Message:   "please let me in",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Create(ctx, req); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := store.GetByID(ctx, "req-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.AgentID != "agent-1" || got.UserID != "user-1" || got.Status != "pending" {
		t.Errorf("got %+v", got)
	}
	if got.Message != "please let me in" {
		t.Errorf("message = %q, want %q", got.Message, "please let me in")
	}
}

func TestStore_GetByID_NotFound(t *testing.T) {
	store := newTestStore(t)
	got, err := store.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestStore_GetByAgentAndUser(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	_ = store.Create(ctx, &AccessRequest{
		ID: "req-1", AgentID: "agent-1", UserID: "user-1", Status: "pending",
		CreatedAt: now, UpdatedAt: now,
	})

	got, err := store.GetByAgentAndUser(ctx, "agent-1", "user-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil || got.ID != "req-1" {
		t.Errorf("expected req-1, got %+v", got)
	}

	got, _ = store.GetByAgentAndUser(ctx, "agent-1", "user-999")
	if got != nil {
		t.Errorf("expected nil for unknown user, got %+v", got)
	}
}

func TestStore_ListByAgent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	_ = store.Create(ctx, &AccessRequest{
		ID: "r1", AgentID: "a1", UserID: "u1", Status: "pending", CreatedAt: now, UpdatedAt: now,
	})
	_ = store.Create(ctx, &AccessRequest{
		ID: "r2", AgentID: "a1", UserID: "u2", Status: "approved", CreatedAt: now.Add(time.Second), UpdatedAt: now,
	})
	_ = store.Create(ctx, &AccessRequest{
		ID: "r3", AgentID: "a2", UserID: "u1", Status: "pending", CreatedAt: now, UpdatedAt: now,
	})

	// All for agent a1.
	list, err := store.ListByAgent(ctx, "a1", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}

	// Only pending for a1.
	list, _ = store.ListByAgent(ctx, "a1", "pending")
	if len(list) != 1 || list[0].ID != "r1" {
		t.Errorf("filtered list = %+v", list)
	}
}

func TestStore_ListByUser(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	_ = store.Create(ctx, &AccessRequest{
		ID: "r1", AgentID: "a1", UserID: "u1", Status: "pending", CreatedAt: now, UpdatedAt: now,
	})
	_ = store.Create(ctx, &AccessRequest{
		ID: "r2", AgentID: "a2", UserID: "u1", Status: "approved", CreatedAt: now.Add(time.Second), UpdatedAt: now,
	})

	list, err := store.ListByUser(ctx, "u1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
}

func TestStore_UpdateStatus(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	_ = store.Create(ctx, &AccessRequest{
		ID: "r1", AgentID: "a1", UserID: "u1", Status: "pending", CreatedAt: now, UpdatedAt: now,
	})

	expires := now.Add(24 * time.Hour)
	if err := store.UpdateStatus(ctx, "r1", "approved", "", &expires); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, _ := store.GetByID(ctx, "r1")
	if got.Status != "approved" {
		t.Errorf("status = %q, want approved", got.Status)
	}
	if got.ExpiresAt == nil {
		t.Error("expected non-nil expires_at")
	}
}

func TestStore_IsAllowed(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Not allowed before approval.
	allowed, err := store.IsAllowed(ctx, "a1", "u1")
	if err != nil {
		t.Fatalf("is_allowed: %v", err)
	}
	if allowed {
		t.Error("should not be allowed before any request")
	}

	// Create and approve.
	_ = store.Create(ctx, &AccessRequest{
		ID: "r1", AgentID: "a1", UserID: "u1", Status: "approved", CreatedAt: now, UpdatedAt: now,
	})
	allowed, _ = store.IsAllowed(ctx, "a1", "u1")
	if !allowed {
		t.Error("should be allowed after approval")
	}

	// Expired access.
	past := now.Add(-time.Hour)
	_ = store.UpdateStatus(ctx, "r1", "approved", "", &past)
	allowed, _ = store.IsAllowed(ctx, "a1", "u1")
	if allowed {
		t.Error("should not be allowed after expiry")
	}
}

func TestStore_Delete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	_ = store.Create(ctx, &AccessRequest{
		ID: "r1", AgentID: "a1", UserID: "u1", Status: "approved", CreatedAt: now, UpdatedAt: now,
	})

	if err := store.Delete(ctx, "r1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, _ := store.GetByID(ctx, "r1")
	if got != nil {
		t.Error("expected nil after delete")
	}

	// Delete nonexistent.
	if err := store.Delete(ctx, "r1"); err == nil {
		t.Error("expected error deleting nonexistent")
	}
}

func TestStore_MigrateIdempotent(t *testing.T) {
	store := newTestStore(t)
	// Second migrate should not fail.
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

// --- Service tests ---

func TestService_SubmitRequest(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	req, err := svc.SubmitRequest(ctx, "a1", "u1", "hello")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if req.ID == "" || req.Status != "pending" {
		t.Errorf("unexpected request: %+v", req)
	}

	// Duplicate pending should fail.
	_, err = svc.SubmitRequest(ctx, "a1", "u1", "again")
	if err == nil {
		t.Error("expected error for duplicate pending request")
	}
}

func TestService_SubmitRequest_Validation(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.SubmitRequest(ctx, "", "u1", ""); err == nil {
		t.Error("expected error for empty agent ID")
	}
	if _, err := svc.SubmitRequest(ctx, "a1", "", ""); err == nil {
		t.Error("expected error for empty user ID")
	}
}

func TestService_ApproveAndReject(t *testing.T) {
	svc, store := newTestService(t)
	ctx := context.Background()

	req, _ := svc.SubmitRequest(ctx, "a1", "u1", "")
	if err := svc.Approve(ctx, req.ID, nil); err != nil {
		t.Fatalf("approve: %v", err)
	}

	got, _ := store.GetByID(ctx, req.ID)
	if got.Status != "approved" {
		t.Errorf("status = %q, want approved", got.Status)
	}

	// Submitting when approved should fail.
	_, err := svc.SubmitRequest(ctx, "a1", "u1", "")
	if err == nil {
		t.Error("expected error when already approved")
	}

	// Test reject flow with a different user.
	req2, _ := svc.SubmitRequest(ctx, "a1", "u2", "")
	if err := svc.Reject(ctx, req2.ID, "no thanks"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	got2, _ := store.GetByID(ctx, req2.ID)
	if got2.Status != "rejected" || got2.RejectReason != "no thanks" {
		t.Errorf("got %+v", got2)
	}

	// Re-request after rejection should succeed.
	_, err = svc.SubmitRequest(ctx, "a1", "u2", "please reconsider")
	if err != nil {
		t.Fatalf("re-request after rejection: %v", err)
	}
}

func TestService_Revoke(t *testing.T) {
	svc, store := newTestService(t)
	ctx := context.Background()

	req, _ := svc.SubmitRequest(ctx, "a1", "u1", "")
	_ = svc.Approve(ctx, req.ID, nil)

	if err := svc.Revoke(ctx, req.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	got, _ := store.GetByID(ctx, req.ID)
	if got != nil {
		t.Error("expected nil after revoke")
	}
}

func TestService_IsAllowed(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	allowed, _ := svc.IsAllowed(ctx, "a1", "u1")
	if allowed {
		t.Error("should not be allowed before any request")
	}

	req, _ := svc.SubmitRequest(ctx, "a1", "u1", "")
	_ = svc.Approve(ctx, req.ID, nil)

	allowed, _ = svc.IsAllowed(ctx, "a1", "u1")
	if !allowed {
		t.Error("should be allowed after approval")
	}
}

func TestService_ListAndGet(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	req, _ := svc.SubmitRequest(ctx, "a1", "u1", "msg1")

	list, err := svc.ListByAgent(ctx, "a1", "")
	if err != nil || len(list) != 1 {
		t.Fatalf("list by agent: err=%v len=%d", err, len(list))
	}

	list, err = svc.ListByUser(ctx, "u1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list by user: err=%v len=%d", err, len(list))
	}

	got, err := svc.GetByID(ctx, req.ID)
	if err != nil || got == nil || got.ID != req.ID {
		t.Errorf("get by id: err=%v got=%+v", err, got)
	}

	got, err = svc.GetByAgentAndUser(ctx, "a1", "u1")
	if err != nil || got == nil || got.ID != req.ID {
		t.Errorf("get by agent+user: err=%v got=%+v", err, got)
	}
}

func TestNewStore_Dispatch(t *testing.T) {
	db := newTestDB(t)
	s := NewStore("sqlite", db)
	if _, ok := s.(*SQLiteStore); !ok {
		t.Error("expected SQLiteStore for sqlite driver")
	}
	s = NewStore("", db)
	if _, ok := s.(*SQLiteStore); !ok {
		t.Error("expected SQLiteStore for empty driver")
	}
}
