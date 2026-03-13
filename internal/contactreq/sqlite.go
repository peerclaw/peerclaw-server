package contactreq

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

// NewSQLiteStore creates a new SQLite-backed contact request store.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Migrate(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS agent_contact_requests (
		id              TEXT PRIMARY KEY,
		from_agent_id   TEXT NOT NULL,
		to_agent_id     TEXT NOT NULL,
		status          TEXT NOT NULL DEFAULT 'pending',
		message         TEXT DEFAULT '',
		reject_reason   TEXT DEFAULT '',
		created_at      DATETIME NOT NULL,
		updated_at      DATETIME NOT NULL,
		UNIQUE(from_agent_id, to_agent_id)
	);
	CREATE INDEX IF NOT EXISTS idx_contact_req_to ON agent_contact_requests(to_agent_id);
	CREATE INDEX IF NOT EXISTS idx_contact_req_from ON agent_contact_requests(from_agent_id);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *SQLiteStore) Create(ctx context.Context, req *ContactRequest) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO agent_contact_requests (id, from_agent_id, to_agent_id, status, message, reject_reason, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ID, req.FromAgentID, req.ToAgentID, req.Status, req.Message, req.RejectReason,
		req.CreatedAt.UTC().Format(time.RFC3339),
		req.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetByID(ctx context.Context, id string) (*ContactRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, from_agent_id, to_agent_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
			created_at, updated_at
		FROM agent_contact_requests WHERE id = ?`, id)
	return s.scanRow(row)
}

func (s *SQLiteStore) GetByAgents(ctx context.Context, fromAgentID, toAgentID string) (*ContactRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, from_agent_id, to_agent_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
			created_at, updated_at
		FROM agent_contact_requests WHERE from_agent_id = ? AND to_agent_id = ?`, fromAgentID, toAgentID)
	return s.scanRow(row)
}

func (s *SQLiteStore) ListByTarget(ctx context.Context, toAgentID string, status string) ([]ContactRequest, error) {
	query := `SELECT id, from_agent_id, to_agent_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
		created_at, updated_at
		FROM agent_contact_requests WHERE to_agent_id = ?`
	args := []any{toAgentID}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	return s.queryRows(ctx, query, args...)
}

func (s *SQLiteStore) ListBySender(ctx context.Context, fromAgentID string, status string) ([]ContactRequest, error) {
	query := `SELECT id, from_agent_id, to_agent_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
		created_at, updated_at
		FROM agent_contact_requests WHERE from_agent_id = ?`
	args := []any{fromAgentID}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	return s.queryRows(ctx, query, args...)
}

func (s *SQLiteStore) UpdateStatus(ctx context.Context, id, status, rejectReason string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE agent_contact_requests SET status = ?, reject_reason = ?, updated_at = ?
		WHERE id = ?`,
		status, rejectReason,
		time.Now().UTC().Format(time.RFC3339), id,
	)
	return err
}

func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM agent_contact_requests WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("contact request not found")
	}
	return nil
}

func (s *SQLiteStore) Close() error {
	return nil // shared db
}

func (s *SQLiteStore) scanRow(row *sql.Row) (*ContactRequest, error) {
	var r ContactRequest
	var createdAt, updatedAt sql.NullString
	err := row.Scan(&r.ID, &r.FromAgentID, &r.ToAgentID, &r.Status, &r.Message, &r.RejectReason,
		&createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if createdAt.Valid {
		if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
			r.CreatedAt = t
		}
	}
	if updatedAt.Valid {
		if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
			r.UpdatedAt = t
		}
	}
	return &r, nil
}

func (s *SQLiteStore) queryRows(ctx context.Context, query string, args ...any) ([]ContactRequest, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ContactRequest
	for rows.Next() {
		var r ContactRequest
		var createdAt, updatedAt sql.NullString
		if err := rows.Scan(&r.ID, &r.FromAgentID, &r.ToAgentID, &r.Status, &r.Message, &r.RejectReason,
			&createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
				r.CreatedAt = t
			}
		}
		if updatedAt.Valid {
			if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
				r.UpdatedAt = t
			}
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
