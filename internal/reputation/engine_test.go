package reputation

import (
	"context"
	"database/sql"
	"math"
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

	// The reputation store requires an "agents" table to exist (it adds columns to it).
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		name TEXT DEFAULT '',
		status TEXT DEFAULT 'online',
		last_heartbeat DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create agents table: %v", err)
	}

	return db
}

func newTestEngine(t *testing.T, db *sql.DB) *Engine {
	t.Helper()
	store := NewSQLiteStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewEngine(store, nil)
}

func TestEngine_RecordEventAndGetScore(t *testing.T) {
	db := newTestDB(t)
	engine := newTestEngine(t, db)
	ctx := context.Background()

	// Insert a test agent.
	_, err := db.Exec("INSERT INTO agents (id, name) VALUES (?, ?)", "agent-1", "TestAgent")
	if err != nil {
		t.Fatalf("insert agent: %v", err)
	}

	// Initial score should be default (0.5).
	score, err := engine.GetScore(ctx, "agent-1")
	if err != nil {
		t.Fatalf("GetScore: %v", err)
	}
	if score != DefaultScore {
		t.Errorf("initial score = %f, want %f", score, DefaultScore)
	}

	// Record a positive event (verification pass).
	if err := engine.RecordEvent(ctx, "agent-1", EventVerificationPass, "test"); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}

	// Score should increase: EWMA = 0.1 * 1.0 + 0.9 * 0.5 = 0.55
	score, err = engine.GetScore(ctx, "agent-1")
	if err != nil {
		t.Fatalf("GetScore after event: %v", err)
	}
	expected := Alpha*1.0 + (1-Alpha)*DefaultScore // 0.55
	if math.Abs(score-expected) > 0.001 {
		t.Errorf("score after verification_pass = %f, want ~%f", score, expected)
	}

	// Record a negative event (heartbeat miss).
	if err := engine.RecordEvent(ctx, "agent-1", EventHeartbeatMiss, ""); err != nil {
		t.Fatalf("RecordEvent heartbeat_miss: %v", err)
	}

	// Score should decrease: EWMA = 0.1 * 0.35 + 0.9 * 0.55 = 0.53
	score, err = engine.GetScore(ctx, "agent-1")
	if err != nil {
		t.Fatalf("GetScore after heartbeat miss: %v", err)
	}
	expected2 := Alpha*0.35 + (1-Alpha)*expected
	if math.Abs(score-expected2) > 0.001 {
		t.Errorf("score after heartbeat_miss = %f, want ~%f", score, expected2)
	}
}

func TestEngine_GetHistory(t *testing.T) {
	db := newTestDB(t)
	engine := newTestEngine(t, db)
	ctx := context.Background()

	_, err := db.Exec("INSERT INTO agents (id, name) VALUES (?, ?)", "agent-2", "Agent2")
	if err != nil {
		t.Fatalf("insert agent: %v", err)
	}

	// Record multiple events.
	engine.RecordEvent(ctx, "agent-2", EventRegistration, "")
	engine.RecordEvent(ctx, "agent-2", EventBridgeSuccess, "")
	engine.RecordEvent(ctx, "agent-2", EventHeartbeatSuccess, "")

	history, err := engine.GetHistory(ctx, "agent-2", 10)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("history length = %d, want 3", len(history))
	}

	// Verify all three event types are present (order may vary when timestamps
	// are identical at sub-second granularity with RFC3339 storage).
	types := make(map[EventType]bool)
	for _, e := range history {
		types[e.EventType] = true
	}
	for _, et := range []EventType{EventRegistration, EventBridgeSuccess, EventHeartbeatSuccess} {
		if !types[et] {
			t.Errorf("missing event type %q in history", et)
		}
	}
}

func TestEngine_ScoreClampedToRange(t *testing.T) {
	db := newTestDB(t)
	engine := newTestEngine(t, db)
	ctx := context.Background()

	_, err := db.Exec("INSERT INTO agents (id, name) VALUES (?, ?)", "agent-3", "Agent3")
	if err != nil {
		t.Fatalf("insert agent: %v", err)
	}

	// Record many negative events to try to push score below 0.
	for range 50 {
		engine.RecordEvent(ctx, "agent-3", EventVerificationFail, "")
	}

	score, err := engine.GetScore(ctx, "agent-3")
	if err != nil {
		t.Fatalf("GetScore: %v", err)
	}
	if score < 0 || score > 1 {
		t.Errorf("score = %f, should be clamped to [0, 1]", score)
	}
}
