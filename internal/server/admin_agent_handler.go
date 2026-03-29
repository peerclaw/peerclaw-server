package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-server/internal/registry"
)

// --- Agent Management ---

// handleAdminListAgents handles GET /api/v1/admin/agents.
func (s *HTTPServer) handleAdminListAgents(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	filter := registry.ListFilter{
		Search:    r.URL.Query().Get("search"),
		Protocol:  r.URL.Query().Get("protocol"),
		Status:    agentcard.AgentStatus(r.URL.Query().Get("status")),
		SortBy:    r.URL.Query().Get("sort"),
		PageSize:  limit,
		PageToken: fmt.Sprintf("%d", offset),
	}

	result, err := s.registry.ListAgents(r.Context(), filter)
	if err != nil {
		s.internalError(w, r, "list agents", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, result)
}

// handleAdminGetAgent handles GET /api/v1/admin/agents/{id}.
func (s *HTTPServer) handleAdminGetAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	card, err := s.registry.GetAgent(r.Context(), id)
	if err != nil {
		s.jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	resp := map[string]any{
		"agent": card,
	}

	// Owner info.
	if s.userAuth != nil && card.Metadata != nil {
		if ownerID, ok := card.Metadata["owner_user_id"]; ok && ownerID != "" {
			if owner, err := s.userAuth.GetUser(r.Context(), ownerID); err == nil {
				resp["owner"] = owner
			}
		}
	}

	// Reputation.
	if s.reputation != nil {
		if score, err := s.reputation.GetScore(r.Context(), id); err == nil {
			resp["reputation_score"] = score
		}
		if events, err := s.reputation.GetHistory(r.Context(), id, 20); err == nil {
			resp["reputation_events"] = events
		}
	}

	// Review summary.
	if s.reviewService != nil {
		if summary, err := s.reviewService.GetSummary(r.Context(), id); err == nil {
			resp["review_summary"] = summary
		}
	}

	// Invocation stats.
	if s.invocation != nil {
		since := time.Now().Add(-30 * 24 * time.Hour)
		if stats, err := s.invocation.AgentStats(r.Context(), id, since); err == nil {
			resp["invocation_stats"] = stats
		}
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

// handleAdminDeleteAgent handles DELETE /api/v1/admin/agents/{id}.
func (s *HTTPServer) handleAdminDeleteAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.registry.Deregister(r.Context(), id); err != nil {
		s.jsonError(w, "not found", http.StatusNotFound)
		return
	}
	s.engine.RemoveAgent(id)
	s.recordAdminAudit(r, "agent.delete", "agent", id, "")
	w.WriteHeader(http.StatusNoContent)
}

// handleAdminVerifyAgent handles POST /api/v1/admin/agents/{id}/verify.
func (s *HTTPServer) handleAdminVerifyAgent(w http.ResponseWriter, r *http.Request) {
	if s.reputation == nil {
		s.jsonError(w, "reputation engine not enabled", http.StatusNotImplemented)
		return
	}

	id := r.PathValue("id")
	if err := s.reputation.SetVerified(r.Context(), id); err != nil {
		s.internalError(w, r, "verify agent", err)
		return
	}

	s.recordAdminAudit(r, "agent.verify", "agent", id, "")
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "verified"})
}

// handleAdminUnverifyAgent handles DELETE /api/v1/admin/agents/{id}/verify.
func (s *HTTPServer) handleAdminUnverifyAgent(w http.ResponseWriter, r *http.Request) {
	if s.reputation == nil {
		s.jsonError(w, "reputation engine not enabled", http.StatusNotImplemented)
		return
	}

	id := r.PathValue("id")
	if err := s.reputation.UnsetVerified(r.Context(), id); err != nil {
		s.internalError(w, r, "unverify agent", err)
		return
	}

	s.recordAdminAudit(r, "agent.unverify", "agent", id, "")
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "unverified"})
}

// handleAdminBulkAgents handles POST /api/v1/admin/agents/bulk.
func (s *HTTPServer) handleAdminBulkAgents(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string   `json:"action"`
		IDs    []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.IDs) == 0 {
		s.jsonError(w, "ids is required", http.StatusBadRequest)
		return
	}

	var successCount, errorCount int
	for _, id := range req.IDs {
		var err error
		switch req.Action {
		case "verify":
			if s.reputation != nil {
				err = s.reputation.SetVerified(r.Context(), id)
			}
		case "unverify":
			if s.reputation != nil {
				err = s.reputation.UnsetVerified(r.Context(), id)
			}
		case "delete":
			err = s.registry.Deregister(r.Context(), id)
			if err == nil {
				s.engine.RemoveAgent(id)
			}
		default:
			s.jsonError(w, "invalid action: must be verify, unverify, or delete", http.StatusBadRequest)
			return
		}
		if err != nil {
			errorCount++
		} else {
			successCount++
			s.recordAdminAudit(r, "agent."+req.Action, "agent", id, "")
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"success": successCount,
		"errors":  errorCount,
	})
}
