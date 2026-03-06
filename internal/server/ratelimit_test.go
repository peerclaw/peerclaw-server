package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIPRateLimiter_GetLimiter(t *testing.T) {
	rl := NewIPRateLimiter(10, 20)

	l1 := rl.GetLimiter("1.2.3.4")
	l2 := rl.GetLimiter("1.2.3.4")
	l3 := rl.GetLimiter("5.6.7.8")

	if l1 != l2 {
		t.Error("same IP should return same limiter")
	}
	if l1 == l3 {
		t.Error("different IPs should return different limiters")
	}
}

func TestIPRateLimiter_Cleanup(t *testing.T) {
	rl := NewIPRateLimiter(10, 20)
	rl.GetLimiter("1.2.3.4")

	// Simulate staleness.
	rl.mu.Lock()
	rl.limiters["1.2.3.4"].lastSeen = time.Now().Add(-10 * time.Minute)
	rl.mu.Unlock()

	rl.Cleanup()

	rl.mu.Lock()
	_, exists := rl.limiters["1.2.3.4"]
	rl.mu.Unlock()

	if exists {
		t.Error("stale entry should have been cleaned up")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	logger := slog.Default()
	rl := NewIPRateLimiter(1, 1) // 1 request/sec, burst=1

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	chain := Chain(handler, RateLimitMiddleware(rl, logger))

	// First request should succeed.
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("first request status = %d, want %d", w.Code, http.StatusOK)
	}

	// Second request should be rate limited.
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	w = httptest.NewRecorder()
	chain.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("second request status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}

	if w.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header should be set")
	}
}

func TestClientIP(t *testing.T) {
	// Without trusted proxies, X-Forwarded-For/X-Real-IP are ignored.
	rl := NewIPRateLimiter(10, 20)

	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "remote addr with port",
			remoteAddr: "1.2.3.4:5678",
			want:       "1.2.3.4",
		},
		{
			name:       "X-Real-IP header ignored without trusted proxy",
			remoteAddr: "1.2.3.4:5678",
			headers:    map[string]string{"X-Real-IP": "10.0.0.1"},
			want:       "1.2.3.4",
		},
		{
			name:       "X-Forwarded-For ignored without trusted proxy",
			remoteAddr: "1.2.3.4:5678",
			headers:    map[string]string{"X-Forwarded-For": "9.8.7.6"},
			want:       "1.2.3.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			got := clientIP(req, rl)
			if got != tt.want {
				t.Errorf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientIP_TrustedProxy(t *testing.T) {
	rl := NewIPRateLimiter(10, 20)
	rl.SetTrustedProxies([]string{"10.0.0.0/8"})

	// Request from trusted proxy — X-Forwarded-For is trusted.
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1")

	got := clientIP(req, rl)
	if got != "203.0.113.50" {
		t.Errorf("clientIP() = %q, want %q", got, "203.0.113.50")
	}

	// Request NOT from trusted proxy — X-Forwarded-For is ignored.
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")

	got = clientIP(req, rl)
	if got != "1.2.3.4" {
		t.Errorf("clientIP() = %q, want %q", got, "1.2.3.4")
	}
}
