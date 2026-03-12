package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/claimtoken"
	"github.com/peerclaw/peerclaw-server/internal/identity"
	"github.com/peerclaw/peerclaw-server/internal/registry"
)

// --- Generate Claim Token (JWT-protected) ---

type generateClaimTokenRequest struct {
	AgentName    string   `json:"agent_name"`
	Capabilities []string `json:"capabilities,omitempty"`
	Protocols    []string `json:"protocols,omitempty"`
}

func (s *HTTPServer) handleGenerateClaimToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req generateClaimTokenRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
	}

	if req.AgentName == "" {
		s.jsonError(w, "agent_name is required", http.StatusBadRequest)
		return
	}

	params := claimtoken.GenerateParams{
		AgentName:    req.AgentName,
		Capabilities: strings.Join(req.Capabilities, ","),
		Protocols:    strings.Join(req.Protocols, ","),
	}
	if params.Protocols == "" {
		params.Protocols = "a2a"
	}

	token, err := s.claimToken.Generate(r.Context(), userID, params)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]any{
		"token":      token.Code,
		"agent_name": token.AgentName,
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
	if req.Token == "" || req.PublicKey == "" || req.Signature == "" {
		s.jsonError(w, "token, public_key, and signature are required", http.StatusBadRequest)
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
	// Use token metadata as defaults; request fields override if provided.
	agentName := ct.AgentName
	if req.Name != "" {
		agentName = req.Name
	}
	if agentName == "" {
		s.jsonError(w, "agent name is required (set in token or request)", http.StatusBadRequest)
		return
	}

	protoList := req.Protocols
	if len(protoList) == 0 && ct.Protocols != "" {
		protoList = strings.Split(ct.Protocols, ",")
	}
	if len(protoList) == 0 {
		protoList = []string{"a2a"}
	}

	caps := req.Capabilities
	if len(caps) == 0 && ct.Capabilities != "" {
		caps = strings.Split(ct.Capabilities, ",")
	}

	protocols := make([]protocol.Protocol, len(protoList))
	for i, p := range protoList {
		protocols[i] = protocol.Protocol(p)
	}

	card, err := s.registry.Register(r.Context(), registry.RegisterRequest{
		Name:         agentName,
		PublicKey:    req.PublicKey,
		Capabilities: caps,
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
		if err := s.reputation.RecordEvent(r.Context(), card.ID, "registration", "claim_token"); err != nil {
			s.logger.Debug("failed to record reputation event", "agent_id", card.ID, "error", err)
		}
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
