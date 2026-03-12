package useracl

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

// NewSQLiteStore creates a new SQLite-backed user ACL store.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Migrate(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS agent_user_acl (
		id            TEXT PRIMARY KEY,
		agent_id      TEXT NOT NULL,
		user_id       TEXT NOT NULL,
		status        TEXT NOT NULL DEFAULT 'pending',
		message       TEXT DEFAULT '',
		reject_reason TEXT DEFAULT '',
		expires_at    DATETIME,
		created_at    DATETIME NOT NULL,
		updated_at    DATETIME NOT NULL,
		UNIQUE(agent_id, user_id)
	);
	CREATE INDEX IF NOT EXISTS idx_user_acl_agent ON agent_user_acl(agent_id);
	CREATE INDEX IF NOT EXISTS idx_user_acl_user ON agent_user_acl(user_id);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *SQLiteStore) Create(ctx context.Context, req *AccessRequest) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO agent_user_acl (id, agent_id, user_id, status, message, reject_reason, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ID, req.AgentID, req.UserID, req.Status, req.Message, req.RejectReason,
		nullTimeStr(req.ExpiresAt),
		req.CreatedAt.UTC().Format(time.RFC3339),
		req.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetByID(ctx context.Context, id string) (*AccessRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, agent_id, user_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
			expires_at, created_at, updated_at
		FROM agent_user_acl WHERE id = ?`, id)
	return s.scanRow(row)
}

func (s *SQLiteStore) GetByAgentAndUser(ctx context.Context, agentID, userID string) (*AccessRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, agent_id, user_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
			expires_at, created_at, updated_at
		FROM agent_user_acl WHERE agent_id = ? AND user_id = ?`, agentID, userID)
	return s.scanRow(row)
}

func (s *SQLiteStore) ListByAgent(ctx context.Context, agentID string, status string) ([]AccessRequest, error) {
	query := `SELECT id, agent_id, user_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
		expires_at, created_at, updated_at
		FROM agent_user_acl WHERE agent_id = ?`
	args := []any{agentID}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	return s.queryRows(ctx, query, args...)
}

func (s *SQLiteStore) ListByUser(ctx context.Context, userID string) ([]AccessRequest, error) {
	return s.queryRows(ctx, `
		SELECT id, agent_id, user_id, status, COALESCE(message, ''), COALESCE(reject_reason, ''),
			expires_at, created_at, updated_at
		FROM agent_user_acl WHERE user_id = ? ORDER BY created_at DESC`, userID)
}

func (s *SQLiteStore) UpdateStatus(ctx context.Context, id, status, rejectReason string, expiresAt *time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE agent_user_acl SET status = ?, reject_reason = ?, expires_at = ?, updated_at = ?
		WHERE id = ?`,
		status, rejectReason, nullTimeStr(expiresAt),
		time.Now().UTC().Format(time.RFC3339), id,
	)
	return err
}

func (s *SQLiteStore) IsAllowed(ctx context.Context, agentID, userID string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM agent_user_acl
		WHERE agent_id = ? AND user_id = ? AND status = 'approved'
		AND (expires_at IS NULL OR expires_at > ?)`,
		agentID, userID, time.Now().UTC().Format(time.RFC3339),
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM agent_user_acl WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("access request not found")
	}
	return nil
}

func (s *SQLiteStore) Close() error {
	return nil // shared db
}

func (s *SQLiteStore) scanRow(row *sql.Row) (*AccessRequest, error) {
	var r AccessRequest
	var expiresAt, createdAt, updatedAt sql.NullString
	err := row.Scan(&r.ID, &r.AgentID, &r.UserID, &r.Status, &r.Message, &r.RejectReason,
		&expiresAt, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if expiresAt.Valid {
		if t, err := time.Parse(time.RFC3339, expiresAt.String); err == nil {
			r.ExpiresAt = &t
		}
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

func (s *SQLiteStore) queryRows(ctx context.Context, query string, args ...any) ([]AccessRequest, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AccessRequest
	for rows.Next() {
		var r AccessRequest
		var expiresAt, createdAt, updatedAt sql.NullString
		if err := rows.Scan(&r.ID, &r.AgentID, &r.UserID, &r.Status, &r.Message, &r.RejectReason,
			&expiresAt, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			if t, err := time.Parse(time.RFC3339, expiresAt.String); err == nil {
				r.ExpiresAt = &t
			}
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

func nullTimeStr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
