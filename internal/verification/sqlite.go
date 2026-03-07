package verification

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

// NewSQLiteStore creates a new SQLite-backed verification store.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// Migrate creates the verification_challenges table.
func (s *SQLiteStore) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS verification_challenges (
			agent_id    TEXT NOT NULL,
			challenge   TEXT NOT NULL,
			created_at  DATETIME NOT NULL,
			expires_at  DATETIME NOT NULL,
			status      TEXT DEFAULT 'pending',
			PRIMARY KEY (agent_id, challenge)
		)
	`)
	return err
}

// InsertChallenge creates a new verification challenge.
func (s *SQLiteStore) InsertChallenge(ctx context.Context, ch *Challenge) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO verification_challenges (agent_id, challenge, created_at, expires_at, status)
		 VALUES (?, ?, ?, ?, ?)`,
		ch.AgentID, ch.Challenge,
		ch.CreatedAt.UTC().Format(time.RFC3339),
		ch.ExpiresAt.UTC().Format(time.RFC3339),
		string(ch.Status),
	)
	return err
}

// GetPendingChallenge retrieves a pending challenge that hasn't expired.
func (s *SQLiteStore) GetPendingChallenge(ctx context.Context, agentID, nonce string) (*Challenge, error) {
	var ch Challenge
	var createdAt, expiresAt, status string
	err := s.db.QueryRowContext(ctx,
		`SELECT agent_id, challenge, created_at, expires_at, status
		 FROM verification_challenges
		 WHERE agent_id = ? AND challenge = ? AND status = 'pending' AND expires_at > ?`,
		agentID, nonce, time.Now().UTC().Format(time.RFC3339),
	).Scan(&ch.AgentID, &ch.Challenge, &createdAt, &expiresAt, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no pending challenge found")
		}
		return nil, err
	}
	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		ch.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, expiresAt); err == nil {
		ch.ExpiresAt = t
	}
	ch.Status = ChallengeStatus(status)
	return &ch, nil
}

// UpdateChallengeStatus updates the status of a challenge.
func (s *SQLiteStore) UpdateChallengeStatus(ctx context.Context, agentID, nonce string, status ChallengeStatus) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE verification_challenges SET status = ? WHERE agent_id = ? AND challenge = ?`,
		string(status), agentID, nonce,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("challenge not found")
	}
	return nil
}

// CleanExpired removes expired challenges.
func (s *SQLiteStore) CleanExpired(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM verification_challenges WHERE expires_at < ?`,
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// Close is a no-op since the db is shared.
func (s *SQLiteStore) Close() error {
	return nil
}
