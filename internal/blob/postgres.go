package blob

import (
	"context"
	"database/sql"
	"fmt"
)

// PostgresMetaStore implements MetaStore using PostgreSQL.
type PostgresMetaStore struct {
	db *sql.DB
}

// NewPostgresMetaStore creates a new PostgreSQL-backed blob metadata store.
func NewPostgresMetaStore(db *sql.DB) *PostgresMetaStore {
	return &PostgresMetaStore{db: db}
}

func (s *PostgresMetaStore) Migrate(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS blobs (
		id           TEXT PRIMARY KEY,
		filename     TEXT NOT NULL DEFAULT '',
		content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
		size         BIGINT NOT NULL DEFAULT 0,
		owner_id     TEXT NOT NULL,
		created_at   TIMESTAMPTZ NOT NULL,
		expires_at   TIMESTAMPTZ NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_blobs_owner ON blobs(owner_id);
	CREATE INDEX IF NOT EXISTS idx_blobs_expires ON blobs(expires_at);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *PostgresMetaStore) Create(ctx context.Context, meta *BlobMeta) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO blobs (id, filename, content_type, size, owner_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		meta.ID, meta.Filename, meta.ContentType, meta.Size, meta.OwnerID,
		meta.CreatedAt.UTC(), meta.ExpiresAt.UTC(),
	)
	return err
}

func (s *PostgresMetaStore) Get(ctx context.Context, id string) (*BlobMeta, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, filename, content_type, size, owner_id, created_at, expires_at
		FROM blobs WHERE id = $1`, id)

	m := &BlobMeta{}
	err := row.Scan(&m.ID, &m.Filename, &m.ContentType, &m.Size, &m.OwnerID,
		&m.CreatedAt, &m.ExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("blob not found")
		}
		return nil, err
	}
	return m, nil
}

func (s *PostgresMetaStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM blobs WHERE id = $1`, id)
	return err
}

func (s *PostgresMetaStore) ListExpired(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM blobs WHERE expires_at < NOW()`)
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

func (s *PostgresMetaStore) SumSizeByOwner(ctx context.Context, ownerID string) (int64, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(size), 0) FROM blobs WHERE owner_id = $1`, ownerID)
	var total int64
	err := row.Scan(&total)
	return total, err
}

func (s *PostgresMetaStore) Close() error {
	return nil // shared db, don't close
}
