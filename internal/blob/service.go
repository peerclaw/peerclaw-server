package blob

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultMaxFileSize is the maximum size for a single blob (100 MB).
	DefaultMaxFileSize int64 = 100 << 20

	// DefaultOwnerQuota is the maximum total blob storage per owner (1 GB).
	DefaultOwnerQuota int64 = 1 << 30

	// DefaultTTL is the default expiration time for blobs (24 hours).
	DefaultTTL = 24 * time.Hour
)

// ServiceConfig holds configuration for the blob service.
type ServiceConfig struct {
	MaxFileSize int64         // per-file limit in bytes (default 100MB)
	OwnerQuota  int64         // per-owner total limit in bytes (default 1GB)
	TTL         time.Duration // blob expiration time (default 24h)
}

// Service implements blob upload/download/cleanup logic.
type Service struct {
	meta   MetaStore
	files  FileStore
	cfg    ServiceConfig
	logger *slog.Logger
}

// NewService creates a new blob service.
func NewService(meta MetaStore, files FileStore, cfg ServiceConfig, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.MaxFileSize <= 0 {
		cfg.MaxFileSize = DefaultMaxFileSize
	}
	if cfg.OwnerQuota <= 0 {
		cfg.OwnerQuota = DefaultOwnerQuota
	}
	if cfg.TTL <= 0 {
		cfg.TTL = DefaultTTL
	}
	return &Service{meta: meta, files: files, cfg: cfg, logger: logger}
}

// NewMetaStore creates a MetaStore based on the database driver and a shared *sql.DB.
func NewMetaStore(driver string, db *sql.DB) MetaStore {
	switch driver {
	case "postgres":
		return NewPostgresMetaStore(db)
	default:
		return NewSQLiteMetaStore(db)
	}
}

// UploadResult contains information about a successfully uploaded blob.
type UploadResult struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	ExpiresAt   time.Time `json:"expires_at"`
	DownloadURL string    `json:"download_url"`
}

// Upload stores a blob file and its metadata.
func (s *Service) Upload(ctx context.Context, ownerID, filename, contentType string, r io.Reader) (*UploadResult, error) {
	// Check owner quota.
	used, err := s.meta.SumSizeByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("check quota: %w", err)
	}
	if used >= s.cfg.OwnerQuota {
		return nil, fmt.Errorf("storage quota exceeded (used %d bytes, limit %d bytes)", used, s.cfg.OwnerQuota)
	}

	id := uuid.New().String()

	// Limit the reader to max file size.
	limited := io.LimitReader(r, s.cfg.MaxFileSize+1)

	// Write to file store.
	n, err := s.files.Write(id, limited)
	if err != nil {
		return nil, fmt.Errorf("write blob: %w", err)
	}

	if n > s.cfg.MaxFileSize {
		// File too large — clean up.
		_ = s.files.Remove(id)
		return nil, fmt.Errorf("file too large (max %d bytes)", s.cfg.MaxFileSize)
	}

	// Check quota again after write (race-safe).
	if used+n > s.cfg.OwnerQuota {
		_ = s.files.Remove(id)
		return nil, fmt.Errorf("storage quota exceeded")
	}

	now := time.Now().UTC()
	meta := &BlobMeta{
		ID:          id,
		Filename:    filename,
		ContentType: contentType,
		Size:        n,
		OwnerID:     ownerID,
		CreatedAt:   now,
		ExpiresAt:   now.Add(s.cfg.TTL),
	}

	if err := s.meta.Create(ctx, meta); err != nil {
		_ = s.files.Remove(id)
		return nil, fmt.Errorf("save metadata: %w", err)
	}

	s.logger.Info("blob uploaded",
		"id", id, "owner", ownerID, "size", n, "filename", filename)

	return &UploadResult{
		ID:          id,
		Filename:    filename,
		ContentType: contentType,
		Size:        n,
		ExpiresAt:   meta.ExpiresAt,
	}, nil
}

// Download returns the blob metadata and a reader for its content.
func (s *Service) Download(ctx context.Context, id string) (*BlobMeta, io.ReadCloser, error) {
	meta, err := s.meta.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	// Check expiration.
	if time.Now().UTC().After(meta.ExpiresAt) {
		return nil, nil, fmt.Errorf("blob expired")
	}

	rc, err := s.files.Open(id)
	if err != nil {
		return nil, nil, err
	}

	return meta, rc, nil
}

// CleanupExpired removes expired blobs from both file and metadata stores.
func (s *Service) CleanupExpired(ctx context.Context) (int64, error) {
	ids, err := s.meta.ListExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("list expired: %w", err)
	}

	var cleaned int64
	for _, id := range ids {
		if err := s.files.Remove(id); err != nil {
			s.logger.Warn("failed to remove blob file", "id", id, "error", err)
		}
		if err := s.meta.Delete(ctx, id); err != nil {
			s.logger.Warn("failed to delete blob metadata", "id", id, "error", err)
			continue
		}
		cleaned++
	}

	if cleaned > 0 {
		s.logger.Info("expired blobs cleaned up", "count", cleaned)
	}
	return cleaned, nil
}

// MaxFileSize returns the configured max file size.
func (s *Service) MaxFileSize() int64 {
	return s.cfg.MaxFileSize
}
