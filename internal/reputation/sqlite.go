package reputation

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed reputation store.
// It expects the same *sql.DB that the registry uses.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// Migrate creates the reputation_events table and adds reputation columns to the agents table.
func (s *SQLiteStore) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS reputation_events (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id    TEXT NOT NULL,
			event_type  TEXT NOT NULL,
			weight      REAL NOT NULL,
			score_after REAL NOT NULL,
			metadata    TEXT,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reputation_events_agent ON reputation_events(agent_id, created_at DESC)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("reputation migrate: %w", err)
		}
	}

	// Add columns to agents table if they don't exist.
	// SQLite doesn't support IF NOT EXISTS for ALTER TABLE,
	// so we check if the column exists first.
	columns := []struct {
		name string
		def  string
	}{
		{"reputation_score", "REAL DEFAULT 0.5"},
		{"reputation_event_count", "INTEGER DEFAULT 0"},
		{"reputation_updated_at", "DATETIME"},
		{"verified", "BOOLEAN DEFAULT 0"},
		{"verified_at", "DATETIME"},
		{"public_endpoint", "BOOLEAN DEFAULT 0"},
		{"owner_user_id", "TEXT DEFAULT ''"},
		{"playground_enabled", "BOOLEAN DEFAULT 0"},
		{"visibility", "TEXT DEFAULT 'public'"},
	}

	for _, col := range columns {
		if !s.columnExists(ctx, "agents", col.name) {
			stmt := fmt.Sprintf("ALTER TABLE agents ADD COLUMN %s %s", col.name, col.def)
			if _, err := s.db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("add column %s: %w", col.name, err)
			}
		}
	}

	return nil
}

func (s *SQLiteStore) columnExists(ctx context.Context, table, column string) bool {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typeName string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typeName, &notNull, &dfltValue, &pk); err != nil {
			continue
		}
		if name == column {
			return true
		}
	}
	return false
}

// InsertEvent records a reputation event.
func (s *SQLiteStore) InsertEvent(ctx context.Context, event *Event) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO reputation_events (agent_id, event_type, weight, score_after, metadata, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		event.AgentID, string(event.EventType), event.Weight, event.ScoreAfter,
		event.Metadata, event.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// GetScore returns the current reputation score and event count for an agent.
func (s *SQLiteStore) GetScore(ctx context.Context, agentID string) (float64, int64, error) {
	var score float64
	var count int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(reputation_score, 0.5), COALESCE(reputation_event_count, 0)
		 FROM agents WHERE id = ?`, agentID,
	).Scan(&score, &count)
	if err != nil {
		return 0, 0, fmt.Errorf("get reputation score: %w", err)
	}
	return score, count, nil
}

// ListEvents returns reputation events for an agent, ordered by most recent first.
func (s *SQLiteStore) ListEvents(ctx context.Context, agentID string, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_id, event_type, weight, score_after, COALESCE(metadata, ''), created_at
		 FROM reputation_events WHERE agent_id = ?
		 ORDER BY created_at DESC LIMIT ?`,
		agentID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var createdAt string
		if err := rows.Scan(&e.ID, &e.AgentID, &e.EventType, &e.Weight, &e.ScoreAfter, &e.Metadata, &createdAt); err != nil {
			return nil, err
		}
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			e.CreatedAt = t
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// UpdateAgentReputation updates the cached reputation score on the agents table.
func (s *SQLiteStore) UpdateAgentReputation(ctx context.Context, agentID string, score float64, eventCount int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET reputation_score = ?, reputation_event_count = ?, reputation_updated_at = ? WHERE id = ?`,
		score, eventCount, time.Now().UTC().Format(time.RFC3339), agentID,
	)
	return err
}

// SetAgentVerified marks an agent as verified.
func (s *SQLiteStore) SetAgentVerified(ctx context.Context, agentID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET verified = 1, verified_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), agentID,
	)
	return err
}

// UnsetAgentVerified removes the verified status from an agent.
func (s *SQLiteStore) UnsetAgentVerified(ctx context.Context, agentID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET verified = 0, verified_at = NULL WHERE id = ?`,
		agentID,
	)
	return err
}

// ListStaleOnlineAgents returns IDs of agents whose status is online but
// whose last heartbeat is older than the given timeout.
func (s *SQLiteStore) ListStaleOnlineAgents(ctx context.Context, timeout time.Duration) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id FROM agents WHERE status = 'online' AND last_heartbeat < ?`,
		time.Now().UTC().Add(-timeout).Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// Close is a no-op since the db is shared with the registry store.
func (s *SQLiteStore) Close() error {
	return nil
}
