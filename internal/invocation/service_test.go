package invocation

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
	db := newTestDB(t)
	store := NewSQLiteStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewService(store, nil)
}

func TestService_RecordAndGetByID(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	record := &InvocationRecord{
		ID:         "inv-001",
		AgentID:    "agent-a",
		UserID:     "user-1",
		Protocol:   "a2a",
		StatusCode: 200,
		DurationMs: 42,
		CreatedAt:  time.Now().UTC(),
	}

	if err := svc.Record(ctx, record); err != nil {
		t.Fatalf("Record: %v", err)
	}

	got, err := svc.GetByID(ctx, "inv-001")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.AgentID != "agent-a" {
		t.Errorf("AgentID = %q, want %q", got.AgentID, "agent-a")
	}
	if got.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", got.StatusCode)
	}
	if got.DurationMs != 42 {
		t.Errorf("DurationMs = %d, want 42", got.DurationMs)
	}

	// Non-existent ID should return error.
	_, err = svc.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID, got nil")
	}
}

func TestService_ListByUserAndAgent(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Insert multiple records.
	for i, rec := range []InvocationRecord{
		{ID: "inv-1", AgentID: "agent-a", UserID: "user-1", StatusCode: 200, DurationMs: 10, CreatedAt: time.Now().UTC()},
		{ID: "inv-2", AgentID: "agent-a", UserID: "user-1", StatusCode: 500, DurationMs: 20, Error: "timeout", CreatedAt: time.Now().UTC()},
		{ID: "inv-3", AgentID: "agent-b", UserID: "user-2", StatusCode: 200, DurationMs: 30, CreatedAt: time.Now().UTC()},
	} {
		r := rec
		_ = i
		if err := svc.Record(ctx, &r); err != nil {
			t.Fatalf("Record %s: %v", r.ID, err)
		}
	}

	// List by user.
	records, total, err := svc.ListByUser(ctx, "user-1", 10, 0)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if total != 2 {
		t.Errorf("ListByUser total = %d, want 2", total)
	}
	if len(records) != 2 {
		t.Errorf("ListByUser count = %d, want 2", len(records))
	}

	// List by agent.
	records, total, err = svc.ListByAgent(ctx, "agent-a", 10, 0)
	if err != nil {
		t.Fatalf("ListByAgent: %v", err)
	}
	if total != 2 {
		t.Errorf("ListByAgent total = %d, want 2", total)
	}

	// List by agent that has only 1 record.
	records, total, err = svc.ListByAgent(ctx, "agent-b", 10, 0)
	if err != nil {
		t.Fatalf("ListByAgent agent-b: %v", err)
	}
	if total != 1 {
		t.Errorf("ListByAgent agent-b total = %d, want 1", total)
	}
}
