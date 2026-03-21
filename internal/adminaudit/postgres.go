package adminaudit

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed admin audit store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS admin_audit_log (
			id            TEXT PRIMARY KEY,
			admin_user_id TEXT NOT NULL,
			action        TEXT NOT NULL,
			target_type   TEXT NOT NULL DEFAULT '',
			target_id     TEXT NOT NULL DEFAULT '',
			details       TEXT DEFAULT '',
			ip_address    TEXT DEFAULT '',
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_admin_audit_admin ON admin_audit_log(admin_user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_admin_audit_time ON admin_audit_log(created_at DESC)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("admin audit migrate: %w", err)
		}
	}
	return nil
}

func (s *PostgresStore) Insert(ctx context.Context, event *AdminAuditEvent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO admin_audit_log (id, admin_user_id, action, target_type, target_id, details, ip_address, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		event.ID, event.AdminUserID, event.Action, event.TargetType,
		event.TargetID, event.Details, event.IPAddress,
		event.CreatedAt.UTC(),
	)
	return err
}

func (s *PostgresStore) List(ctx context.Context, adminUserID, action, targetType string, since time.Time, limit, offset int) ([]AdminAuditEvent, int, error) {
	if limit <= 0 {
		limit = 50
	}

	var where []string
	var args []any
	paramIdx := 1

	if adminUserID != "" {
		where = append(where, fmt.Sprintf("admin_user_id = $%d", paramIdx))
		args = append(args, adminUserID)
		paramIdx++
	}
	if action != "" {
		where = append(where, fmt.Sprintf("action = $%d", paramIdx))
		args = append(args, action)
		paramIdx++
	}
	if targetType != "" {
		where = append(where, fmt.Sprintf("target_type = $%d", paramIdx))
		args = append(args, targetType)
		paramIdx++
	}
	if !since.IsZero() {
		where = append(where, fmt.Sprintf("created_at >= $%d", paramIdx))
		args = append(args, since.UTC())
		paramIdx++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// Count total.
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM admin_audit_log %s", whereClause)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch page.
	query := fmt.Sprintf(
		"SELECT id, admin_user_id, action, target_type, target_id, details, ip_address, created_at FROM admin_audit_log %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		whereClause, paramIdx, paramIdx+1,
	)
	pageArgs := append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, query, pageArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []AdminAuditEvent
	for rows.Next() {
		var e AdminAuditEvent
		if err := rows.Scan(&e.ID, &e.AdminUserID, &e.Action, &e.TargetType, &e.TargetID, &e.Details, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

func (s *PostgresStore) Close() error {
	return nil // shared db, don't close
}
