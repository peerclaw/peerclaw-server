package retention

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/peerclaw/peerclaw-server/internal/invocation"
	"github.com/peerclaw/peerclaw-server/internal/reputation"
	"github.com/peerclaw/peerclaw-server/internal/review"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	// Create agents table (needed by reputation store migration).
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		name TEXT DEFAULT '',
		status TEXT DEFAULT 'offline',
		last_heartbeat DATETIME,
		registered_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestRunOnce_PrunesOldData(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	repStore := reputation.NewSQLiteStore(db)
	if err := repStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	invStore := invocation.NewSQLiteStore(db)
	if err := invStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	revStore := review.NewSQLiteStore(db)
	if err := revStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	// Insert test agent for reputation.
	_, err := db.Exec(`INSERT INTO agents (id, name) VALUES ('agent1', 'Test Agent')`)
	if err != nil {
		t.Fatal(err)
	}

	// Insert old reputation event (120 days ago).
	oldTime := time.Now().UTC().AddDate(0, 0, -120)
	_, err = db.Exec(`INSERT INTO reputation_events (agent_id, event_type, weight, score_after, created_at) VALUES (?, ?, ?, ?, ?)`,
		"agent1", "heartbeat_success", 1.0, 0.6, oldTime.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	// Insert recent reputation event (10 days ago).
	recentTime := time.Now().UTC().AddDate(0, 0, -10)
	_, err = db.Exec(`INSERT INTO reputation_events (agent_id, event_type, weight, score_after, created_at) VALUES (?, ?, ?, ?, ?)`,
		"agent1", "heartbeat_success", 1.0, 0.7, recentTime.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	// Insert old invocation (60 days ago).
	_, err = db.Exec(`INSERT INTO invocations (id, agent_id, created_at) VALUES (?, ?, ?)`,
		"inv1", "agent1", oldTime.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	// Insert recent invocation (5 days ago).
	_, err = db.Exec(`INSERT INTO invocations (id, agent_id, created_at) VALUES (?, ?, ?)`,
		"inv2", "agent1", recentTime.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	// Insert old resolved abuse report (400 days ago).
	veryOldTime := time.Now().UTC().AddDate(0, 0, -400)
	_, err = db.Exec(`INSERT INTO abuse_reports (id, reporter_id, target_type, target_id, reason, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"report1", "user1", "agent", "agent1", "spam", "reviewed", veryOldTime.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	// Insert old pending abuse report (400 days ago) — should NOT be pruned.
	_, err = db.Exec(`INSERT INTO abuse_reports (id, reporter_id, target_type, target_id, reason, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"report2", "user1", "agent", "agent1", "spam", "pending", veryOldTime.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	svc := NewService(repStore, invStore, revStore, Config{
		ReputationEventsDays: 90,
		InvocationsDays:      30,
		AbuseReportsDays:     365,
	}, logger)

	result, err := svc.RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if result.ReputationEvents != 1 {
		t.Errorf("expected 1 reputation event pruned, got %d", result.ReputationEvents)
	}
	if result.Invocations != 1 {
		t.Errorf("expected 1 invocation pruned, got %d", result.Invocations)
	}
	if result.AbuseReports != 1 {
		t.Errorf("expected 1 abuse report pruned, got %d", result.AbuseReports)
	}

	// Verify remaining data.
	var repCount int
	db.QueryRow("SELECT COUNT(*) FROM reputation_events").Scan(&repCount)
	if repCount != 1 {
		t.Errorf("expected 1 remaining reputation event, got %d", repCount)
	}

	var invCount int
	db.QueryRow("SELECT COUNT(*) FROM invocations").Scan(&invCount)
	if invCount != 1 {
		t.Errorf("expected 1 remaining invocation, got %d", invCount)
	}

	var reportCount int
	db.QueryRow("SELECT COUNT(*) FROM abuse_reports").Scan(&reportCount)
	if reportCount != 1 {
		t.Errorf("expected 1 remaining abuse report (pending), got %d", reportCount)
	}
}

func TestRunOnce_NilStoresNoPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	svc := NewService(nil, nil, nil, Config{
		ReputationEventsDays: 90,
		InvocationsDays:      30,
		AbuseReportsDays:     365,
	}, logger)

	result, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.ReputationEvents != 0 || result.Invocations != 0 || result.AbuseReports != 0 {
		t.Errorf("expected all zeros for nil stores, got %+v", result)
	}
}

func TestRunOnce_ZeroDaysSkips(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	repStore := reputation.NewSQLiteStore(db)
	if err := repStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	// Insert old reputation event.
	oldTime := time.Now().UTC().AddDate(0, 0, -120)
	_, err := db.Exec(`INSERT INTO agents (id, name) VALUES ('agent1', 'Test Agent')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO reputation_events (agent_id, event_type, weight, score_after, created_at) VALUES (?, ?, ?, ?, ?)`,
		"agent1", "heartbeat_success", 1.0, 0.6, oldTime.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	// Setting days to 0 should skip pruning.
	svc := NewService(repStore, nil, nil, Config{
		ReputationEventsDays: 0,
	}, logger)

	result, err := svc.RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.ReputationEvents != 0 {
		t.Errorf("expected 0 pruned with days=0, got %d", result.ReputationEvents)
	}

	// Verify data still exists.
	var count int
	db.QueryRow("SELECT COUNT(*) FROM reputation_events").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 reputation event to remain, got %d", count)
	}
}
