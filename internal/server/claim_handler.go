package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/identity"
	"github.com/peerclaw/peerclaw-server/internal/registry"
)

// --- Generate Claim Token (JWT-protected) ---

func (s *HTTPServer) handleGenerateClaimToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := s.claimToken.Generate(r.Context(), userID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]any{
		"token":      token.Code,
		"expires_at": token.ExpiresAt.Format(time.RFC3339),
		"expires_in": int(time.Until(token.ExpiresAt).Seconds()),
	})
}

// --- List Claim Tokens (JWT-protected) ---

func (s *HTTPServer) handleListClaimTokens(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	tokens, err := s.claimToken.ListByUser(r.Context(), userID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"tokens": tokens})
}

// --- Agent Claim (no auth — the token IS the auth) ---

type claimAgentRequest struct {
	Token        string            `json:"token"`
	Name         string            `json:"name"`
	PublicKey    string            `json:"public_key"`
	Capabilities []string          `json:"capabilities,omitempty"`
	Protocols    []string          `json:"protocols"`
	Endpoint     endpointReq       `json:"endpoint"`
	Signature    string            `json:"signature"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

func (s *HTTPServer) handleClaimAgent(w http.ResponseWriter, r *http.Request) {
	var req claimAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields.
	if req.Token == "" || req.Name == "" || req.PublicKey == "" || req.Signature == "" {
		s.jsonError(w, "token, name, public_key, and signature are required", http.StatusBadRequest)
		return
	}
	if len(req.Protocols) == 0 {
		s.jsonError(w, "at least one protocol is required", http.StatusBadRequest)
		return
	}

	// 1. Validate token: exists, pending, not expired.
	ct, err := s.claimToken.Validate(r.Context(), req.Token)
	if err != nil {
		// Determine appropriate status code.
		status := http.StatusBadRequest
		switch {
		case contains(err.Error(), "already claimed"):
			status = http.StatusConflict
		case contains(err.Error(), "expired"):
			status = http.StatusGone
		case contains(err.Error(), "not found"):
			status = http.StatusNotFound
		}
		s.jsonError(w, err.Error(), status)
		return
	}

	// 2. Verify signature: the agent must prove ownership of the private key.
	if err := s.verifier.VerifySignature(req.PublicKey, []byte(req.Token), req.Signature); err != nil {
		s.jsonError(w, "invalid signature: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// 3. Register the agent (same as handleRegister), with owner bound to token's user.
	protocols := make([]protocol.Protocol, len(req.Protocols))
	for i, p := range req.Protocols {
		protocols[i] = protocol.Protocol(p)
	}

	card, err := s.registry.Register(r.Context(), registry.RegisterRequest{
		Name:         req.Name,
		PublicKey:    req.PublicKey,
		Capabilities: req.Capabilities,
		Endpoint: agentcard.Endpoint{
			URL:       req.Endpoint.URL,
			Host:      req.Endpoint.Host,
			Port:      req.Endpoint.Port,
			Transport: protocol.Transport(req.Endpoint.Transport),
		},
		Protocols:   protocols,
		Metadata:    req.Metadata,
		OwnerUserID: ct.UserID,
	})
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 4. Mark token as claimed.
	if err := s.claimToken.Claim(r.Context(), req.Token, card.ID); err != nil {
		s.logger.Error("failed to mark token as claimed", "error", err, "token", req.Token, "agent_id", card.ID)
		// Agent is registered; don't fail the response.
	}

	// 5. Update routing table.
	s.engine.UpdateFromCard(card)

	// Record reputation event.
	if s.reputation != nil {
		_ = s.reputation.RecordEvent(r.Context(), card.ID, "registration", "claim_token")
	}

	// Audit log.
	if s.audit != nil {
		s.audit.LogRegistration(r.Context(), card.ID, card.Name, r.RemoteAddr)
	}
	if s.metrics != nil {
		s.metrics.RegisteredAgents.Add(r.Context(), 1)
	}

	s.jsonResponse(w, http.StatusCreated, card)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
