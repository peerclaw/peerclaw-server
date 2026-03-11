package blob

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DiskFileStore implements FileStore using the local filesystem.
type DiskFileStore struct {
	basePath string
}

// NewDiskFileStore creates a new filesystem-based blob store.
// It creates the base directory if it does not exist.
func NewDiskFileStore(basePath string) (*DiskFileStore, error) {
	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, fmt.Errorf("create blob directory: %w", err)
	}
	return &DiskFileStore{basePath: basePath}, nil
}

func (fs *DiskFileStore) Write(id string, r io.Reader) (int64, error) {
	path := fs.path(id)

	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("create blob file: %w", err)
	}
	defer func() { _ = f.Close() }()

	n, err := io.Copy(f, r)
	if err != nil {
		// Clean up partial file on error.
		_ = os.Remove(path)
		return 0, fmt.Errorf("write blob file: %w", err)
	}

	return n, nil
}

func (fs *DiskFileStore) Open(id string) (io.ReadCloser, error) {
	f, err := os.Open(fs.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("blob file not found")
		}
		return nil, fmt.Errorf("open blob file: %w", err)
	}
	return f, nil
}

func (fs *DiskFileStore) Remove(id string) error {
	err := os.Remove(fs.path(id))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove blob file: %w", err)
	}
	return nil
}

func (fs *DiskFileStore) path(id string) string {
	// Clean the path and verify it stays within basePath to prevent traversal.
	joined := filepath.Join(fs.basePath, id)
	cleaned := filepath.Clean(joined)
	// Ensure the resolved path is a direct child of basePath.
	if !strings.HasPrefix(cleaned, filepath.Clean(fs.basePath)+string(filepath.Separator)) {
		// Return a safe path that will simply not exist.
		return filepath.Join(fs.basePath, "invalid-blob-id")
	}
	return cleaned
}
