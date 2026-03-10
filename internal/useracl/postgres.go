package useracl

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

// NewPostgresStore creates a new PostgreSQL-backed user ACL store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Migrate(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS agent_user_acl (
		id            TEXT PRIMARY KEY,
		agent_id      TEXT NOT NULL,
		user_id       TEXT NOT NULL,
		status        TEXT NOT NULL DEFAULT 'pending',
		message       TEXT DEFAULT '',
		reject_reason TEXT DEFAULT '',
		expires_at    TIMESTAMPTZ,
		created_at    TIMESTAMPTZ NOT NULL,
		updated_at    TIMESTAMPTZ NOT NULL,
		UNIQUE(agent_id, user_id)
	);
	CREATE INDEX IF NOT EXISTS idx_user_acl_agent ON agent_user_acl(agent_id);
	CREATE INDEX IF NOT EXISTS idx_user_acl_user ON agent_user_acl(user_id);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *PostgresStore) Create(ctx context.Context, req *AccessRequest) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_user_acl (id, agent_id, user_id, status, message, reject_reason, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (agent_id, user_id) DO UPDATE SET
			id = EXCLUDED.id, status = EXCLUDED.status, message = EXCLUDED.message,
			reject_reason = EXCLUDED.reject_reason, expires_at = EXCLUDED.expires_at,
			updated_at = EXCLUDED.updated_at`,
		req.ID, req.AgentID, req.UserID, req.Status, req.Message, req.RejectReason,
		req.ExpiresAt, req.CreatedAt.UTC(), req.UpdatedAt.UTC(),
	)
	return err
}

func (s *PostgresStore) GetByID(ctx context.Context, id string) (*AccessRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, agent_id, user_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
			expires_at, created_at, updated_at
		FROM agent_user_acl WHERE id = $1`, id)
	return s.scanRow(row)
}

func (s *PostgresStore) GetByAgentAndUser(ctx context.Context, agentID, userID string) (*AccessRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, agent_id, user_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
			expires_at, created_at, updated_at
		FROM agent_user_acl WHERE agent_id = $1 AND user_id = $2`, agentID, userID)
	return s.scanRow(row)
}

func (s *PostgresStore) ListByAgent(ctx context.Context, agentID string, status string) ([]AccessRequest, error) {
	query := `SELECT id, agent_id, user_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
		expires_at, created_at, updated_at
		FROM agent_user_acl WHERE agent_id = $1`
	args := []any{agentID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	return s.queryRows(ctx, query, args...)
}

func (s *PostgresStore) ListByUser(ctx context.Context, userID string) ([]AccessRequest, error) {
	return s.queryRows(ctx, `
		SELECT id, agent_id, user_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
			expires_at, created_at, updated_at
		FROM agent_user_acl WHERE user_id = $1 ORDER BY created_at DESC`, userID)
}

func (s *PostgresStore) UpdateStatus(ctx context.Context, id, status, rejectReason string, expiresAt *time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE agent_user_acl SET status = $1, reject_reason = $2, expires_at = $3, updated_at = $4
		WHERE id = $5`,
		status, rejectReason, expiresAt, time.Now().UTC(), id,
	)
	return err
}

func (s *PostgresStore) IsAllowed(ctx context.Context, agentID, userID string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM agent_user_acl
		WHERE agent_id = $1 AND user_id = $2 AND status = 'approved'
		AND (expires_at IS NULL OR expires_at > NOW())`,
		agentID, userID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM agent_user_acl WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("access request not found")
	}
	return nil
}

func (s *PostgresStore) Close() error {
	return nil // shared db
}

func (s *PostgresStore) scanRow(row *sql.Row) (*AccessRequest, error) {
	var r AccessRequest
	var expiresAt sql.NullTime
	err := row.Scan(&r.ID, &r.AgentID, &r.UserID, &r.Status, &r.Message, &r.RejectReason,
		&expiresAt, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if expiresAt.Valid {
		r.ExpiresAt = &expiresAt.Time
	}
	return &r, nil
}

func (s *PostgresStore) queryRows(ctx context.Context, query string, args ...any) ([]AccessRequest, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []AccessRequest
	for rows.Next() {
		var r AccessRequest
		var expiresAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.AgentID, &r.UserID, &r.Status, &r.Message, &r.RejectReason,
			&expiresAt, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			r.ExpiresAt = &expiresAt.Time
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
