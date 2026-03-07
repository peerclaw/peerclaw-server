package review

import (
	"context"
	"time"
)

// Review represents a user's review of an agent.
type Review struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	UserID    string    `json:"user_id"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReviewSummary holds aggregate review statistics for an agent.
type ReviewSummary struct {
	AverageRating float64 `json:"average_rating"`
	TotalReviews  int     `json:"total_reviews"`
	Distribution  [5]int  `json:"distribution"` // index 0 = 1-star, index 4 = 5-star
}

// AbuseReport represents a user-submitted abuse report.
type AbuseReport struct {
	ID         string    `json:"id"`
	ReporterID string    `json:"reporter_id"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Reason     string    `json:"reason"`
	Details    string    `json:"details"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// Category represents an agent category.
type Category struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	SortOrder   int    `json:"sort_order"`
}

// Store defines the persistence interface for review data.
type Store interface {
	// UpsertReview inserts or updates a review.
	UpsertReview(ctx context.Context, review *Review) error

	// GetReview retrieves a review by agent ID and user ID.
	GetReview(ctx context.Context, agentID, userID string) (*Review, error)

	// DeleteReview removes a review by agent ID and user ID.
	DeleteReview(ctx context.Context, agentID, userID string) error

	// ListReviews returns reviews for an agent with pagination.
	ListReviews(ctx context.Context, agentID string, limit, offset int) ([]Review, int, error)

	// GetReviewSummary returns aggregate review statistics for an agent.
	GetReviewSummary(ctx context.Context, agentID string) (*ReviewSummary, error)

	// CreateReport inserts a new abuse report.
	CreateReport(ctx context.Context, report *AbuseReport) error

	// ListCategories returns all categories ordered by sort_order.
	ListCategories(ctx context.Context) ([]Category, error)

	// GetCategoriesByAgent returns the categories associated with an agent.
	GetCategoriesByAgent(ctx context.Context, agentID string) ([]Category, error)

	// SetAgentCategories replaces the category associations for an agent.
	SetAgentCategories(ctx context.Context, agentID string, categoryIDs []string) error

	// ListReports returns abuse reports with optional status filter and pagination.
	ListReports(ctx context.Context, status string, limit, offset int) ([]AbuseReport, int, error)

	// GetReport retrieves a single abuse report by ID.
	GetReport(ctx context.Context, id string) (*AbuseReport, error)

	// UpdateReportStatus updates the status of an abuse report.
	UpdateReportStatus(ctx context.Context, id, status string) error

	// DeleteReport removes an abuse report by ID.
	DeleteReport(ctx context.Context, id string) error

	// CreateCategory inserts a new category.
	CreateCategory(ctx context.Context, category *Category) error

	// UpdateCategory updates an existing category.
	UpdateCategory(ctx context.Context, category *Category) error

	// DeleteCategory removes a category by ID.
	DeleteCategory(ctx context.Context, id string) error

	// CountReviews returns the total number of reviews.
	CountReviews(ctx context.Context) (int, error)

	// CountReports returns the number of abuse reports, optionally filtered by status.
	CountReports(ctx context.Context, status string) (int, error)

	// Migrate creates the required tables.
	Migrate(ctx context.Context) error

	// Close releases resources.
	Close() error
}
