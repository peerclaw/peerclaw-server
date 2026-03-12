package blob

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SQLiteMetaStore implements MetaStore using SQLite.
type SQLiteMetaStore struct {
	db *sql.DB
}

// NewSQLiteMetaStore creates a new SQLite-backed blob metadata store.
func NewSQLiteMetaStore(db *sql.DB) *SQLiteMetaStore {
	return &SQLiteMetaStore{db: db}
}

func (s *SQLiteMetaStore) Migrate(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS blobs (
		id           TEXT PRIMARY KEY,
		filename     TEXT NOT NULL DEFAULT '',
		content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
		size         INTEGER NOT NULL DEFAULT 0,
		owner_id     TEXT NOT NULL,
		created_at   DATETIME NOT NULL,
		expires_at   DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_blobs_owner ON blobs(owner_id);
	CREATE INDEX IF NOT EXISTS idx_blobs_expires ON blobs(expires_at);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *SQLiteMetaStore) Create(ctx context.Context, meta *BlobMeta) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO blobs (id, filename, content_type, size, owner_id, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		meta.ID, meta.Filename, meta.ContentType, meta.Size, meta.OwnerID,
		meta.CreatedAt.UTC().Format(time.RFC3339),
		meta.ExpiresAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteMetaStore) Get(ctx context.Context, id string) (*BlobMeta, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, filename, content_type, size, owner_id, created_at, expires_at
		FROM blobs WHERE id = ?`, id)

	m := &BlobMeta{}
	var createdAt, expiresAt string

	err := row.Scan(&m.ID, &m.Filename, &m.ContentType, &m.Size, &m.OwnerID,
		&createdAt, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("blob not found")
		}
		return nil, err
	}

	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		m.CreatedAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, expiresAt); err == nil {
		m.ExpiresAt = parsed
	}
	return m, nil
}

func (s *SQLiteMetaStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM blobs WHERE id = ?`, id)
	return err
}

func (s *SQLiteMetaStore) ListExpired(ctx context.Context) ([]string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM blobs WHERE expires_at < ?`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *SQLiteMetaStore) SumSizeByOwner(ctx context.Context, ownerID string) (int64, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(size), 0) FROM blobs WHERE owner_id = ?`, ownerID)
	var total int64
	err := row.Scan(&total)
	return total, err
}

func (s *SQLiteMetaStore) Close() error {
	return nil // shared db, don't close
}
