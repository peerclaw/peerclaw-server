package claimtoken

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

// NewPostgresStore creates a new PostgreSQL-backed claim token store.
// It expects a shared *sql.DB (from the registry store).
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Migrate(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS claim_tokens (
		id         TEXT PRIMARY KEY,
		code       TEXT NOT NULL UNIQUE,
		user_id    TEXT NOT NULL,
		status     TEXT NOT NULL DEFAULT 'pending',
		agent_id   TEXT DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		claimed_at TIMESTAMPTZ
	);
	CREATE INDEX IF NOT EXISTS idx_claim_tokens_code ON claim_tokens(code);
	CREATE INDEX IF NOT EXISTS idx_claim_tokens_user ON claim_tokens(user_id);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *PostgresStore) Create(ctx context.Context, token *ClaimToken) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO claim_tokens (id, code, user_id, status, agent_id, created_at, expires_at, claimed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		token.ID, token.Code, token.UserID, token.Status, token.AgentID,
		token.CreatedAt.UTC(), token.ExpiresAt.UTC(), nil,
	)
	return err
}

func (s *PostgresStore) GetByCode(ctx context.Context, code string) (*ClaimToken, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, code, user_id, status, COALESCE(agent_id, ''), created_at, expires_at, claimed_at
		FROM claim_tokens WHERE code = $1`, code)

	return s.scanToken(row)
}

func (s *PostgresStore) MarkClaimed(ctx context.Context, code, agentID string) error {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE claim_tokens SET status = $1, agent_id = $2, claimed_at = $3
		WHERE code = $4 AND status = $5`,
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

func (s *PostgresStore) ListByUser(ctx context.Context, userID string) ([]ClaimToken, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, code, user_id, status, COALESCE(agent_id, ''), created_at, expires_at, claimed_at
		FROM claim_tokens WHERE user_id = $1 ORDER BY created_at DESC`, userID)
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

func (s *PostgresStore) DeleteExpired(ctx context.Context) (int64, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM claim_tokens WHERE status = $1 AND expires_at < $2`,
		StatusPending, now,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *PostgresStore) Close() error {
	return nil // shared db, don't close
}

func (s *PostgresStore) scanToken(row *sql.Row) (*ClaimToken, error) {
	t := &ClaimToken{}
	var claimedAt sql.NullTime

	err := row.Scan(&t.ID, &t.Code, &t.UserID, &t.Status, &t.AgentID,
		&t.CreatedAt, &t.ExpiresAt, &claimedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("claim token not found")
		}
		return nil, err
	}
	if claimedAt.Valid {
		t.ClaimedAt = &claimedAt.Time
	}
	return t, nil
}

func (s *PostgresStore) scanTokenFromRows(rows *sql.Rows) (*ClaimToken, error) {
	t := &ClaimToken{}
	var claimedAt sql.NullTime

	err := rows.Scan(&t.ID, &t.Code, &t.UserID, &t.Status, &t.AgentID,
		&t.CreatedAt, &t.ExpiresAt, &claimedAt)
	if err != nil {
		return nil, err
	}
	if claimedAt.Valid {
		t.ClaimedAt = &claimedAt.Time
	}
	return t, nil
}
