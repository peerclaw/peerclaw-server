package invocation

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

// NewPostgresStore creates a new PostgreSQL-backed invocation store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS invocations (
			id            TEXT PRIMARY KEY,
			agent_id      TEXT NOT NULL,
			user_id       TEXT DEFAULT '',
			protocol      TEXT DEFAULT '',
			request_body  TEXT DEFAULT '',
			response_body TEXT DEFAULT '',
			status_code   INTEGER DEFAULT 0,
			duration_ms   BIGINT DEFAULT 0,
			error         TEXT DEFAULT '',
			ip_address    TEXT DEFAULT '',
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_invocations_agent ON invocations(agent_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_invocations_user ON invocations(user_id, created_at DESC)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("invocation migrate: %w", err)
		}
	}
	return nil
}

func (s *PostgresStore) Insert(ctx context.Context, record *InvocationRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO invocations (id, agent_id, user_id, protocol, request_body, response_body, status_code, duration_ms, error, ip_address, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		record.ID, record.AgentID, record.UserID, record.Protocol,
		record.RequestBody, record.ResponseBody, record.StatusCode,
		record.DurationMs, record.Error, record.IPAddress,
		record.CreatedAt.UTC(),
	)
	return err
}

func (s *PostgresStore) GetByID(ctx context.Context, id string) (*InvocationRecord, error) {
	var r InvocationRecord
	err := s.db.QueryRowContext(ctx,
		`SELECT id, agent_id, user_id, protocol, request_body, response_body, status_code, duration_ms, error, ip_address, created_at
		 FROM invocations WHERE id = $1`, id,
	).Scan(&r.ID, &r.AgentID, &r.UserID, &r.Protocol, &r.RequestBody, &r.ResponseBody,
		&r.StatusCode, &r.DurationMs, &r.Error, &r.IPAddress, &r.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invocation not found")
		}
		return nil, err
	}
	return &r, nil
}

func (s *PostgresStore) ListByUser(ctx context.Context, userID string, limit, offset int) ([]InvocationRecord, int, error) {
	if limit <= 0 {
		limit = 50
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invocations WHERE user_id = $1", userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_id, user_id, protocol, status_code, duration_ms, error, ip_address, created_at
		 FROM invocations WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var records []InvocationRecord
	for rows.Next() {
		var r InvocationRecord
		if err := rows.Scan(&r.ID, &r.AgentID, &r.UserID, &r.Protocol, &r.StatusCode, &r.DurationMs, &r.Error, &r.IPAddress, &r.CreatedAt); err != nil {
			return nil, 0, err
		}
		records = append(records, r)
	}
	return records, total, rows.Err()
}

func (s *PostgresStore) ListByAgent(ctx context.Context, agentID string, limit, offset int) ([]InvocationRecord, int, error) {
	if limit <= 0 {
		limit = 50
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invocations WHERE agent_id = $1", agentID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_id, user_id, protocol, status_code, duration_ms, error, ip_address, created_at
		 FROM invocations WHERE agent_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		agentID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var records []InvocationRecord
	for rows.Next() {
		var r InvocationRecord
		if err := rows.Scan(&r.ID, &r.AgentID, &r.UserID, &r.Protocol, &r.StatusCode, &r.DurationMs, &r.Error, &r.IPAddress, &r.CreatedAt); err != nil {
			return nil, 0, err
		}
		records = append(records, r)
	}
	return records, total, rows.Err()
}

func (s *PostgresStore) AgentStats(ctx context.Context, agentID string, since time.Time) (*AgentInvocationStats, error) {
	var stats AgentInvocationStats
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*),
		        SUM(CASE WHEN status_code >= 200 AND status_code < 400 THEN 1 ELSE 0 END),
		        SUM(CASE WHEN status_code >= 400 OR error != '' THEN 1 ELSE 0 END),
		        COALESCE(AVG(duration_ms), 0)
		 FROM invocations WHERE agent_id = $1 AND created_at >= $2`,
		agentID, since.UTC(),
	).Scan(&stats.TotalCalls, &stats.SuccessCalls, &stats.ErrorCalls, &stats.AvgDurationMs)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *PostgresStore) AgentTimeSeries(ctx context.Context, agentID string, since time.Time, bucketMinutes int) ([]TimeSeriesPoint, error) {
	if bucketMinutes <= 0 {
		bucketMinutes = 60
	}
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT
			date_trunc('hour', created_at) + (EXTRACT(minute FROM created_at)::int / %d * %d) * interval '1 minute' as bucket,
			COUNT(*),
			SUM(CASE WHEN status_code >= 200 AND status_code < 400 THEN 1 ELSE 0 END),
			SUM(CASE WHEN status_code >= 400 OR error != '' THEN 1 ELSE 0 END),
			COALESCE(AVG(duration_ms), 0)
		 FROM invocations WHERE agent_id = $1 AND created_at >= $2
		 GROUP BY bucket ORDER BY bucket`, bucketMinutes, bucketMinutes),
		agentID, since.UTC(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var points []TimeSeriesPoint
	for rows.Next() {
		var p TimeSeriesPoint
		if err := rows.Scan(&p.Timestamp, &p.TotalCalls, &p.SuccessCalls, &p.ErrorCalls, &p.AvgDurationMs); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (s *PostgresStore) ProviderDashboardStats(ctx context.Context, ownerUserID string) (*AgentInvocationStats, error) {
	var stats AgentInvocationStats
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*),
		        SUM(CASE WHEN i.status_code >= 200 AND i.status_code < 400 THEN 1 ELSE 0 END),
		        SUM(CASE WHEN i.status_code >= 400 OR i.error != '' THEN 1 ELSE 0 END),
		        COALESCE(AVG(i.duration_ms), 0)
		 FROM invocations i
		 INNER JOIN agents a ON a.id = i.agent_id
		 WHERE a.owner_user_id = $1`,
		ownerUserID,
	).Scan(&stats.TotalCalls, &stats.SuccessCalls, &stats.ErrorCalls, &stats.AvgDurationMs)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *PostgresStore) Close() error {
	return nil
}
