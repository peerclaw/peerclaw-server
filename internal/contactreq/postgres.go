package contactreq

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

// NewPostgresStore creates a new PostgreSQL-backed contact request store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Migrate(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS agent_contact_requests (
		id              TEXT PRIMARY KEY,
		from_agent_id   TEXT NOT NULL,
		to_agent_id     TEXT NOT NULL,
		status          TEXT NOT NULL DEFAULT 'pending',
		message         TEXT DEFAULT '',
		reject_reason   TEXT DEFAULT '',
		created_at      TIMESTAMPTZ NOT NULL,
		updated_at      TIMESTAMPTZ NOT NULL,
		UNIQUE(from_agent_id, to_agent_id)
	);
	CREATE INDEX IF NOT EXISTS idx_contact_req_to ON agent_contact_requests(to_agent_id);
	CREATE INDEX IF NOT EXISTS idx_contact_req_from ON agent_contact_requests(from_agent_id);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *PostgresStore) Create(ctx context.Context, req *ContactRequest) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_contact_requests (id, from_agent_id, to_agent_id, status, message, reject_reason, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (from_agent_id, to_agent_id) DO UPDATE SET
			id = EXCLUDED.id, status = EXCLUDED.status, message = EXCLUDED.message,
			reject_reason = EXCLUDED.reject_reason, updated_at = EXCLUDED.updated_at`,
		req.ID, req.FromAgentID, req.ToAgentID, req.Status, req.Message, req.RejectReason,
		req.CreatedAt.UTC(), req.UpdatedAt.UTC(),
	)
	return err
}

func (s *PostgresStore) GetByID(ctx context.Context, id string) (*ContactRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, from_agent_id, to_agent_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
			created_at, updated_at
		FROM agent_contact_requests WHERE id = $1`, id)
	return s.scanRow(row)
}

func (s *PostgresStore) GetByAgents(ctx context.Context, fromAgentID, toAgentID string) (*ContactRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, from_agent_id, to_agent_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
			created_at, updated_at
		FROM agent_contact_requests WHERE from_agent_id = $1 AND to_agent_id = $2`, fromAgentID, toAgentID)
	return s.scanRow(row)
}

func (s *PostgresStore) ListByTarget(ctx context.Context, toAgentID string, status string) ([]ContactRequest, error) {
	query := `SELECT id, from_agent_id, to_agent_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
		created_at, updated_at
		FROM agent_contact_requests WHERE to_agent_id = $1`
	args := []any{toAgentID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	return s.queryRows(ctx, query, args...)
}

func (s *PostgresStore) ListBySender(ctx context.Context, fromAgentID string, status string) ([]ContactRequest, error) {
	query := `SELECT id, from_agent_id, to_agent_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
		created_at, updated_at
		FROM agent_contact_requests WHERE from_agent_id = $1`
	args := []any{fromAgentID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	return s.queryRows(ctx, query, args...)
}

func (s *PostgresStore) UpdateStatus(ctx context.Context, id, status, rejectReason string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE agent_contact_requests SET status = $1, reject_reason = $2, updated_at = $3
		WHERE id = $4`,
		status, rejectReason, time.Now().UTC(), id,
	)
	return err
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM agent_contact_requests WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("contact request not found")
	}
	return nil
}

func (s *PostgresStore) Close() error {
	return nil // shared db
}

func (s *PostgresStore) scanRow(row *sql.Row) (*ContactRequest, error) {
	var r ContactRequest
	err := row.Scan(&r.ID, &r.FromAgentID, &r.ToAgentID, &r.Status, &r.Message, &r.RejectReason,
		&r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (s *PostgresStore) queryRows(ctx context.Context, query string, args ...any) ([]ContactRequest, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ContactRequest
	for rows.Next() {
		var r ContactRequest
		if err := rows.Scan(&r.ID, &r.FromAgentID, &r.ToAgentID, &r.Status, &r.Message, &r.RejectReason,
			&r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
