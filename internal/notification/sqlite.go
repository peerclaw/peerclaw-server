package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed notification store.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Migrate(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS notifications (
		id         TEXT PRIMARY KEY,
		user_id    TEXT NOT NULL,
		agent_id   TEXT NOT NULL,
		type       TEXT NOT NULL,
		severity   TEXT NOT NULL DEFAULT 'info',
		title      TEXT NOT NULL,
		body       TEXT DEFAULT '',
		metadata   TEXT DEFAULT '{}',
		read       INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, read, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *SQLiteStore) Create(ctx context.Context, n *Notification) error {
	meta, err := json.Marshal(n.Metadata)
	if err != nil {
		meta = []byte("{}")
	}
	readInt := 0
	if n.Read {
		readInt = 1
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, agent_id, type, severity, title, body, metadata, read, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.UserID, n.AgentID, string(n.Type), string(n.Severity),
		n.Title, n.Body, string(meta), readInt,
		n.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetByID(ctx context.Context, id string) (*Notification, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, agent_id, type, severity, title, COALESCE(body, ''), metadata, read, created_at
		FROM notifications WHERE id = ?`, id)
	return s.scanRow(row)
}

func (s *SQLiteStore) ListByUser(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, int, error) {
	// Count total matching rows.
	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_id = ?`
	args := []any{userID}
	if unreadOnly {
		countQuery += " AND read = 0"
	}
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch rows.
	query := `SELECT id, user_id, agent_id, type, severity, title, COALESCE(body, ''), metadata, read, created_at
		FROM notifications WHERE user_id = ?`
	fetchArgs := []any{userID}
	if unreadOnly {
		query += " AND read = 0"
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	fetchArgs = append(fetchArgs, limit, offset)

	results, err := s.queryRows(ctx, query, fetchArgs...)
	if err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (s *SQLiteStore) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read = 0`, userID).Scan(&count)
	return count, err
}

func (s *SQLiteStore) MarkRead(ctx context.Context, id, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE notifications SET read = 1 WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

func (s *SQLiteStore) MarkAllRead(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE notifications SET read = 1 WHERE user_id = ? AND read = 0`, userID)
	return err
}

func (s *SQLiteStore) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM notifications WHERE created_at < ?`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) Close() error {
	return nil // shared db
}

func (s *SQLiteStore) scanRow(row *sql.Row) (*Notification, error) {
	var n Notification
	var metaStr string
	var readInt int
	var createdAt sql.NullString
	err := row.Scan(&n.ID, &n.UserID, &n.AgentID, &n.Type, &n.Severity,
		&n.Title, &n.Body, &metaStr, &readInt, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	n.Read = readInt != 0
	if createdAt.Valid {
		if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
			n.CreatedAt = t
		}
	}
	if metaStr != "" {
		_ = json.Unmarshal([]byte(metaStr), &n.Metadata)
	}
	return &n, nil
}

func (s *SQLiteStore) queryRows(ctx context.Context, query string, args ...any) ([]Notification, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Notification
	for rows.Next() {
		var n Notification
		var metaStr string
		var readInt int
		var createdAt sql.NullString
		if err := rows.Scan(&n.ID, &n.UserID, &n.AgentID, &n.Type, &n.Severity,
			&n.Title, &n.Body, &metaStr, &readInt, &createdAt); err != nil {
			return nil, err
		}
		n.Read = readInt != 0
		if createdAt.Valid {
			if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
				n.CreatedAt = t
			}
		}
		if metaStr != "" {
			_ = json.Unmarshal([]byte(metaStr), &n.Metadata)
		}
		results = append(results, n)
	}
	return results, rows.Err()
}
