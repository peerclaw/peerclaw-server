package server

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter implements per-IP token bucket rate limiting.
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
	rate     rate.Limit
	burst    int
}

// NewIPRateLimiter creates a new per-IP rate limiter.
func NewIPRateLimiter(r float64, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rate:     rate.Limit(r),
		burst:    burst,
	}
}

// GetLimiter returns the rate limiter for the given IP, creating one if needed.
func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(l.rate, l.burst)
		l.limiters[ip] = &rateLimiterEntry{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	entry.lastSeen = time.Now()
	return entry.limiter
}

// Cleanup removes entries that have been inactive for more than 5 minutes.
func (l *IPRateLimiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	threshold := time.Now().Add(-5 * time.Minute)
	for ip, entry := range l.limiters {
		if entry.lastSeen.Before(threshold) {
			delete(l.limiters, ip)
		}
	}
}

// StartCleanup starts a background goroutine that periodically cleans up
// stale rate limiter entries. Returns a stop function.
func (l *IPRateLimiter) StartCleanup(interval time.Duration) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				l.Cleanup()
			case <-done:
				return
			}
		}
	}()
	return func() { close(done) }
}

// RateLimitMiddleware returns HTTP 429 when a client exceeds the rate limit.
func RateLimitMiddleware(limiter *IPRateLimiter, logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			lim := limiter.GetLimiter(ip)
			if !lim.Allow() {
				retryAfter := int(1.0 / float64(limiter.rate))
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				logger.Warn("rate limited",
					"ip", ip,
					"request_id", RequestIDFromContext(r.Context()),
				)
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	// Check X-Forwarded-For first for reverse proxy setups.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// First IP in the list is the original client.
		if i := net.ParseIP(xff); i != nil {
			return xff
		}
	}
	if xff := r.Header.Get("X-Real-IP"); xff != "" {
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
