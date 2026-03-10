package contacts

import (
	"context"
	"database/sql"
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

func TestService_AddListRemove(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Add a contact.
	contact, err := svc.Add(ctx, "agent-owner", "agent-friend", "My Friend", nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if contact.OwnerAgentID != "agent-owner" {
		t.Errorf("OwnerAgentID = %q, want %q", contact.OwnerAgentID, "agent-owner")
	}
	if contact.Alias != "My Friend" {
		t.Errorf("Alias = %q, want %q", contact.Alias, "My Friend")
	}

	// List contacts.
	contacts, err := svc.ListByOwner(ctx, "agent-owner")
	if err != nil {
		t.Fatalf("ListByOwner: %v", err)
	}
	if len(contacts) != 1 {
		t.Errorf("ListByOwner count = %d, want 1", len(contacts))
	}

	// IsAllowed should return true for the contact.
	allowed, err := svc.IsAllowed(ctx, "agent-friend", "agent-owner")
	if err != nil {
		t.Fatalf("IsAllowed: %v", err)
	}
	if !allowed {
		t.Error("IsAllowed = false, want true")
	}

	// Non-contact should not be allowed.
	allowed, err = svc.IsAllowed(ctx, "agent-stranger", "agent-owner")
	if err != nil {
		t.Fatalf("IsAllowed stranger: %v", err)
	}
	if allowed {
		t.Error("IsAllowed stranger = true, want false")
	}

	// Remove the contact.
	if err := svc.Remove(ctx, "agent-owner", "agent-friend"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// List should now be empty.
	contacts, err = svc.ListByOwner(ctx, "agent-owner")
	if err != nil {
		t.Fatalf("ListByOwner after remove: %v", err)
	}
	if len(contacts) != 0 {
		t.Errorf("ListByOwner after remove count = %d, want 0", len(contacts))
	}
}

func TestService_AddValidation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Cannot add self as contact.
	_, err := svc.Add(ctx, "agent-1", "agent-1", "", nil)
	if err == nil {
		t.Fatal("expected error for self-contact, got nil")
	}

	// Cannot add with empty IDs.
	_, err = svc.Add(ctx, "", "agent-2", "", nil)
	if err == nil {
		t.Fatal("expected error for empty owner, got nil")
	}

	_, err = svc.Add(ctx, "agent-1", "", "", nil)
	if err == nil {
		t.Fatal("expected error for empty contact, got nil")
	}
}
