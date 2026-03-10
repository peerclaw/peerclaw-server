package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/peerclaw/peerclaw-server/internal/contacts"
	"github.com/peerclaw/peerclaw-server/internal/identity"
)

var errForbiddenNotOwner = fmt.Errorf("forbidden: not the owner")

// --- Agent-side contacts handlers (AuthMiddleware + OwnerOnlyMiddleware) ---

type addContactRequest struct {
	ContactAgentID string  `json:"contact_agent_id"`
	Alias          string  `json:"alias"`
	ExpiresAt      *string `json:"expires_at,omitempty"`
}

// handleAgentAddContact handles POST /api/v1/agents/{id}/contacts.
func (s *HTTPServer) handleAgentAddContact(w http.ResponseWriter, r *http.Request) {
	if s.contacts == nil {
		s.jsonError(w, "contacts not enabled", http.StatusNotImplemented)
		return
	}

	ownerID := r.PathValue("id")
	var req addContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ContactAgentID == "" {
		s.jsonError(w, "contact_agent_id is required", http.StatusBadRequest)
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			expiresAt = &t
		}
	}
	contact, err := s.contacts.Add(r.Context(), ownerID, req.ContactAgentID, req.Alias, expiresAt)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusCreated, contact)
}

// handleAgentListContacts handles GET /api/v1/agents/{id}/contacts.
func (s *HTTPServer) handleAgentListContacts(w http.ResponseWriter, r *http.Request) {
	if s.contacts == nil {
		s.jsonError(w, "contacts not enabled", http.StatusNotImplemented)
		return
	}

	ownerID := r.PathValue("id")
	list, err := s.contacts.ListByOwner(r.Context(), ownerID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []contacts.Contact{}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"contacts": list})
}

// handleAgentRemoveContact handles DELETE /api/v1/agents/{id}/contacts/{contact_id}.
func (s *HTTPServer) handleAgentRemoveContact(w http.ResponseWriter, r *http.Request) {
	if s.contacts == nil {
		s.jsonError(w, "contacts not enabled", http.StatusNotImplemented)
		return
	}

	ownerID := r.PathValue("id")
	contactAgentID := r.PathValue("contact_id")

	if err := s.contacts.Remove(r.Context(), ownerID, contactAgentID); err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Provider-side contacts handlers (UserAuthMiddleware + ownership check) ---

// handleProviderAddContact handles POST /api/v1/provider/agents/{id}/contacts.
func (s *HTTPServer) handleProviderAddContact(w http.ResponseWriter, r *http.Request) {
	if s.contacts == nil {
		s.jsonError(w, "contacts not enabled", http.StatusNotImplemented)
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

	var req addContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ContactAgentID == "" {
		s.jsonError(w, "contact_agent_id is required", http.StatusBadRequest)
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			expiresAt = &t
		}
	}
	contact, err := s.contacts.Add(r.Context(), agentID, req.ContactAgentID, req.Alias, expiresAt)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusCreated, contact)
}

// handleProviderListContacts handles GET /api/v1/provider/agents/{id}/contacts.
func (s *HTTPServer) handleProviderListContacts(w http.ResponseWriter, r *http.Request) {
	if s.contacts == nil {
		s.jsonError(w, "contacts not enabled", http.StatusNotImplemented)
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

	list, err := s.contacts.ListByOwner(r.Context(), agentID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []contacts.Contact{}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"contacts": list})
}

// handleProviderRemoveContact handles DELETE /api/v1/provider/agents/{id}/contacts/{contact_id}.
func (s *HTTPServer) handleProviderRemoveContact(w http.ResponseWriter, r *http.Request) {
	if s.contacts == nil {
		s.jsonError(w, "contacts not enabled", http.StatusNotImplemented)
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

	contactAgentID := r.PathValue("contact_id")
	if err := s.contacts.Remove(r.Context(), agentID, contactAgentID); err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// verifyAgentOwnership checks that the given user owns the agent.
func (s *HTTPServer) verifyAgentOwnership(r *http.Request, agentID, userID string) error {
	card, err := s.registry.GetAgent(r.Context(), agentID)
	if err != nil {
		return err
	}
	if card.Metadata == nil || card.Metadata["owner_user_id"] != userID {
		return errForbiddenNotOwner
	}
	return nil
}
