package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/identity"
	"github.com/peerclaw/peerclaw-server/internal/registry"
)

// handleProviderPublishAgent handles POST /api/v1/provider/agents.
func (s *HTTPServer) handleProviderPublishAgent(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := validateRegisterRequest(&req); err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	protocols := make([]protocol.Protocol, len(req.Protocols))
	for i, p := range req.Protocols {
		protocols[i] = protocol.Protocol(p)
	}

	card, err := s.registry.Register(r.Context(), registry.RegisterRequest{
		Name:         req.Name,
		Description:  req.Description,
		Version:      req.Version,
		PublicKey:    req.PublicKey,
		Capabilities: req.Capabilities,
		Endpoint: agentcard.Endpoint{
			URL:       req.Endpoint.URL,
			Host:      req.Endpoint.Host,
			Port:      req.Endpoint.Port,
			Transport: protocol.Transport(req.Endpoint.Transport),
		},
		Protocols: protocols,
		Auth: agentcard.AuthInfo{
			Type:   req.Auth.Type,
			Params: req.Auth.Params,
		},
		Metadata: req.Metadata,
		PeerClaw: agentcard.PeerClawExtension{
			NATType:         req.PeerClaw.NATType,
			RelayPreference: req.PeerClaw.RelayPreference,
			Priority:        req.PeerClaw.Priority,
			Tags:            req.PeerClaw.Tags,
			PublicEndpoint:  req.PeerClaw.PublicEndpoint,
		},
		OwnerUserID: userID,
	})
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.engine.UpdateFromCard(card)

	if s.reputation != nil {
		_ = s.reputation.RecordEvent(r.Context(), card.ID, "registration", "")
	}

	s.jsonResponse(w, http.StatusCreated, card)
}

// handleProviderListAgents handles GET /api/v1/provider/agents.
func (s *HTTPServer) handleProviderListAgents(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	filter := registry.ListFilter{
		OwnerUserID: userID,
	}

	result, err := s.registry.ListAgents(r.Context(), filter)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, result)
}

// handleProviderGetAgent handles GET /api/v1/provider/agents/{id}.
func (s *HTTPServer) handleProviderGetAgent(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	card, err := s.registry.GetAgent(r.Context(), id)
	if err != nil {
		s.jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	// Verify ownership via metadata.
	if card.Metadata == nil || card.Metadata["owner_user_id"] != userID {
		s.jsonError(w, "forbidden: not the owner", http.StatusForbidden)
		return
	}

	s.jsonResponse(w, http.StatusOK, card)
}

// handleProviderUpdateAgent handles PUT /api/v1/provider/agents/{id}.
func (s *HTTPServer) handleProviderUpdateAgent(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	existing, err := s.registry.GetAgent(r.Context(), id)
	if err != nil {
		s.jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	if existing.Metadata == nil || existing.Metadata["owner_user_id"] != userID {
		s.jsonError(w, "forbidden: not the owner", http.StatusForbidden)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Update the card fields.
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Version != "" {
		existing.Version = req.Version
	}
	if len(req.Capabilities) > 0 {
		existing.Capabilities = req.Capabilities
	}
	if req.Endpoint.URL != "" {
		existing.Endpoint.URL = req.Endpoint.URL
	}
	if req.Endpoint.Host != "" {
		existing.Endpoint.Host = req.Endpoint.Host
	}
	if req.Endpoint.Port > 0 {
		existing.Endpoint.Port = req.Endpoint.Port
	}
	if req.Endpoint.Transport != "" {
		existing.Endpoint.Transport = protocol.Transport(req.Endpoint.Transport)
	}
	if len(req.Protocols) > 0 {
		protocols := make([]protocol.Protocol, len(req.Protocols))
		for i, p := range req.Protocols {
			protocols[i] = protocol.Protocol(p)
		}
		existing.Protocols = protocols
	}

	if err := s.store.Put(r.Context(), existing); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.engine.UpdateFromCard(existing)
	s.jsonResponse(w, http.StatusOK, existing)
}

// handleProviderDeleteAgent handles DELETE /api/v1/provider/agents/{id}.
func (s *HTTPServer) handleProviderDeleteAgent(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	existing, err := s.registry.GetAgent(r.Context(), id)
	if err != nil {
		s.jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	if existing.Metadata == nil || existing.Metadata["owner_user_id"] != userID {
		s.jsonError(w, "forbidden: not the owner", http.StatusForbidden)
		return
	}

	if err := s.registry.Deregister(r.Context(), id); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.engine.RemoveAgent(id)
	w.WriteHeader(http.StatusNoContent)
}

// handleProviderAgentAnalytics handles GET /api/v1/provider/agents/{id}/analytics.
func (s *HTTPServer) handleProviderAgentAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	existing, err := s.registry.GetAgent(r.Context(), id)
	if err != nil {
		s.jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	if existing.Metadata == nil || existing.Metadata["owner_user_id"] != userID {
		s.jsonError(w, "forbidden: not the owner", http.StatusForbidden)
		return
	}

	if s.invocation == nil {
		s.jsonError(w, "analytics not available", http.StatusServiceUnavailable)
		return
	}

	since := time.Now().Add(-24 * time.Hour)
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}

	bucketMinutes := 60
	if bm := r.URL.Query().Get("bucket_minutes"); bm != "" {
		if b, err := strconv.Atoi(bm); err == nil && b > 0 {
			bucketMinutes = b
		}
	}

	stats, err := s.invocation.AgentStats(r.Context(), id, since)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	timeSeries, err := s.invocation.AgentTimeSeries(r.Context(), id, since, bucketMinutes)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"stats":       stats,
		"time_series": timeSeries,
	})
}

// handleProviderDashboard handles GET /api/v1/provider/dashboard.
func (s *HTTPServer) handleProviderDashboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Get all agents owned by this user.
	result, err := s.registry.ListAgents(r.Context(), registry.ListFilter{
		OwnerUserID: userID,
	})
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build per-agent entries with invocation stats.
	allTimeSince := time.Time{} // zero value = all time
	agents := make([]map[string]any, 0, len(result.Agents))
	for _, card := range result.Agents {
		agent := map[string]any{
			"id":           card.ID,
			"name":         card.Name,
			"description":  card.Description,
			"version":      card.Version,
			"capabilities": card.Capabilities,
			"protocols":    card.Protocols,
			"status":       card.Status,
			"endpoint_url": card.Endpoint.URL,
			"auth_type":    card.Auth.Type,
			"tags":         card.PeerClaw.Tags,
			"created_at":   card.RegisteredAt,
			"updated_at":   card.LastHeartbeat,
		}

		var totalCalls, successCalls int64
		var avgLatency float64
		if s.invocation != nil {
			if st, err := s.invocation.AgentStats(r.Context(), card.ID, allTimeSince); err == nil {
				totalCalls = st.TotalCalls
				successCalls = st.SuccessCalls
				avgLatency = st.AvgDurationMs
			}
		}
		var successRate float64
		if totalCalls > 0 {
			successRate = float64(successCalls) / float64(totalCalls) * 100
		}
		agent["total_calls"] = totalCalls
		agent["success_rate"] = successRate
		agent["avg_latency_ms"] = avgLatency

		agents = append(agents, agent)
	}

	// Aggregated top-level stats.
	var totalCalls int64
	var successRate, avgLatency float64
	if s.invocation != nil {
		if stats, err := s.invocation.ProviderDashboardStats(r.Context(), userID); err == nil {
			totalCalls = stats.TotalCalls
			avgLatency = stats.AvgDurationMs
			if stats.TotalCalls > 0 {
				successRate = float64(stats.SuccessCalls) / float64(stats.TotalCalls) * 100
			}
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"total_agents":  result.TotalCount,
		"total_calls":   totalCalls,
		"success_rate":  successRate,
		"avg_latency_ms": avgLatency,
		"agents":        agents,
	})
}
