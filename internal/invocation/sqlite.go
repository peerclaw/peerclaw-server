package invocation

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

// NewSQLiteStore creates a new SQLite-backed invocation store.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS invocations (
			id            TEXT PRIMARY KEY,
			agent_id      TEXT NOT NULL,
			user_id       TEXT DEFAULT '',
			protocol      TEXT DEFAULT '',
			request_body  TEXT DEFAULT '',
			response_body TEXT DEFAULT '',
			status_code   INTEGER DEFAULT 0,
			duration_ms   INTEGER DEFAULT 0,
			error         TEXT DEFAULT '',
			ip_address    TEXT DEFAULT '',
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
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

func (s *SQLiteStore) Insert(ctx context.Context, record *InvocationRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO invocations (id, agent_id, user_id, protocol, request_body, response_body, status_code, duration_ms, error, ip_address, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.AgentID, record.UserID, record.Protocol,
		record.RequestBody, record.ResponseBody, record.StatusCode,
		record.DurationMs, record.Error, record.IPAddress,
		record.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetByID(ctx context.Context, id string) (*InvocationRecord, error) {
	var r InvocationRecord
	var createdAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, agent_id, user_id, protocol, request_body, response_body, status_code, duration_ms, error, ip_address, created_at
		 FROM invocations WHERE id = ?`, id,
	).Scan(&r.ID, &r.AgentID, &r.UserID, &r.Protocol, &r.RequestBody, &r.ResponseBody,
		&r.StatusCode, &r.DurationMs, &r.Error, &r.IPAddress, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invocation not found")
		}
		return nil, err
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &r, nil
}

func (s *SQLiteStore) ListByUser(ctx context.Context, userID string, limit, offset int) ([]InvocationRecord, int, error) {
	if limit <= 0 {
		limit = 50
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invocations WHERE user_id = ?", userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_id, user_id, protocol, status_code, duration_ms, error, ip_address, created_at
		 FROM invocations WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var records []InvocationRecord
	for rows.Next() {
		var r InvocationRecord
		var createdAt string
		if err := rows.Scan(&r.ID, &r.AgentID, &r.UserID, &r.Protocol, &r.StatusCode, &r.DurationMs, &r.Error, &r.IPAddress, &createdAt); err != nil {
			return nil, 0, err
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		records = append(records, r)
	}
	return records, total, rows.Err()
}

func (s *SQLiteStore) ListByAgent(ctx context.Context, agentID string, limit, offset int) ([]InvocationRecord, int, error) {
	if limit <= 0 {
		limit = 50
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invocations WHERE agent_id = ?", agentID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_id, user_id, protocol, status_code, duration_ms, error, ip_address, created_at
		 FROM invocations WHERE agent_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		agentID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var records []InvocationRecord
	for rows.Next() {
		var r InvocationRecord
		var createdAt string
		if err := rows.Scan(&r.ID, &r.AgentID, &r.UserID, &r.Protocol, &r.StatusCode, &r.DurationMs, &r.Error, &r.IPAddress, &createdAt); err != nil {
			return nil, 0, err
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		records = append(records, r)
	}
	return records, total, rows.Err()
}

func (s *SQLiteStore) AgentStats(ctx context.Context, agentID string, since time.Time) (*AgentInvocationStats, error) {
	var stats AgentInvocationStats
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*),
		        SUM(CASE WHEN status_code >= 200 AND status_code < 400 THEN 1 ELSE 0 END),
		        SUM(CASE WHEN status_code >= 400 OR error != '' THEN 1 ELSE 0 END),
		        COALESCE(AVG(duration_ms), 0)
		 FROM invocations WHERE agent_id = ? AND created_at >= ?`,
		agentID, since.UTC().Format(time.RFC3339),
	).Scan(&stats.TotalCalls, &stats.SuccessCalls, &stats.ErrorCalls, &stats.AvgDurationMs)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *SQLiteStore) AgentTimeSeries(ctx context.Context, agentID string, since time.Time, bucketMinutes int) ([]TimeSeriesPoint, error) {
	if bucketMinutes <= 0 {
		bucketMinutes = 60
	}
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT
			strftime('%%Y-%%m-%%dT%%H:%%M:00Z', created_at, 'utc', '-' || (strftime('%%M', created_at) %% %d) || ' minutes') as bucket,
			COUNT(*),
			SUM(CASE WHEN status_code >= 200 AND status_code < 400 THEN 1 ELSE 0 END),
			SUM(CASE WHEN status_code >= 400 OR error != '' THEN 1 ELSE 0 END),
			COALESCE(AVG(duration_ms), 0)
		 FROM invocations WHERE agent_id = ? AND created_at >= ?
		 GROUP BY bucket ORDER BY bucket`, bucketMinutes),
		agentID, since.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var points []TimeSeriesPoint
	for rows.Next() {
		var p TimeSeriesPoint
		var ts string
		if err := rows.Scan(&ts, &p.TotalCalls, &p.SuccessCalls, &p.ErrorCalls, &p.AvgDurationMs); err != nil {
			return nil, err
		}
		p.Timestamp, _ = time.Parse(time.RFC3339, ts)
		points = append(points, p)
	}
	return points, rows.Err()
}

func (s *SQLiteStore) ProviderDashboardStats(ctx context.Context, ownerUserID string) (*AgentInvocationStats, error) {
	var stats AgentInvocationStats
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*),
		        SUM(CASE WHEN i.status_code >= 200 AND i.status_code < 400 THEN 1 ELSE 0 END),
		        SUM(CASE WHEN i.status_code >= 400 OR i.error != '' THEN 1 ELSE 0 END),
		        COALESCE(AVG(i.duration_ms), 0)
		 FROM invocations i
		 INNER JOIN agents a ON a.id = i.agent_id
		 WHERE a.owner_user_id = ?`,
		ownerUserID,
	).Scan(&stats.TotalCalls, &stats.SuccessCalls, &stats.ErrorCalls, &stats.AvgDurationMs)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *SQLiteStore) ListAll(ctx context.Context, agentID, userID string, limit, offset int) ([]InvocationRecord, int, error) {
	if limit <= 0 {
		limit = 50
	}

	where := "1=1"
	var args []interface{}
	if agentID != "" {
		where += " AND agent_id = ?"
		args = append(args, agentID)
	}
	if userID != "" {
		where += " AND user_id = ?"
		args = append(args, userID)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invocations WHERE "+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, agent_id, user_id, protocol, status_code, duration_ms, error, ip_address, created_at FROM invocations WHERE "+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var records []InvocationRecord
	for rows.Next() {
		var r InvocationRecord
		var createdAt string
		if err := rows.Scan(&r.ID, &r.AgentID, &r.UserID, &r.Protocol, &r.StatusCode, &r.DurationMs, &r.Error, &r.IPAddress, &createdAt); err != nil {
			return nil, 0, err
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		records = append(records, r)
	}
	return records, total, rows.Err()
}

func (s *SQLiteStore) GlobalStats(ctx context.Context, since time.Time) (*AgentInvocationStats, error) {
	var stats AgentInvocationStats
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*),
		        SUM(CASE WHEN status_code >= 200 AND status_code < 400 THEN 1 ELSE 0 END),
		        SUM(CASE WHEN status_code >= 400 OR error != '' THEN 1 ELSE 0 END),
		        COALESCE(AVG(duration_ms), 0)
		 FROM invocations WHERE created_at >= ?`,
		since.UTC().Format(time.RFC3339),
	).Scan(&stats.TotalCalls, &stats.SuccessCalls, &stats.ErrorCalls, &stats.AvgDurationMs)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *SQLiteStore) GlobalTimeSeries(ctx context.Context, since time.Time, bucketMinutes int) ([]TimeSeriesPoint, error) {
	if bucketMinutes <= 0 {
		bucketMinutes = 60
	}
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT
			strftime('%%Y-%%m-%%dT%%H:%%M:00Z', created_at, 'utc', '-' || (strftime('%%M', created_at) %% %d) || ' minutes') as bucket,
			COUNT(*),
			SUM(CASE WHEN status_code >= 200 AND status_code < 400 THEN 1 ELSE 0 END),
			SUM(CASE WHEN status_code >= 400 OR error != '' THEN 1 ELSE 0 END),
			COALESCE(AVG(duration_ms), 0)
		 FROM invocations WHERE created_at >= ?
		 GROUP BY bucket ORDER BY bucket`, bucketMinutes),
		since.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var points []TimeSeriesPoint
	for rows.Next() {
		var p TimeSeriesPoint
		var ts string
		if err := rows.Scan(&ts, &p.TotalCalls, &p.SuccessCalls, &p.ErrorCalls, &p.AvgDurationMs); err != nil {
			return nil, err
		}
		p.Timestamp, _ = time.Parse(time.RFC3339, ts)
		points = append(points, p)
	}
	return points, rows.Err()
}

func (s *SQLiteStore) TopAgents(ctx context.Context, since time.Time, limit int) ([]AgentCallStats, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT i.agent_id, COALESCE(a.name, i.agent_id),
		        COUNT(*),
		        SUM(CASE WHEN i.status_code >= 200 AND i.status_code < 400 THEN 1 ELSE 0 END),
		        SUM(CASE WHEN i.status_code >= 400 OR i.error != '' THEN 1 ELSE 0 END),
		        COALESCE(AVG(i.duration_ms), 0)
		 FROM invocations i
		 LEFT JOIN agents a ON a.id = i.agent_id
		 WHERE i.created_at >= ?
		 GROUP BY i.agent_id
		 ORDER BY COUNT(*) DESC LIMIT ?`,
		since.UTC().Format(time.RFC3339), limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var stats []AgentCallStats
	for rows.Next() {
		var s AgentCallStats
		if err := rows.Scan(&s.AgentID, &s.AgentName, &s.TotalCalls, &s.SuccessCalls, &s.ErrorCalls, &s.AvgDurationMs); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (s *SQLiteStore) CountInvocations(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invocations").Scan(&count)
	return count, err
}

func (s *SQLiteStore) Close() error {
	return nil
}
