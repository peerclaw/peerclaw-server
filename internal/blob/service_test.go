package blob

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// memFileStore is an in-memory FileStore for testing (avoids filesystem).
type memFileStore struct {
	files map[string][]byte
}

func newMemFileStore() *memFileStore {
	return &memFileStore{files: make(map[string][]byte)}
}

func (m *memFileStore) Write(id string, r io.Reader) (int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	m.files[id] = data
	return int64(len(data)), nil
}

func (m *memFileStore) Open(id string) (io.ReadCloser, error) {
	data, ok := m.files[id]
	if !ok {
		return nil, io.EOF
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *memFileStore) Remove(id string) error {
	delete(m.files, id)
	return nil
}

func newTestService(t *testing.T) (*Service, *memFileStore) {
	t.Helper()
	db := newTestDB(t)
	meta := NewSQLiteMetaStore(db)
	if err := meta.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	fs := newMemFileStore()
	svc := NewService(meta, fs, ServiceConfig{
		MaxFileSize: 1024,       // 1 KB for tests
		OwnerQuota:  4096,       // 4 KB for tests
		TTL:         time.Hour,
	}, nil)
	return svc, fs
}

func TestService_UploadAndDownload(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	content := "hello world blob content"
	result, err := svc.Upload(ctx, "owner-1", "test.txt", "text/plain", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if result.Filename != "test.txt" {
		t.Errorf("Filename = %q, want %q", result.Filename, "test.txt")
	}
	if result.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", result.Size, len(content))
	}
	if result.ContentType != "text/plain" {
		t.Errorf("ContentType = %q, want %q", result.ContentType, "text/plain")
	}

	// Download the blob.
	meta, rc, err := svc.Download(ctx, result.ID)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer rc.Close()

	if meta.Filename != "test.txt" {
		t.Errorf("Download Filename = %q, want %q", meta.Filename, "test.txt")
	}

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != content {
		t.Errorf("Download content = %q, want %q", string(data), content)
	}
}

func TestService_QuotaEnforcement(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// Upload blobs that fill up most of the 4 KB quota (each under 1 KB max file size).
	chunk := strings.Repeat("x", 900)
	for i := range 4 {
		_, err := svc.Upload(ctx, "owner-quota", "chunk.bin", "application/octet-stream", strings.NewReader(chunk))
		if err != nil {
			t.Fatalf("Upload chunk %d: %v", i, err)
		}
	}
	// At this point we have used 3600 bytes of the 4096 quota.
	// Upload another chunk that would exceed quota (3600 + 900 > 4096).
	_, err := svc.Upload(ctx, "owner-quota", "overflow.bin", "application/octet-stream", strings.NewReader(chunk))
	if err == nil {
		t.Fatal("expected quota exceeded error, got nil")
	}
}

func TestService_FileSizeLimit(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// Upload a blob that exceeds max file size (1 KB).
	tooBig := strings.Repeat("z", 2000)
	_, err := svc.Upload(ctx, "owner-size", "huge.bin", "application/octet-stream", strings.NewReader(tooBig))
	if err == nil {
		t.Fatal("expected file too large error, got nil")
	}
}
