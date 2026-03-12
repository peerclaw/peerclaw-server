package review

import (
	"context"
	"database/sql"
	"fmt"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed review store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Migrate creates the required tables.
func (s *PostgresStore) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS reviews (
			id         TEXT PRIMARY KEY,
			agent_id   TEXT NOT NULL,
			user_id    TEXT NOT NULL,
			rating     INTEGER NOT NULL CHECK(rating >= 1 AND rating <= 5),
			comment    TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			UNIQUE(agent_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_agent ON reviews(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_user ON reviews(user_id)`,
		`CREATE TABLE IF NOT EXISTS abuse_reports (
			id          TEXT PRIMARY KEY,
			reporter_id TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id   TEXT NOT NULL,
			reason      TEXT NOT NULL,
			details     TEXT NOT NULL DEFAULT '',
			status      TEXT NOT NULL DEFAULT 'pending',
			created_at  TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_abuse_reports_target ON abuse_reports(target_type, target_id)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL UNIQUE,
			slug        TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			icon        TEXT NOT NULL DEFAULT '',
			sort_order  INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS agent_categories (
			agent_id    TEXT NOT NULL,
			category_id TEXT NOT NULL,
			PRIMARY KEY(agent_id, category_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_categories_agent ON agent_categories(agent_id)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("review migrate: %w", err)
		}
	}
	return nil
}

func (s *PostgresStore) UpsertReview(ctx context.Context, review *Review) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO reviews (id, agent_id, user_id, rating, comment, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT(agent_id, user_id) DO UPDATE SET
		   id = EXCLUDED.id,
		   rating = EXCLUDED.rating,
		   comment = EXCLUDED.comment,
		   updated_at = EXCLUDED.updated_at`,
		review.ID, review.AgentID, review.UserID, review.Rating, review.Comment,
		review.CreatedAt.UTC(), review.UpdatedAt.UTC(),
	)
	return err
}

func (s *PostgresStore) GetReview(ctx context.Context, agentID, userID string) (*Review, error) {
	var r Review
	err := s.db.QueryRowContext(ctx,
		`SELECT id, agent_id, user_id, rating, comment, created_at, updated_at
		 FROM reviews WHERE agent_id = $1 AND user_id = $2`, agentID, userID,
	).Scan(&r.ID, &r.AgentID, &r.UserID, &r.Rating, &r.Comment, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("review not found")
		}
		return nil, err
	}
	return &r, nil
}

func (s *PostgresStore) DeleteReview(ctx context.Context, agentID, userID string) error {
	res, err := s.db.ExecContext(ctx,
		"DELETE FROM reviews WHERE agent_id = $1 AND user_id = $2", agentID, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("review not found")
	}
	return nil
}

func (s *PostgresStore) ListReviews(ctx context.Context, agentID string, limit, offset int) ([]Review, int, error) {
	var total int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM reviews WHERE agent_id = $1", agentID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_id, user_id, rating, comment, created_at, updated_at
		 FROM reviews WHERE agent_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		agentID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		if err := rows.Scan(&r.ID, &r.AgentID, &r.UserID, &r.Rating, &r.Comment, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, 0, err
		}
		reviews = append(reviews, r)
	}
	return reviews, total, rows.Err()
}

func (s *PostgresStore) GetReviewSummary(ctx context.Context, agentID string) (*ReviewSummary, error) {
	var summary ReviewSummary
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(AVG(rating::DOUBLE PRECISION), 0), COUNT(*)
		 FROM reviews WHERE agent_id = $1`, agentID,
	).Scan(&summary.AverageRating, &summary.TotalReviews)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT rating, COUNT(*) FROM reviews WHERE agent_id = $1 GROUP BY rating`,
		agentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var rating, count int
		if err := rows.Scan(&rating, &count); err != nil {
			return nil, err
		}
		if rating >= 1 && rating <= 5 {
			summary.Distribution[rating-1] = count
		}
	}
	return &summary, rows.Err()
}

func (s *PostgresStore) CreateReport(ctx context.Context, report *AbuseReport) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO abuse_reports (id, reporter_id, target_type, target_id, reason, details, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		report.ID, report.ReporterID, report.TargetType, report.TargetID,
		report.Reason, report.Details, report.Status,
		report.CreatedAt.UTC(),
	)
	return err
}

func (s *PostgresStore) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, slug, description, icon, sort_order
		 FROM categories ORDER BY sort_order ASC, name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.Icon, &c.SortOrder); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (s *PostgresStore) GetCategoriesByAgent(ctx context.Context, agentID string) ([]Category, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.name, c.slug, c.description, c.icon, c.sort_order
		 FROM categories c
		 JOIN agent_categories ac ON ac.category_id = c.id
		 WHERE ac.agent_id = $1
		 ORDER BY c.sort_order ASC, c.name ASC`, agentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.Icon, &c.SortOrder); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (s *PostgresStore) SetAgentCategories(ctx context.Context, agentID string, categoryIDs []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "DELETE FROM agent_categories WHERE agent_id = $1", agentID); err != nil {
		return err
	}

	for _, catID := range categoryIDs {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO agent_categories (agent_id, category_id) VALUES ($1, $2)",
			agentID, catID,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) ListReports(ctx context.Context, status string, limit, offset int) ([]AbuseReport, int, error) {
	if limit <= 0 {
		limit = 50
	}

	where := "1=1"
	var args []interface{}
	argN := 1
	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, status)
		argN++
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM abuse_reports WHERE "+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf("SELECT id, reporter_id, target_type, target_id, reason, details, status, created_at FROM abuse_reports WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", where, argN, argN+1),
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []AbuseReport
	for rows.Next() {
		var r AbuseReport
		if err := rows.Scan(&r.ID, &r.ReporterID, &r.TargetType, &r.TargetID, &r.Reason, &r.Details, &r.Status, &r.CreatedAt); err != nil {
			return nil, 0, err
		}
		reports = append(reports, r)
	}
	return reports, total, rows.Err()
}

func (s *PostgresStore) GetReport(ctx context.Context, id string) (*AbuseReport, error) {
	var r AbuseReport
	err := s.db.QueryRowContext(ctx,
		`SELECT id, reporter_id, target_type, target_id, reason, details, status, created_at
		 FROM abuse_reports WHERE id = $1`, id,
	).Scan(&r.ID, &r.ReporterID, &r.TargetType, &r.TargetID, &r.Reason, &r.Details, &r.Status, &r.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("report not found")
		}
		return nil, err
	}
	return &r, nil
}

func (s *PostgresStore) UpdateReportStatus(ctx context.Context, id, status string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE abuse_reports SET status = $1 WHERE id = $2", status, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("report not found")
	}
	return nil
}

func (s *PostgresStore) DeleteReport(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM abuse_reports WHERE id = $1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("report not found")
	}
	return nil
}

func (s *PostgresStore) CreateCategory(ctx context.Context, category *Category) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO categories (id, name, slug, description, icon, sort_order)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		category.ID, category.Name, category.Slug, category.Description, category.Icon, category.SortOrder,
	)
	return err
}

func (s *PostgresStore) UpdateCategory(ctx context.Context, category *Category) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE categories SET name = $1, slug = $2, description = $3, icon = $4, sort_order = $5 WHERE id = $6`,
		category.Name, category.Slug, category.Description, category.Icon, category.SortOrder, category.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("category not found")
	}
	return nil
}

func (s *PostgresStore) DeleteCategory(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM categories WHERE id = $1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("category not found")
	}
	return nil
}

func (s *PostgresStore) CountReviews(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM reviews").Scan(&count)
	return count, err
}

func (s *PostgresStore) CountReports(ctx context.Context, status string) (int, error) {
	var count int
	if status != "" {
		err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM abuse_reports WHERE status = $1", status).Scan(&count)
		return count, err
	}
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM abuse_reports").Scan(&count)
	return count, err
}

// Close is a no-op since the db is shared.
func (s *PostgresStore) Close() error {
	return nil
}
