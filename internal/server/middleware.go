package server

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-server/internal/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Middleware is a standard HTTP middleware function.
type Middleware func(http.Handler) http.Handler

// Chain wraps a handler with the given middlewares.
// Middlewares are applied in reverse order so the first middleware in the list
// is the outermost wrapper (executed first on request, last on response).
func Chain(handler http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return handler
}

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestIDFromContext extracts the request ID from a context.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// RecoveryMiddleware recovers from panics in downstream handlers,
// logs the stack trace, and returns 500 Internal Server Error.
func RecoveryMiddleware(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()
					logger.Error("panic recovered",
						"error", fmt.Sprintf("%v", rec),
						"stack", string(stack),
						"method", r.Method,
						"path", r.URL.Path,
						"request_id", RequestIDFromContext(r.Context()),
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDMiddleware injects a unique request ID into the request context
// and sets the X-Request-ID response header. If the request already carries
// an X-Request-ID header, that value is reused.
func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				id = uuid.New().String()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, id)
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LoggingMiddleware logs each HTTP request with method, path, status code,
// duration, and request ID.
func LoggingMiddleware(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", RequestIDFromContext(r.Context()),
			)
		})
	}
}

// MaxBodyMiddleware limits the size of request bodies.
func MaxBodyMiddleware(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// TracingMiddleware creates a span for each HTTP request with method and status attributes.
func TracingMiddleware(tracer trace.Tracer) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
			defer span.End()

			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r.WithContext(ctx))

			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.Path),
				attribute.Int("http.status_code", sw.status),
			)
		})
	}
}

// MetricsMiddleware records HTTP request count, duration, and active requests.
func MetricsMiddleware(metrics *observability.Metrics) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			metrics.HTTPActiveRequests.Add(r.Context(), 1)

			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			duration := time.Since(start).Seconds()
			attrs := metric.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.status_code", strconv.Itoa(sw.status)),
			)
			metrics.HTTPActiveRequests.Add(r.Context(), -1)
			metrics.HTTPRequestsTotal.Add(r.Context(), 1, attrs)
			metrics.HTTPRequestDuration.Record(r.Context(), duration, attrs)
		})
	}
}

// CORSMiddleware sets CORS headers and handles OPTIONS preflight requests.
// If allowedOrigins is empty, no origins are allowed. If allowedOrigins
// contains "*", all origins are allowed. Otherwise the request Origin must
// appear in allowedOrigins to receive CORS headers.
func CORSMiddleware(allowedOrigins []string) Middleware {
	allowAll := false
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
		}
		originSet[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Determine whether this origin is allowed.
			allowed := false
			if allowAll {
				allowed = true
			} else if len(originSet) > 0 {
				_, allowed = originSet[origin]
			}

			if !allowed {
				next.ServeHTTP(w, r)
				return
			}

			// Set the allowed origin (mirror the exact origin rather than "*"
			// so credentials can work).
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-PeerClaw-Signature, X-PeerClaw-PublicKey, X-PeerClaw-Agent-ID")

			// Handle preflight.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (sw *statusWriter) WriteHeader(code int) {
	if !sw.wroteHeader {
		sw.status = code
		sw.wroteHeader = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	if !sw.wroteHeader {
		sw.wroteHeader = true
	}
	return sw.ResponseWriter.Write(b)
}

// GzipMiddleware compresses responses when the client accepts gzip encoding.
// It skips compression for SSE streams (text/event-stream) and small responses.
func GzipMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			gz := &gzipResponseWriter{
				ResponseWriter: w,
				Writer:         nil, // lazy init
			}
			defer func() {
				if gz.Writer != nil {
					_ = gz.Writer.Close()
				}
			}()

			next.ServeHTTP(gz, r)
		})
	}
}

// gzipResponseWriter wraps http.ResponseWriter with gzip compression.
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer      *gzip.Writer
	wroteHeader bool
}

func (gz *gzipResponseWriter) WriteHeader(code int) {
	if gz.wroteHeader {
		return
	}
	gz.wroteHeader = true

	ct := gz.ResponseWriter.Header().Get("Content-Type")
	// Don't compress SSE streams — they need to be flushed immediately.
	if strings.HasPrefix(ct, "text/event-stream") {
		gz.ResponseWriter.WriteHeader(code)
		return
	}

	gz.ResponseWriter.Header().Set("Content-Encoding", "gzip")
	gz.ResponseWriter.Header().Set("Vary", "Accept-Encoding")
	gz.ResponseWriter.Header().Del("Content-Length") // length changes with compression
	gz.Writer = gzip.NewWriter(gz.ResponseWriter)
	gz.ResponseWriter.WriteHeader(code)
}

func (gz *gzipResponseWriter) Write(b []byte) (int, error) {
	if !gz.wroteHeader {
		gz.WriteHeader(http.StatusOK)
	}
	if gz.Writer != nil {
		return gz.Writer.Write(b)
	}
	return gz.ResponseWriter.Write(b)
}

func (gz *gzipResponseWriter) Flush() {
	if gz.Writer != nil {
		_ = gz.Writer.Flush()
	}
	if f, ok := gz.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap supports http.ResponseController.
func (gz *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return gz.ResponseWriter
}

// Ensure gzipResponseWriter does not implement io.ReaderFrom
// to prevent net/http from bypassing the gzip writer.
var _ io.Writer = (*gzipResponseWriter)(nil)
