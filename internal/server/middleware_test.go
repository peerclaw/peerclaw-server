package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChain(t *testing.T) {
	var order []string
	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m1-before")
			next.ServeHTTP(w, r)
			order = append(order, "m1-after")
		})
	}
	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m2-before")
			next.ServeHTTP(w, r)
			order = append(order, "m2-after")
		})
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	chain := Chain(handler, m1, m2)
	req := httptest.NewRequest("GET", "/", nil)
	chain.ServeHTTP(httptest.NewRecorder(), req)

	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	if len(order) != len(expected) {
		t.Fatalf("order length = %d, want %d", len(order), len(expected))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	logger := slog.Default()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	chain := Chain(handler, RecoveryMiddleware(logger))
	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromContext(r.Context())
		if id == "" {
			t.Error("request ID not set in context")
		}
	})

	t.Run("generates new ID", func(t *testing.T) {
		chain := Chain(handler, RequestIDMiddleware())
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		chain.ServeHTTP(w, req)

		if w.Header().Get("X-Request-ID") == "" {
			t.Error("X-Request-ID header not set")
		}
	})

	t.Run("reuses existing ID", func(t *testing.T) {
		chain := Chain(handler, RequestIDMiddleware())
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Request-ID", "custom-id-123")
		w := httptest.NewRecorder()
		chain.ServeHTTP(w, req)

		if w.Header().Get("X-Request-ID") != "custom-id-123" {
			t.Errorf("X-Request-ID = %q, want %q", w.Header().Get("X-Request-ID"), "custom-id-123")
		}
	})
}

func TestLoggingMiddleware(t *testing.T) {
	logger := slog.Default()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	chain := Chain(handler, LoggingMiddleware(logger))
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestMaxBodyMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		_, err := r.Body.Read(buf)
		if err != nil {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	chain := Chain(handler, MaxBodyMiddleware(10))
	// Send a body larger than 10 bytes.
	body := "this is a body that exceeds the 10 byte limit"
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestStatusWriter(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

	sw.WriteHeader(http.StatusNotFound)
	if sw.status != http.StatusNotFound {
		t.Errorf("status = %d, want %d", sw.status, http.StatusNotFound)
	}

	// Second call should not change status.
	sw.WriteHeader(http.StatusOK)
	if sw.status != http.StatusNotFound {
		t.Errorf("status = %d, want %d after second WriteHeader", sw.status, http.StatusNotFound)
	}
}
