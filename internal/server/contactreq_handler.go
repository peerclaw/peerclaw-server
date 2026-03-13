package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	coresignaling "github.com/peerclaw/peerclaw-core/signaling"
	"github.com/peerclaw/peerclaw-server/internal/contactreq"
	"github.com/peerclaw/peerclaw-server/internal/identity"
)

// --- Agent-side contact request handlers (AuthMiddleware + OwnerOnlyMiddleware) ---

type sendContactRequestBody struct {
	TargetAgentID string `json:"target_agent_id"`
	Message       string `json:"message"`
}

// handleAgentSendContactRequest handles POST /api/v1/agents/{id}/contact-requests.
func (s *HTTPServer) handleAgentSendContactRequest(w http.ResponseWriter, r *http.Request) {
	if s.contactReq == nil {
		s.jsonError(w, "contact requests not enabled", http.StatusNotImplemented)
		return
	}

	fromAgentID := r.PathValue("id")
	var body sendContactRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.TargetAgentID == "" {
		s.jsonError(w, "target_agent_id is required", http.StatusBadRequest)
		return
	}

	req, err := s.contactReq.Submit(r.Context(), fromAgentID, body.TargetAgentID, body.Message)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusCreated, req)
}

// handleAgentListIncomingContactRequests handles GET /api/v1/agents/{id}/contact-requests/incoming.
func (s *HTTPServer) handleAgentListIncomingContactRequests(w http.ResponseWriter, r *http.Request) {
	if s.contactReq == nil {
		s.jsonError(w, "contact requests not enabled", http.StatusNotImplemented)
		return
	}

	agentID := r.PathValue("id")
	status := r.URL.Query().Get("status")
	requests, err := s.contactReq.ListIncoming(r.Context(), agentID, status)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if requests == nil {
		requests = []contactreq.ContactRequest{}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"requests": requests})
}

// handleAgentListSentContactRequests handles GET /api/v1/agents/{id}/contact-requests/sent.
func (s *HTTPServer) handleAgentListSentContactRequests(w http.ResponseWriter, r *http.Request) {
	if s.contactReq == nil {
		s.jsonError(w, "contact requests not enabled", http.StatusNotImplemented)
		return
	}

	agentID := r.PathValue("id")
	status := r.URL.Query().Get("status")
	requests, err := s.contactReq.ListSent(r.Context(), agentID, status)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if requests == nil {
		requests = []contactreq.ContactRequest{}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"requests": requests})
}

type updateContactRequestBody struct {
	Action string `json:"action"` // "approve" or "reject"
	Reason string `json:"reason,omitempty"`
}

// handleAgentUpdateContactRequest handles PUT /api/v1/agents/{id}/contact-requests/{request_id}.
func (s *HTTPServer) handleAgentUpdateContactRequest(w http.ResponseWriter, r *http.Request) {
	if s.contactReq == nil {
		s.jsonError(w, "contact requests not enabled", http.StatusNotImplemented)
		return
	}

	requestID := r.PathValue("request_id")
	var body updateContactRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	switch body.Action {
	case "approve":
		req, err := s.contactReq.Approve(r.Context(), requestID)
		if err != nil {
			s.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.notifyContactAdded(r.Context(), req)
	case "reject":
		if err := s.contactReq.Reject(r.Context(), requestID, body.Reason); err != nil {
			s.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		s.jsonError(w, "action must be 'approve' or 'reject'", http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// notifyContactAdded pushes a contact_added WebSocket notification to both agents.
func (s *HTTPServer) notifyContactAdded(ctx context.Context, req *contactreq.ContactRequest) {
	if s.sigHub == nil || req == nil {
		return
	}
	for _, pair := range [][2]string{
		{req.FromAgentID, req.ToAgentID},
		{req.ToAgentID, req.FromAgentID},
	} {
		notif := coresignaling.SignalMessage{
			Type:    coresignaling.MessageTypeContactAdded,
			From:    "server",
			To:      pair[0],
			Payload: json.RawMessage(fmt.Sprintf(`{"agent_id":"%s"}`, pair[1])),
		}
		s.sigHub.DeliverLocal(ctx, notif)
	}
}

// --- Provider-side contact request handlers (UserAuthMiddleware + ownership check) ---

// handleProviderListContactRequests handles GET /api/v1/provider/agents/{id}/contact-requests.
func (s *HTTPServer) handleProviderListContactRequests(w http.ResponseWriter, r *http.Request) {
	if s.contactReq == nil {
		s.jsonError(w, "contact requests not enabled", http.StatusNotImplemented)
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

	status := r.URL.Query().Get("status")
	requests, err := s.contactReq.ListIncoming(r.Context(), agentID, status)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if requests == nil {
		requests = []contactreq.ContactRequest{}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"requests": requests})
}

// handleProviderUpdateContactRequest handles PUT /api/v1/provider/agents/{id}/contact-requests/{request_id}.
func (s *HTTPServer) handleProviderUpdateContactRequest(w http.ResponseWriter, r *http.Request) {
	if s.contactReq == nil {
		s.jsonError(w, "contact requests not enabled", http.StatusNotImplemented)
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

	requestID := r.PathValue("request_id")
	var body updateContactRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	switch body.Action {
	case "approve":
		req, err := s.contactReq.Approve(r.Context(), requestID)
		if err != nil {
			s.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.notifyContactAdded(r.Context(), req)
	case "reject":
		if err := s.contactReq.Reject(r.Context(), requestID, body.Reason); err != nil {
			s.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		s.jsonError(w, "action must be 'approve' or 'reject'", http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}
