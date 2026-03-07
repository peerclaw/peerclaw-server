package verification

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

// NewPostgresStore creates a new PostgreSQL-backed verification store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Migrate creates the verification_challenges table.
func (s *PostgresStore) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS verification_challenges (
			agent_id    TEXT NOT NULL,
			challenge   TEXT NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL,
			expires_at  TIMESTAMPTZ NOT NULL,
			status      TEXT DEFAULT 'pending',
			PRIMARY KEY (agent_id, challenge)
		)
	`)
	return err
}

// InsertChallenge creates a new verification challenge.
func (s *PostgresStore) InsertChallenge(ctx context.Context, ch *Challenge) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO verification_challenges (agent_id, challenge, created_at, expires_at, status)
		 VALUES ($1, $2, $3, $4, $5)`,
		ch.AgentID, ch.Challenge,
		ch.CreatedAt.UTC(),
		ch.ExpiresAt.UTC(),
		string(ch.Status),
	)
	return err
}

// GetPendingChallenge retrieves a pending challenge that hasn't expired.
func (s *PostgresStore) GetPendingChallenge(ctx context.Context, agentID, nonce string) (*Challenge, error) {
	var ch Challenge
	var status string
	err := s.db.QueryRowContext(ctx,
		`SELECT agent_id, challenge, created_at, expires_at, status
		 FROM verification_challenges
		 WHERE agent_id = $1 AND challenge = $2 AND status = 'pending' AND expires_at > $3`,
		agentID, nonce, time.Now().UTC(),
	).Scan(&ch.AgentID, &ch.Challenge, &ch.CreatedAt, &ch.ExpiresAt, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no pending challenge found")
		}
		return nil, err
	}
	ch.Status = ChallengeStatus(status)
	return &ch, nil
}

// UpdateChallengeStatus updates the status of a challenge.
func (s *PostgresStore) UpdateChallengeStatus(ctx context.Context, agentID, nonce string, status ChallengeStatus) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE verification_challenges SET status = $1 WHERE agent_id = $2 AND challenge = $3`,
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
func (s *PostgresStore) CleanExpired(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM verification_challenges WHERE expires_at < $1`,
		time.Now().UTC(),
	)
	return err
}

// Close is a no-op since the db is shared.
func (s *PostgresStore) Close() error {
	return nil
}
