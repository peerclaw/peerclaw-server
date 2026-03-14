package versioncheck

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Service periodically fetches the latest CLI release from GitHub and caches
// the result so that admin/provider pages can show upgrade prompts.
type Service struct {
	repo     string
	interval time.Duration
	logger   *slog.Logger
	client   *http.Client

	mu         sync.RWMutex
	latest     string // e.g. "v0.8.0"
	releaseURL string // e.g. "https://github.com/peerclaw/peerclaw-cli/releases/tag/v0.8.0"
}

// New creates a new version check service.
func New(repo string, interval time.Duration, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if interval <= 0 {
		interval = time.Hour
	}
	return &Service{
		repo:     repo,
		interval: interval,
		logger:   logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Start begins the background polling loop. It blocks until ctx is cancelled.
func (s *Service) Start(ctx context.Context) {
	// Fetch once immediately.
	s.fetch()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.fetch()
		}
	}
}

// Latest returns the cached latest version and release URL.
func (s *Service) Latest() (version, releaseURL string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latest, s.releaseURL
}

// IsOutdated returns true if current is older than the cached latest version.
// Returns false if either version cannot be parsed.
func (s *Service) IsOutdated(current string) bool {
	s.mu.RLock()
	latest := s.latest
	s.mu.RUnlock()
	if latest == "" || current == "" {
		return false
	}
	return compareSemver(current, latest) < 0
}

func (s *Service) fetch() {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", s.repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s.logger.Warn("versioncheck: failed to create request", "error", err)
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Debug("versioncheck: fetch failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Debug("versioncheck: unexpected status", "status", resp.StatusCode)
		return
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		s.logger.Warn("versioncheck: decode failed", "error", err)
		return
	}

	s.mu.Lock()
	s.latest = release.TagName
	s.releaseURL = release.HTMLURL
	s.mu.Unlock()

	s.logger.Debug("versioncheck: latest version", "version", release.TagName)
}

// compareSemver compares two semver strings (with optional "v" prefix).
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Returns 0 on parse error (safe default: not outdated).
func compareSemver(a, b string) int {
	aParts := parseSemver(a)
	bParts := parseSemver(b)
	if aParts == nil || bParts == nil {
		return 0
	}
	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}
	return 0
}

func parseSemver(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return nil
	}
	result := make([]int, 3)
	for i, p := range parts {
		// Strip pre-release suffix (e.g. "1-beta")
		if idx := strings.IndexByte(p, '-'); idx >= 0 {
			p = p[:idx]
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		result[i] = n
	}
	return result
}
