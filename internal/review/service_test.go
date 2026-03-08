package review

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
	// Pass nil for reputation engine (not needed for unit tests).
	return NewService(store, nil, nil)
}

func TestService_SubmitAndListReviews(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Submit a review.
	review, err := svc.SubmitReview(ctx, "agent-1", "user-1", 5, "Excellent agent!")
	if err != nil {
		t.Fatalf("SubmitReview: %v", err)
	}
	if review.Rating != 5 {
		t.Errorf("Rating = %d, want 5", review.Rating)
	}
	if review.Comment != "Excellent agent!" {
		t.Errorf("Comment = %q, want %q", review.Comment, "Excellent agent!")
	}

	// Submit another review from a different user.
	_, err = svc.SubmitReview(ctx, "agent-1", "user-2", 3, "Okay agent")
	if err != nil {
		t.Fatalf("SubmitReview user-2: %v", err)
	}

	// List reviews for the agent.
	reviews, total, err := svc.ListReviews(ctx, "agent-1", 10, 0)
	if err != nil {
		t.Fatalf("ListReviews: %v", err)
	}
	if total != 2 {
		t.Errorf("total reviews = %d, want 2", total)
	}
	if len(reviews) != 2 {
		t.Errorf("reviews count = %d, want 2", len(reviews))
	}

	// Get summary.
	summary, err := svc.GetSummary(ctx, "agent-1")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if summary.TotalReviews != 2 {
		t.Errorf("TotalReviews = %d, want 2", summary.TotalReviews)
	}
	// Average of 5 and 3 = 4.0
	if summary.AverageRating != 4.0 {
		t.Errorf("AverageRating = %f, want 4.0", summary.AverageRating)
	}
}

func TestService_SubmitReview_Validation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Rating below 1 should fail.
	_, err := svc.SubmitReview(ctx, "agent-1", "user-1", 0, "")
	if err == nil {
		t.Fatal("expected error for rating 0, got nil")
	}

	// Rating above 5 should fail.
	_, err = svc.SubmitReview(ctx, "agent-1", "user-1", 6, "")
	if err == nil {
		t.Fatal("expected error for rating 6, got nil")
	}
}

func TestService_SubmitReview_Upsert(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Submit a review.
	_, err := svc.SubmitReview(ctx, "agent-1", "user-1", 3, "initial")
	if err != nil {
		t.Fatalf("SubmitReview: %v", err)
	}

	// Submit again with the same user/agent to update.
	_, err = svc.SubmitReview(ctx, "agent-1", "user-1", 5, "updated")
	if err != nil {
		t.Fatalf("SubmitReview (upsert): %v", err)
	}

	// Should still have only one review.
	reviews, total, err := svc.ListReviews(ctx, "agent-1", 10, 0)
	if err != nil {
		t.Fatalf("ListReviews: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1 after upsert", total)
	}
	if len(reviews) > 0 && reviews[0].Rating != 5 {
		t.Errorf("Rating after upsert = %d, want 5", reviews[0].Rating)
	}
}
