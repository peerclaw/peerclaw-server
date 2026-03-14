package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed notification store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Migrate(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS notifications (
		id         TEXT PRIMARY KEY,
		user_id    TEXT NOT NULL,
		agent_id   TEXT NOT NULL,
		type       TEXT NOT NULL,
		severity   TEXT NOT NULL DEFAULT 'info',
		title      TEXT NOT NULL,
		body       TEXT DEFAULT '',
		metadata   JSONB DEFAULT '{}',
		read       BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMPTZ NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, read, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *PostgresStore) Create(ctx context.Context, n *Notification) error {
	meta, err := json.Marshal(n.Metadata)
	if err != nil {
		meta = []byte("{}")
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, agent_id, type, severity, title, body, metadata, read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		n.ID, n.UserID, n.AgentID, string(n.Type), string(n.Severity),
		n.Title, n.Body, string(meta), n.Read,
		n.CreatedAt.UTC(),
	)
	return err
}

func (s *PostgresStore) GetByID(ctx context.Context, id string) (*Notification, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, agent_id, type, severity, title, COALESCE(body, ''), metadata, read, created_at
		FROM notifications WHERE id = $1`, id)
	return s.scanRow(row)
}

func (s *PostgresStore) ListByUser(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, int, error) {
	// Count total matching rows.
	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_id = $1`
	countArgs := []any{userID}
	if unreadOnly {
		countQuery += " AND read = FALSE"
	}
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch rows.
	query := `SELECT id, user_id, agent_id, type, severity, title, COALESCE(body, ''), metadata, read, created_at
		FROM notifications WHERE user_id = $1`
	fetchArgs := []any{userID}
	if unreadOnly {
		query += " AND read = FALSE"
	}
	query += " ORDER BY created_at DESC LIMIT $2 OFFSET $3"
	if unreadOnly {
		query = `SELECT id, user_id, agent_id, type, severity, title, COALESCE(body, ''), metadata, read, created_at
			FROM notifications WHERE user_id = $1 AND read = FALSE
			ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	}
	fetchArgs = append(fetchArgs, limit, offset)

	results, err := s.queryRows(ctx, query, fetchArgs...)
	if err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (s *PostgresStore) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE`, userID).Scan(&count)
	return count, err
}

func (s *PostgresStore) MarkRead(ctx context.Context, id, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE notifications SET read = TRUE WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (s *PostgresStore) MarkAllRead(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE notifications SET read = TRUE WHERE user_id = $1 AND read = FALSE`, userID)
	return err
}

func (s *PostgresStore) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM notifications WHERE created_at < $1`, cutoff.UTC())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *PostgresStore) Close() error {
	return nil // shared db
}

func (s *PostgresStore) scanRow(row *sql.Row) (*Notification, error) {
	var n Notification
	var metaStr string
	err := row.Scan(&n.ID, &n.UserID, &n.AgentID, &n.Type, &n.Severity,
		&n.Title, &n.Body, &metaStr, &n.Read, &n.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if metaStr != "" {
		_ = json.Unmarshal([]byte(metaStr), &n.Metadata)
	}
	return &n, nil
}

func (s *PostgresStore) queryRows(ctx context.Context, query string, args ...any) ([]Notification, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Notification
	for rows.Next() {
		var n Notification
		var metaStr string
		if err := rows.Scan(&n.ID, &n.UserID, &n.AgentID, &n.Type, &n.Severity,
			&n.Title, &n.Body, &metaStr, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		if metaStr != "" {
			_ = json.Unmarshal([]byte(metaStr), &n.Metadata)
		}
		results = append(results, n)
	}
	return results, rows.Err()
}
