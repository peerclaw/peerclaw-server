package server

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
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
	mu             sync.Mutex
	limiters       map[string]*rateLimiterEntry
	rate           rate.Limit
	burst          int
	trustedProxies []*net.IPNet
}

// NewIPRateLimiter creates a new per-IP rate limiter.
func NewIPRateLimiter(r float64, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rate:     rate.Limit(r),
		burst:    burst,
	}
}

// SetTrustedProxies sets CIDR ranges whose X-Forwarded-For headers are trusted.
func (l *IPRateLimiter) SetTrustedProxies(cidrs []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.trustedProxies = nil
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			l.trustedProxies = append(l.trustedProxies, ipNet)
		}
	}
}

// isTrustedProxy checks if the given IP is in the trusted proxy list.
func (l *IPRateLimiter) isTrustedProxy(ip string) bool {
	l.mu.Lock()
	proxies := l.trustedProxies
	l.mu.Unlock()

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, ipNet := range proxies {
		if ipNet.Contains(parsed) {
			return true
		}
	}
	return false
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
			ip := clientIP(r, limiter)
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

func clientIP(r *http.Request, limiter *IPRateLimiter) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}

	// Only trust proxy headers if RemoteAddr is from a trusted proxy.
	if limiter != nil && limiter.isTrustedProxy(remoteHost) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// First IP in the comma-separated list is the original client.
			parts := strings.SplitN(xff, ",", 2)
			ip := strings.TrimSpace(parts[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			ip := strings.TrimSpace(xri)
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	return remoteHost
}
