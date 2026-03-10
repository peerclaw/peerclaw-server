package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/peerclaw/peerclaw-server/internal/identity"
	"github.com/peerclaw/peerclaw-server/internal/useracl"
)

// --- User-facing endpoints ---

type submitAccessRequestBody struct {
	Message string `json:"message"`
}

// handleSubmitAccessRequest handles POST /api/v1/agents/{id}/access-requests.
func (s *HTTPServer) handleSubmitAccessRequest(w http.ResponseWriter, r *http.Request) {
	if s.useracl == nil {
		s.jsonError(w, "access control not enabled", http.StatusNotImplemented)
		return
	}

	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	agentID := r.PathValue("id")
	var body submitAccessRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	svc, ok := s.useracl.(*useracl.Service)
	if !ok {
		s.jsonError(w, "access control not available", http.StatusNotImplemented)
		return
	}

	req, err := svc.SubmitRequest(r.Context(), agentID, userID, body.Message)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusCreated, req)
}

// handleGetAccessRequestStatus handles GET /api/v1/agents/{id}/access-requests/me.
func (s *HTTPServer) handleGetAccessRequestStatus(w http.ResponseWriter, r *http.Request) {
	if s.useracl == nil {
		s.jsonError(w, "access control not enabled", http.StatusNotImplemented)
		return
	}

	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	agentID := r.PathValue("id")

	svc, ok := s.useracl.(*useracl.Service)
	if !ok {
		s.jsonError(w, "access control not available", http.StatusNotImplemented)
		return
	}

	req, err := svc.GetByAgentAndUser(r.Context(), agentID, userID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if req == nil {
		s.jsonResponse(w, http.StatusOK, map[string]any{"status": "none"})
		return
	}

	s.jsonResponse(w, http.StatusOK, req)
}

// handleListMyAccessRequests handles GET /api/v1/user/access-requests.
func (s *HTTPServer) handleListMyAccessRequests(w http.ResponseWriter, r *http.Request) {
	if s.useracl == nil {
		s.jsonError(w, "access control not enabled", http.StatusNotImplemented)
		return
	}

	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	svc, ok := s.useracl.(*useracl.Service)
	if !ok {
		s.jsonError(w, "access control not available", http.StatusNotImplemented)
		return
	}

	requests, err := svc.ListByUser(r.Context(), userID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if requests == nil {
		requests = []useracl.AccessRequest{}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"requests": requests})
}

// --- Provider-facing endpoints ---

// handleProviderListAccessRequests handles GET /api/v1/provider/agents/{id}/access-requests.
func (s *HTTPServer) handleProviderListAccessRequests(w http.ResponseWriter, r *http.Request) {
	if s.useracl == nil {
		s.jsonError(w, "access control not enabled", http.StatusNotImplemented)
		return
	}

	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	agentID := r.PathValue("id")
	if err := s.verifyAgentOwnership(r, agentID, userID); err != nil {
		s.jsonError(w, err.Error(), http.StatusForbidden)
		return
	}

	svc, ok := s.useracl.(*useracl.Service)
	if !ok {
		s.jsonError(w, "access control not available", http.StatusNotImplemented)
		return
	}

	status := r.URL.Query().Get("status")
	requests, err := svc.ListByAgent(r.Context(), agentID, status)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if requests == nil {
		requests = []useracl.AccessRequest{}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"requests": requests})
}

type updateAccessRequestBody struct {
	Action       string  `json:"action"` // "approve" or "reject"
	RejectReason string  `json:"reject_reason,omitempty"`
	ExpiresAt    *string `json:"expires_at,omitempty"`
}

// handleProviderUpdateAccessRequest handles PUT /api/v1/provider/agents/{id}/access-requests/{request_id}.
func (s *HTTPServer) handleProviderUpdateAccessRequest(w http.ResponseWriter, r *http.Request) {
	if s.useracl == nil {
		s.jsonError(w, "access control not enabled", http.StatusNotImplemented)
		return
	}

	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	agentID := r.PathValue("id")
	if err := s.verifyAgentOwnership(r, agentID, userID); err != nil {
		s.jsonError(w, err.Error(), http.StatusForbidden)
		return
	}

	svc, ok := s.useracl.(*useracl.Service)
	if !ok {
		s.jsonError(w, "access control not available", http.StatusNotImplemented)
		return
	}

	requestID := r.PathValue("request_id")
	var body updateAccessRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	switch body.Action {
	case "approve":
		var expiresAt *time.Time
		if body.ExpiresAt != nil {
			if t, err := time.Parse(time.RFC3339, *body.ExpiresAt); err == nil {
				expiresAt = &t
			}
		}
		if err := svc.Approve(r.Context(), requestID, expiresAt); err != nil {
			s.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
	case "reject":
		if err := svc.Reject(r.Context(), requestID, body.RejectReason); err != nil {
			s.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		s.jsonError(w, "action must be 'approve' or 'reject'", http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleProviderRevokeAccessRequest handles DELETE /api/v1/provider/agents/{id}/access-requests/{request_id}.
func (s *HTTPServer) handleProviderRevokeAccessRequest(w http.ResponseWriter, r *http.Request) {
	if s.useracl == nil {
		s.jsonError(w, "access control not enabled", http.StatusNotImplemented)
		return
	}

	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	agentID := r.PathValue("id")
	if err := s.verifyAgentOwnership(r, agentID, userID); err != nil {
		s.jsonError(w, err.Error(), http.StatusForbidden)
		return
	}

	svc, ok := s.useracl.(*useracl.Service)
	if !ok {
		s.jsonError(w, "access control not available", http.StatusNotImplemented)
		return
	}

	requestID := r.PathValue("request_id")
	if err := svc.Revoke(r.Context(), requestID); err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
