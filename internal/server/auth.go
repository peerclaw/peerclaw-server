package server

import (
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/peerclaw/peerclaw-server/internal/identity"
	"github.com/peerclaw/peerclaw-server/internal/userauth"
)

// AuthConfig controls authentication behavior.
type AuthConfig struct {
	// Required when true rejects unauthenticated requests; when false only logs warnings.
	Required bool
	// Verifier validates signatures and API keys.
	Verifier *identity.Verifier
}

// AuthMiddleware validates requests using bearer token or Ed25519 signature.
// Public routes should be registered outside the middleware chain.
func AuthMiddleware(cfg AuthConfig, logger *slog.Logger) Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			agentID, ok := authenticate(r, cfg, logger)
			if !ok {
				if cfg.Required {
					writeJSONError(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				// Transition mode: log warning but allow through.
				logger.Warn("unauthenticated request allowed (auth.required=false)",
					"method", r.Method,
					"path", r.URL.Path,
					"remote", r.RemoteAddr,
				)
				next.ServeHTTP(w, r)
				return
			}

			ctx := identity.WithAgentID(r.Context(), agentID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// authenticate tries bearer token then signature-based auth.
// Returns (agentID, true) on success, ("", false) on failure.
func authenticate(r *http.Request, cfg AuthConfig, logger *slog.Logger) (string, bool) {
	// Method 1: Bearer token (API key) — requires X-PeerClaw-Agent-ID header.
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		token, err := identity.ExtractBearerToken(authHeader)
		if err != nil {
			logger.Debug("invalid authorization header", "error", err)
			return "", false
		}
		agentID := r.Header.Get("X-PeerClaw-Agent-ID")
		if agentID == "" {
			logger.Debug("bearer token present but missing X-PeerClaw-Agent-ID")
			return "", false
		}
		if cfg.Verifier != nil {
			if err := cfg.Verifier.VerifyAPIKey(agentID, token); err != nil {
				logger.Debug("API key verification failed", "agent_id", agentID, "error", err)
				return "", false
			}
		}
		return agentID, true
	}

	// Method 2: Ed25519 signature — X-PeerClaw-Signature + X-PeerClaw-PublicKey headers.
	sig := r.Header.Get("X-PeerClaw-Signature")
	pubKeyStr := r.Header.Get("X-PeerClaw-PublicKey")
	if sig != "" && pubKeyStr != "" {
		// Read body for verification, then restore it.
		var body []byte
		if r.Body != nil {
			var err error
			body, err = io.ReadAll(r.Body)
			if err != nil {
				logger.Debug("failed to read body for signature verification", "error", err)
				return "", false
			}
			r.Body = io.NopCloser(newBytesReader(body))
		}

		if cfg.Verifier != nil {
			if err := cfg.Verifier.VerifySignature(pubKeyStr, body, sig); err != nil {
				logger.Debug("signature verification failed", "error", err)
				return "", false
			}
		}

		// Derive agent ID from public key (use the public key string as agent identifier).
		agentID := r.Header.Get("X-PeerClaw-Agent-ID")
		if agentID == "" {
			// Fall back to using public key as agent ID.
			agentID = pubKeyStr
		}
		return agentID, true
	}

	return "", false
}

// OwnerOnlyMiddleware ensures the authenticated agent ID matches the {id} path parameter.
func OwnerOnlyMiddleware(logger *slog.Logger) Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pathID := r.PathValue("id")
			if pathID == "" {
				next.ServeHTTP(w, r)
				return
			}

			ctxAgentID, ok := identity.AgentIDFromContext(r.Context())
			if !ok {
				// No auth context — let auth middleware handle rejection.
				next.ServeHTTP(w, r)
				return
			}

			if ctxAgentID != pathID {
				logger.Warn("owner-only access denied",
					"authenticated_agent", ctxAgentID,
					"target_agent", pathID,
				)
				writeJSONError(w, "forbidden: not the owner", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// bytesReaderWrapper is a simple bytes.Reader wrapper.
type bytesReaderWrapper struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReaderWrapper {
	return &bytesReaderWrapper{data: data}
}

func (br *bytesReaderWrapper) Read(p []byte) (int, error) {
	if br.pos >= len(br.data) {
		return 0, io.EOF
	}
	n := copy(p, br.data[br.pos:])
	br.pos += n
	return n, nil
}

// UserAuthMiddleware validates JWT tokens from Authorization: Bearer <jwt> headers
// and stores the user ID and role in the request context.
func UserAuthMiddleware(jwtMgr *userauth.JWTManager, logger *slog.Logger) Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, "authorization header required", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeJSONError(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}

			claims, err := jwtMgr.ValidateAccessToken(parts[1])
			if err != nil {
				logger.Debug("JWT validation failed", "error", err)
				writeJSONError(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := identity.WithUserID(r.Context(), claims.UserID)
			ctx = identity.WithUserRole(ctx, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminOnlyMiddleware ensures the authenticated user has the "admin" role.
// Must be used after UserAuthMiddleware which stores the role in context.
func AdminOnlyMiddleware(logger *slog.Logger) Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := identity.UserRoleFromContext(r.Context())
			if !ok || role != "admin" {
				logger.Warn("admin-only access denied",
					"role", role,
					"path", r.URL.Path,
				)
				writeJSONError(w, "forbidden: admin access required", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// bridgeAuthMiddleware performs optional dual authentication for bridge endpoints.
// If agent auth headers or a JWT token are present and valid, the caller's identity
// is stored in the request context. If no credentials are provided, the request
// is passed through — the handler uses PlaygroundEnabled as the fallback gate.
func (s *HTTPServer) bridgeAuthMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try agent auth (bearer token or Ed25519 signature).
			agentID, ok := authenticate(r, s.authCfg, s.logger)
			if ok {
				ctx := identity.WithAgentID(r.Context(), agentID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Try user auth (JWT) if available.
			if s.userAuth != nil {
				if authHeader := r.Header.Get("Authorization"); authHeader != "" {
					parts := strings.SplitN(authHeader, " ", 2)
					if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
						claims, err := s.userAuth.JWTManager().ValidateAccessToken(parts[1])
						if err == nil {
							ctx := identity.WithUserID(r.Context(), claims.UserID)
							ctx = identity.WithUserRole(ctx, claims.Role)
							next.ServeHTTP(w, r.WithContext(ctx))
							return
						}
					}
				}
			}

			// No valid auth — pass through for handler-level PlaygroundEnabled check.
			next.ServeHTTP(w, r)
		})
	}
}

// bridgeAccessAllowed returns true if the caller is authenticated (agent or user)
// or the target agent has PlaygroundEnabled. Use this in bridge handlers instead
// of checking only PlaygroundEnabled.
func bridgeAccessAllowed(r *http.Request, playgroundEnabled bool) bool {
	if playgroundEnabled {
		return true
	}
	if _, ok := identity.AgentIDFromContext(r.Context()); ok {
		return true
	}
	if _, ok := identity.UserIDFromContext(r.Context()); ok {
		return true
	}
	return false
}

// OptionalUserAuthMiddleware extracts JWT if present but does not require it.
func OptionalUserAuthMiddleware(jwtMgr *userauth.JWTManager, logger *slog.Logger) Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := jwtMgr.ValidateAccessToken(parts[1])
			if err != nil {
				logger.Debug("optional JWT validation failed", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			ctx := identity.WithUserID(r.Context(), claims.UserID)
			ctx = identity.WithUserRole(ctx, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
