package claimtoken

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

// NewSQLiteStore creates a new SQLite-backed claim token store.
// It expects a shared *sql.DB (from the registry store).
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Migrate(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS claim_tokens (
		id         TEXT PRIMARY KEY,
		code       TEXT NOT NULL UNIQUE,
		user_id    TEXT NOT NULL,
		status     TEXT NOT NULL DEFAULT 'pending',
		agent_id   TEXT DEFAULT '',
		created_at DATETIME NOT NULL,
		expires_at DATETIME NOT NULL,
		claimed_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_claim_tokens_code ON claim_tokens(code);
	CREATE INDEX IF NOT EXISTS idx_claim_tokens_user ON claim_tokens(user_id);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *SQLiteStore) Create(ctx context.Context, token *ClaimToken) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO claim_tokens (id, code, user_id, status, agent_id, created_at, expires_at, claimed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		token.ID, token.Code, token.UserID, token.Status, token.AgentID,
		token.CreatedAt.UTC().Format(time.RFC3339),
		token.ExpiresAt.UTC().Format(time.RFC3339),
		nil,
	)
	return err
}

func (s *SQLiteStore) GetByCode(ctx context.Context, code string) (*ClaimToken, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, code, user_id, status, COALESCE(agent_id, ''), created_at, expires_at, claimed_at
		FROM claim_tokens WHERE code = ?`, code)

	return s.scanToken(row)
}

func (s *SQLiteStore) MarkClaimed(ctx context.Context, code, agentID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		UPDATE claim_tokens SET status = ?, agent_id = ?, claimed_at = ?
		WHERE code = ? AND status = ?`,
		StatusClaimed, agentID, now, code, StatusPending,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("token not found or already claimed")
	}
	return nil
}

func (s *SQLiteStore) ListByUser(ctx context.Context, userID string) ([]ClaimToken, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, code, user_id, status, COALESCE(agent_id, ''), created_at, expires_at, claimed_at
		FROM claim_tokens WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tokens []ClaimToken
	for rows.Next() {
		t, err := s.scanTokenFromRows(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, *t)
	}
	return tokens, rows.Err()
}

func (s *SQLiteStore) DeleteExpired(ctx context.Context) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM claim_tokens WHERE status = ? AND expires_at < ?`,
		StatusPending, now,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) Close() error {
	return nil // shared db, don't close
}

func (s *SQLiteStore) scanToken(row *sql.Row) (*ClaimToken, error) {
	t := &ClaimToken{}
	var createdAt, expiresAt string
	var claimedAt sql.NullString

	err := row.Scan(&t.ID, &t.Code, &t.UserID, &t.Status, &t.AgentID,
		&createdAt, &expiresAt, &claimedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("claim token not found")
		}
		return nil, err
	}

	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		t.CreatedAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, expiresAt); err == nil {
		t.ExpiresAt = parsed
	}
	if claimedAt.Valid {
		if parsed, err := time.Parse(time.RFC3339, claimedAt.String); err == nil {
			t.ClaimedAt = &parsed
		}
	}
	return t, nil
}

func (s *SQLiteStore) scanTokenFromRows(rows *sql.Rows) (*ClaimToken, error) {
	t := &ClaimToken{}
	var createdAt, expiresAt string
	var claimedAt sql.NullString

	err := rows.Scan(&t.ID, &t.Code, &t.UserID, &t.Status, &t.AgentID,
		&createdAt, &expiresAt, &claimedAt)
	if err != nil {
		return nil, err
	}

	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		t.CreatedAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, expiresAt); err == nil {
		t.ExpiresAt = parsed
	}
	if claimedAt.Valid {
		if parsed, err := time.Parse(time.RFC3339, claimedAt.String); err == nil {
			t.ClaimedAt = &parsed
		}
	}
	return t, nil
}
