package blob

import (
	"context"
	"io"
	"time"
)

// BlobMeta holds metadata about an uploaded blob.
type BlobMeta struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	OwnerID     string    `json:"owner_id"`   // user or agent ID
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// MetaStore defines the persistence interface for blob metadata.
type MetaStore interface {
	// Create inserts a new blob metadata record.
	Create(ctx context.Context, meta *BlobMeta) error

	// Get retrieves blob metadata by ID.
	Get(ctx context.Context, id string) (*BlobMeta, error)

	// Delete removes a blob metadata record.
	Delete(ctx context.Context, id string) error

	// ListExpired returns IDs of blobs that have expired.
	ListExpired(ctx context.Context) ([]string, error)

	// SumSizeByOwner returns the total size of all blobs owned by a given owner.
	SumSizeByOwner(ctx context.Context, ownerID string) (int64, error)

	// Migrate creates the required database tables and indexes.
	Migrate(ctx context.Context) error

	// Close releases resources.
	Close() error
}

// FileStore defines the interface for blob file storage on disk.
type FileStore interface {
	// Write stores blob data and returns the number of bytes written.
	Write(id string, r io.Reader) (int64, error)

	// Open returns a reader for the blob data.
	Open(id string) (io.ReadCloser, error)

	// Remove deletes the blob file from disk.
	Remove(id string) error
}
