package reputation

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed reputation store.
// It expects the same *sql.DB that the registry uses.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Migrate creates the reputation_events table and adds reputation columns to the agents table.
func (s *PostgresStore) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS reputation_events (
			id          SERIAL PRIMARY KEY,
			agent_id    TEXT NOT NULL,
			event_type  TEXT NOT NULL,
			weight      DOUBLE PRECISION NOT NULL,
			score_after DOUBLE PRECISION NOT NULL,
			metadata    TEXT,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reputation_events_agent ON reputation_events(agent_id, created_at DESC)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("reputation migrate: %w", err)
		}
	}

	// Add columns to agents table if they don't exist.
	columns := []struct {
		name string
		def  string
	}{
		{"reputation_score", "DOUBLE PRECISION DEFAULT 0.5"},
		{"reputation_event_count", "INTEGER DEFAULT 0"},
		{"reputation_updated_at", "TIMESTAMPTZ"},
		{"verified", "BOOLEAN DEFAULT FALSE"},
		{"verified_at", "TIMESTAMPTZ"},
		{"public_endpoint", "BOOLEAN DEFAULT FALSE"},
		{"owner_user_id", "TEXT DEFAULT ''"},
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

func (s *PostgresStore) columnExists(ctx context.Context, table, column string) bool {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = $1 AND column_name = $2
		)`, table, column,
	).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

// InsertEvent records a reputation event.
func (s *PostgresStore) InsertEvent(ctx context.Context, event *Event) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO reputation_events (agent_id, event_type, weight, score_after, metadata, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		event.AgentID, string(event.EventType), event.Weight, event.ScoreAfter,
		event.Metadata, event.CreatedAt.UTC(),
	)
	return err
}

// GetScore returns the current reputation score and event count for an agent.
func (s *PostgresStore) GetScore(ctx context.Context, agentID string) (float64, int64, error) {
	var score float64
	var count int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(reputation_score, 0.5), COALESCE(reputation_event_count, 0)
		 FROM agents WHERE id = $1`, agentID,
	).Scan(&score, &count)
	if err != nil {
		return 0, 0, fmt.Errorf("get reputation score: %w", err)
	}
	return score, count, nil
}

// ListEvents returns reputation events for an agent, ordered by most recent first.
func (s *PostgresStore) ListEvents(ctx context.Context, agentID string, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_id, event_type, weight, score_after, COALESCE(metadata, ''), created_at
		 FROM reputation_events WHERE agent_id = $1
		 ORDER BY created_at DESC LIMIT $2`,
		agentID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.AgentID, &e.EventType, &e.Weight, &e.ScoreAfter, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// UpdateAgentReputation updates the cached reputation score on the agents table.
func (s *PostgresStore) UpdateAgentReputation(ctx context.Context, agentID string, score float64, eventCount int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET reputation_score = $1, reputation_event_count = $2, reputation_updated_at = $3 WHERE id = $4`,
		score, eventCount, time.Now().UTC(), agentID,
	)
	return err
}

// SetAgentVerified marks an agent as verified.
func (s *PostgresStore) SetAgentVerified(ctx context.Context, agentID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET verified = TRUE, verified_at = $1 WHERE id = $2`,
		time.Now().UTC(), agentID,
	)
	return err
}

// UnsetAgentVerified removes the verified status from an agent.
func (s *PostgresStore) UnsetAgentVerified(ctx context.Context, agentID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET verified = FALSE, verified_at = NULL WHERE id = $1`,
		agentID,
	)
	return err
}

// ListStaleOnlineAgents returns IDs of agents whose status is online but
// whose last heartbeat is older than the given timeout.
func (s *PostgresStore) ListStaleOnlineAgents(ctx context.Context, timeout time.Duration) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id FROM agents WHERE status = 'online' AND last_heartbeat < $1`,
		time.Now().UTC().Add(-timeout),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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
func (s *PostgresStore) Close() error {
	return nil
}
