package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-server/internal/registry"
	"github.com/peerclaw/peerclaw-server/internal/review"
)

// --- Admin Dashboard ---

// handleAdminDashboard handles GET /api/v1/admin/dashboard.
func (s *HTTPServer) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{}

	// Total users.
	if s.userAuth != nil {
		if count, err := s.userAuth.CountUsers(r.Context()); err == nil {
			resp["total_users"] = count
		}
	}

	// Total agents & connected agents.
	if s.store != nil {
		if result, err := s.store.List(r.Context(), registry.ListFilter{PageSize: 1}); err == nil {
			resp["total_agents"] = result.TotalCount
		}
	}
	if s.sigHub != nil {
		resp["connected_agents"] = s.sigHub.ConnectedAgents()
	}

	// Total invocations.
	if s.invocation != nil {
		if count, err := s.invocation.CountInvocations(r.Context()); err == nil {
			resp["total_invocations"] = count
		}
	}

	// Total reviews & pending reports.
	if s.reviewService != nil {
		if count, err := s.reviewService.CountReviews(r.Context()); err == nil {
			resp["total_reviews"] = count
		}
		if count, err := s.reviewService.CountReports(r.Context(), "pending"); err == nil {
			resp["pending_reports"] = count
		}
	}

	// Health.
	health := map[string]string{"status": "ok"}
	if s.store != nil {
		if _, err := s.store.List(r.Context(), registry.ListFilter{PageSize: 1}); err != nil {
			health["status"] = "degraded"
			health["database"] = "error"
		} else {
			health["database"] = "ok"
		}
	}
	resp["health"] = health

	// Trends (last 7 days).
	trends := map[string]any{}
	if s.invocation != nil {
		since7d := time.Now().Add(-7 * 24 * time.Hour)
		if stats7d, err := s.invocation.GlobalStats(r.Context(), since7d); err == nil {
			trends["invocations_7d"] = stats7d.TotalCalls
		}
	}
	if len(trends) > 0 {
		resp["trends"] = trends
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

// --- User Management ---

// handleAdminListUsers handles GET /api/v1/admin/users.
func (s *HTTPServer) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	if s.userAuth == nil {
		s.jsonError(w, "user authentication not enabled", http.StatusNotImplemented)
		return
	}

	search := r.URL.Query().Get("search")
	role := r.URL.Query().Get("role")
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	users, total, err := s.userAuth.ListUsers(r.Context(), search, role, limit, offset)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sanitized := make([]map[string]any, len(users))
	for i := range users {
		sanitized[i] = sanitizeUser(&users[i])
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"users": sanitized,
		"total": total,
	})
}

// handleAdminGetUser handles GET /api/v1/admin/users/{id}.
func (s *HTTPServer) handleAdminGetUser(w http.ResponseWriter, r *http.Request) {
	if s.userAuth == nil {
		s.jsonError(w, "user authentication not enabled", http.StatusNotImplemented)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		s.jsonError(w, "user id is required", http.StatusBadRequest)
		return
	}
	user, err := s.userAuth.GetUser(r.Context(), id)
	if err != nil {
		s.jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	s.jsonResponse(w, http.StatusOK, sanitizeUser(user))
}

// handleAdminUpdateUserRole handles PUT /api/v1/admin/users/{id}/role.
func (s *HTTPServer) handleAdminUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	if s.userAuth == nil {
		s.jsonError(w, "user authentication not enabled", http.StatusNotImplemented)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		s.jsonError(w, "user id is required", http.StatusBadRequest)
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := s.userAuth.UpdateRole(r.Context(), id, req.Role)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusOK, sanitizeUser(user))
}

// handleAdminDeleteUser handles DELETE /api/v1/admin/users/{id}.
func (s *HTTPServer) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	if s.userAuth == nil {
		s.jsonError(w, "user authentication not enabled", http.StatusNotImplemented)
		return
	}

	id := r.PathValue("id")
	if err := s.userAuth.DeleteUser(r.Context(), id); err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Agent Management ---

// handleAdminListAgents handles GET /api/v1/admin/agents.
func (s *HTTPServer) handleAdminListAgents(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	filter := registry.ListFilter{
		Search:   r.URL.Query().Get("search"),
		Protocol: r.URL.Query().Get("protocol"),
		Status:   agentcard.AgentStatus(r.URL.Query().Get("status")),
		PageSize: limit,
		PageToken: fmt.Sprintf("%d", offset),
	}

	result, err := s.registry.ListAgents(r.Context(), filter)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
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
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	s.engine.RemoveAgent(id)
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
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

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
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "unverified"})
}

// --- Report Moderation ---

// handleAdminListReports handles GET /api/v1/admin/reports.
func (s *HTTPServer) handleAdminListReports(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "review service not enabled", http.StatusNotImplemented)
		return
	}

	status := r.URL.Query().Get("status")
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	reports, total, err := s.reviewService.ListReports(r.Context(), status, limit, offset)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"reports": reports,
		"total":   total,
	})
}

// handleAdminGetReport handles GET /api/v1/admin/reports/{id}.
func (s *HTTPServer) handleAdminGetReport(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "review service not enabled", http.StatusNotImplemented)
		return
	}

	id := r.PathValue("id")
	report, err := s.reviewService.GetReport(r.Context(), id)
	if err != nil {
		s.jsonError(w, "report not found", http.StatusNotFound)
		return
	}

	s.jsonResponse(w, http.StatusOK, report)
}

// handleAdminUpdateReport handles PUT /api/v1/admin/reports/{id}.
func (s *HTTPServer) handleAdminUpdateReport(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "review service not enabled", http.StatusNotImplemented)
		return
	}

	id := r.PathValue("id")
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.reviewService.UpdateReportStatus(r.Context(), id, req.Status); err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": req.Status})
}

// handleAdminDeleteReport handles DELETE /api/v1/admin/reports/{id}.
func (s *HTTPServer) handleAdminDeleteReport(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "review service not enabled", http.StatusNotImplemented)
		return
	}

	id := r.PathValue("id")
	if err := s.reviewService.DeleteReport(r.Context(), id); err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Category Management ---

// handleAdminCreateCategory handles POST /api/v1/admin/categories.
func (s *HTTPServer) handleAdminCreateCategory(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "review service not enabled", http.StatusNotImplemented)
		return
	}

	var cat review.Category
	if err := json.NewDecoder(r.Body).Decode(&cat); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if cat.Name == "" || cat.Slug == "" {
		s.jsonError(w, "name and slug are required", http.StatusBadRequest)
		return
	}

	if err := s.reviewService.CreateCategory(r.Context(), &cat); err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusCreated, cat)
}

// handleAdminUpdateCategory handles PUT /api/v1/admin/categories/{id}.
func (s *HTTPServer) handleAdminUpdateCategory(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "review service not enabled", http.StatusNotImplemented)
		return
	}

	id := r.PathValue("id")
	var cat review.Category
	if err := json.NewDecoder(r.Body).Decode(&cat); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	cat.ID = id

	if err := s.reviewService.UpdateCategory(r.Context(), &cat); err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	s.jsonResponse(w, http.StatusOK, cat)
}

// handleAdminDeleteCategory handles DELETE /api/v1/admin/categories/{id}.
func (s *HTTPServer) handleAdminDeleteCategory(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "review service not enabled", http.StatusNotImplemented)
		return
	}

	id := r.PathValue("id")
	if err := s.reviewService.DeleteCategory(r.Context(), id); err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Global Analytics ---

// handleAdminGlobalAnalytics handles GET /api/v1/admin/analytics.
func (s *HTTPServer) handleAdminGlobalAnalytics(w http.ResponseWriter, r *http.Request) {
	if s.invocation == nil {
		s.jsonError(w, "invocation tracking not enabled", http.StatusNotImplemented)
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

	resp := map[string]any{}

	if stats, err := s.invocation.GlobalStats(r.Context(), since); err == nil {
		resp["stats"] = stats
	}

	if ts, err := s.invocation.GlobalTimeSeries(r.Context(), since, bucketMinutes); err == nil {
		resp["time_series"] = ts
	}

	if top, err := s.invocation.TopAgents(r.Context(), since, 10); err == nil {
		resp["top_agents"] = top
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

// --- Invocation Log ---

// handleAdminListInvocations handles GET /api/v1/admin/invocations.
func (s *HTTPServer) handleAdminListInvocations(w http.ResponseWriter, r *http.Request) {
	if s.invocation == nil {
		s.jsonError(w, "invocation tracking not enabled", http.StatusNotImplemented)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	userID := r.URL.Query().Get("user_id")
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	records, total, err := s.invocation.ListAll(r.Context(), agentID, userID, limit, offset)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"invocations": records,
		"total":       total,
	})
}

// --- Route Registration ---

// registerAdminRoutes registers all admin API routes.
func (s *HTTPServer) registerAdminRoutes() {
	wrapAdmin := func(h http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.userAuth == nil {
				s.jsonError(w, "user authentication not enabled", http.StatusNotImplemented)
				return
			}
			userAuthMW := UserAuthMiddleware(s.userAuth.JWTManager(), s.logger)
			adminMW := AdminOnlyMiddleware(s.logger)
			userAuthMW(adminMW(http.HandlerFunc(h))).ServeHTTP(w, r)
		})
	}

	// Dashboard.
	s.mux.Handle("GET /api/v1/admin/dashboard", wrapAdmin(s.handleAdminDashboard))

	// User management.
	s.mux.Handle("GET /api/v1/admin/users", wrapAdmin(s.handleAdminListUsers))
	s.mux.Handle("GET /api/v1/admin/users/{id}", wrapAdmin(s.handleAdminGetUser))
	s.mux.Handle("PUT /api/v1/admin/users/{id}/role", wrapAdmin(s.handleAdminUpdateUserRole))
	s.mux.Handle("DELETE /api/v1/admin/users/{id}", wrapAdmin(s.handleAdminDeleteUser))

	// Agent management.
	s.mux.Handle("GET /api/v1/admin/agents", wrapAdmin(s.handleAdminListAgents))
	s.mux.Handle("GET /api/v1/admin/agents/{id}", wrapAdmin(s.handleAdminGetAgent))
	s.mux.Handle("DELETE /api/v1/admin/agents/{id}", wrapAdmin(s.handleAdminDeleteAgent))
	s.mux.Handle("POST /api/v1/admin/agents/{id}/verify", wrapAdmin(s.handleAdminVerifyAgent))
	s.mux.Handle("DELETE /api/v1/admin/agents/{id}/verify", wrapAdmin(s.handleAdminUnverifyAgent))

	// Report moderation.
	s.mux.Handle("GET /api/v1/admin/reports", wrapAdmin(s.handleAdminListReports))
	s.mux.Handle("GET /api/v1/admin/reports/{id}", wrapAdmin(s.handleAdminGetReport))
	s.mux.Handle("PUT /api/v1/admin/reports/{id}", wrapAdmin(s.handleAdminUpdateReport))
	s.mux.Handle("DELETE /api/v1/admin/reports/{id}", wrapAdmin(s.handleAdminDeleteReport))

	// Category management.
	s.mux.Handle("POST /api/v1/admin/categories", wrapAdmin(s.handleAdminCreateCategory))
	s.mux.Handle("PUT /api/v1/admin/categories/{id}", wrapAdmin(s.handleAdminUpdateCategory))
	s.mux.Handle("DELETE /api/v1/admin/categories/{id}", wrapAdmin(s.handleAdminDeleteCategory))

	// Global analytics.
	s.mux.Handle("GET /api/v1/admin/analytics", wrapAdmin(s.handleAdminGlobalAnalytics))

	// Invocation log.
	s.mux.Handle("GET /api/v1/admin/invocations", wrapAdmin(s.handleAdminListInvocations))

	// SDK version check.
	s.mux.Handle("GET /api/v1/admin/sdk-version", wrapAdmin(s.handleAdminSDKVersion))
}

// handleAdminSDKVersion handles GET /api/v1/admin/sdk-version.
func (s *HTTPServer) handleAdminSDKVersion(w http.ResponseWriter, r *http.Request) {
	if s.versionCheck == nil {
		s.jsonError(w, "version check not enabled", http.StatusNotImplemented)
		return
	}
	latest, releaseURL := s.versionCheck.Latest()
	s.jsonResponse(w, http.StatusOK, map[string]any{
		"latest":      latest,
		"release_url": releaseURL,
	})
}

// queryInt extracts an integer query parameter with a default value.
// Values are clamped to [1, 200] to prevent abuse.
func queryInt(r *http.Request, key string, defaultVal int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			return n
		}
	}
	return defaultVal
}
