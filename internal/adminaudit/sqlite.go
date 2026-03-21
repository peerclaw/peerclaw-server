package adminaudit

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed admin audit store.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS admin_audit_log (
			id            TEXT PRIMARY KEY,
			admin_user_id TEXT NOT NULL,
			action        TEXT NOT NULL,
			target_type   TEXT NOT NULL DEFAULT '',
			target_id     TEXT NOT NULL DEFAULT '',
			details       TEXT DEFAULT '',
			ip_address    TEXT DEFAULT '',
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
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

func (s *SQLiteStore) Insert(ctx context.Context, event *AdminAuditEvent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO admin_audit_log (id, admin_user_id, action, target_type, target_id, details, ip_address, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.AdminUserID, event.Action, event.TargetType,
		event.TargetID, event.Details, event.IPAddress,
		event.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) List(ctx context.Context, adminUserID, action, targetType string, since time.Time, limit, offset int) ([]AdminAuditEvent, int, error) {
	if limit <= 0 {
		limit = 50
	}

	var where []string
	var args []any

	if adminUserID != "" {
		where = append(where, "admin_user_id = ?")
		args = append(args, adminUserID)
	}
	if action != "" {
		where = append(where, "action = ?")
		args = append(args, action)
	}
	if targetType != "" {
		where = append(where, "target_type = ?")
		args = append(args, targetType)
	}
	if !since.IsZero() {
		where = append(where, "created_at >= ?")
		args = append(args, since.UTC().Format(time.RFC3339))
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
		"SELECT id, admin_user_id, action, target_type, target_id, details, ip_address, created_at FROM admin_audit_log %s ORDER BY created_at DESC LIMIT ? OFFSET ?",
		whereClause,
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
		var createdAt string
		if err := rows.Scan(&e.ID, &e.AdminUserID, &e.Action, &e.TargetType, &e.TargetID, &e.Details, &e.IPAddress, &createdAt); err != nil {
			return nil, 0, err
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		events = append(events, e)
	}
	return events, total, rows.Err()
}

func (s *SQLiteStore) Close() error {
	return nil // shared db, don't close
}
