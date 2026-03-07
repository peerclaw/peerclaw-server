package server

import (
	"encoding/json"
	"net/http"

	"github.com/peerclaw/peerclaw-server/internal/identity"
	"github.com/peerclaw/peerclaw-server/internal/userauth"
)

// handleAuthRegister handles POST /api/v1/auth/register.
func (s *HTTPServer) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	var req userauth.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, tokens, err := s.userAuth.Register(r.Context(), req)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]any{
		"user":   sanitizeUser(user),
		"tokens": tokens,
	})
}

// handleAuthLogin handles POST /api/v1/auth/login.
func (s *HTTPServer) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	var req userauth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ipAddress := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ipAddress = fwd
	}
	userAgent := r.Header.Get("User-Agent")

	user, tokens, err := s.userAuth.Login(r.Context(), req, ipAddress, userAgent)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"user":   sanitizeUser(user),
		"tokens": tokens,
	})
}

// handleAuthRefresh handles POST /api/v1/auth/refresh.
func (s *HTTPServer) handleAuthRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tokens, err := s.userAuth.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	s.jsonResponse(w, http.StatusOK, tokens)
}

// handleAuthLogout handles POST /api/v1/auth/logout.
func (s *HTTPServer) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	_ = s.userAuth.Logout(r.Context(), req.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}

// handleAuthMe handles GET /api/v1/auth/me.
func (s *HTTPServer) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.userAuth.GetUser(r.Context(), userID)
	if err != nil {
		s.jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	s.jsonResponse(w, http.StatusOK, sanitizeUser(user))
}

// handleAuthUpdateMe handles PUT /api/v1/auth/me.
func (s *HTTPServer) handleAuthUpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := s.userAuth.UpdateProfile(r.Context(), userID, req.DisplayName)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusOK, sanitizeUser(user))
}

// handleAuthCreateAPIKey handles POST /api/v1/auth/api-keys.
func (s *HTTPServer) handleAuthCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	apiKey, rawKey, err := s.userAuth.GenerateAPIKey(r.Context(), userID, req.Name)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]any{
		"api_key": apiKey,
		"key":     rawKey, // Only returned once at creation time
	})
}

// handleAuthListAPIKeys handles GET /api/v1/auth/api-keys.
func (s *HTTPServer) handleAuthListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	keys, err := s.userAuth.ListAPIKeys(r.Context(), userID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if keys == nil {
		keys = []userauth.UserAPIKey{}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"api_keys": keys})
}

// handleAuthRevokeAPIKey handles DELETE /api/v1/auth/api-keys/{key_id}.
func (s *HTTPServer) handleAuthRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	keyID := r.PathValue("key_id")
	if keyID == "" {
		s.jsonError(w, "key_id is required", http.StatusBadRequest)
		return
	}

	if err := s.userAuth.RevokeAPIKey(r.Context(), keyID, userID); err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// sanitizeUser returns a user without sensitive fields.
func sanitizeUser(u *userauth.User) map[string]any {
	return map[string]any{
		"id":           u.ID,
		"email":        u.Email,
		"display_name": u.DisplayName,
		"role":         u.Role,
		"created_at":   u.CreatedAt,
		"updated_at":   u.UpdatedAt,
	}
}
