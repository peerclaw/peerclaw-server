package server

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/peerclaw/peerclaw-server/internal/registry"
)

// --- Admin Dashboard ---

// dashboardStats holds aggregated dashboard metrics.
type dashboardStats struct {
	TotalUsers       int               `json:"total_users"`
	TotalAgents      int               `json:"total_agents"`
	ConnectedAgents  int               `json:"connected_agents"`
	TotalInvocations int               `json:"total_invocations"`
	TotalReviews     int               `json:"total_reviews"`
	PendingReports   int               `json:"pending_reports"`
	Health           map[string]string `json:"health"`
	Trends           map[string]any    `json:"trends,omitempty"`
}

// buildDashboardStats aggregates metrics from all subsystems into a single snapshot.
func (s *HTTPServer) buildDashboardStats(ctx context.Context) *dashboardStats {
	ds := &dashboardStats{
		Health: map[string]string{"status": "ok"},
	}

	// Total users.
	if s.userAuth != nil {
		if count, err := s.userAuth.CountUsers(ctx); err == nil {
			ds.TotalUsers = count
		}
	}

	// Total agents & connected agents.
	if s.store != nil {
		if result, err := s.store.List(ctx, registry.ListFilter{PageSize: 1}); err == nil {
			ds.TotalAgents = result.TotalCount
		}
	}
	if s.sigHub != nil {
		ds.ConnectedAgents = s.sigHub.ConnectedAgents()
	}

	// Total invocations.
	if s.invocation != nil {
		if count, err := s.invocation.CountInvocations(ctx); err == nil {
			ds.TotalInvocations = count
		}
	}

	// Total reviews & pending reports.
	if s.reviewService != nil {
		if count, err := s.reviewService.CountReviews(ctx); err == nil {
			ds.TotalReviews = count
		}
		if count, err := s.reviewService.CountReports(ctx, "pending"); err == nil {
			ds.PendingReports = count
		}
	}

	// Health.
	if s.store != nil {
		if _, err := s.store.List(ctx, registry.ListFilter{PageSize: 1}); err != nil {
			ds.Health["status"] = "degraded"
			ds.Health["database"] = "error"
		} else {
			ds.Health["database"] = "ok"
		}
	}

	// Trends (last 7 days).
	trends := map[string]any{}
	if s.invocation != nil {
		since7d := time.Now().Add(-7 * 24 * time.Hour)
		if stats7d, err := s.invocation.GlobalStats(ctx, since7d); err == nil {
			trends["invocations_7d"] = stats7d.TotalCalls
		}
	}
	if len(trends) > 0 {
		ds.Trends = trends
	}

	return ds
}

// handleAdminDashboard handles GET /api/v1/admin/dashboard.
func (s *HTTPServer) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	stats := s.buildDashboardStats(r.Context())
	s.jsonResponse(w, http.StatusOK, stats)
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
	sortBy := r.URL.Query().Get("sort")
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	var since time.Time
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}

	records, total, err := s.invocation.ListAll(r.Context(), agentID, userID, sortBy, since, limit, offset)
	if err != nil {
		s.internalError(w, r, "list invocations", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"invocations": records,
		"total":       total,
	})
}

// handleAdminGetInvocation handles GET /api/v1/admin/invocations/{id}.
func (s *HTTPServer) handleAdminGetInvocation(w http.ResponseWriter, r *http.Request) {
	if s.invocation == nil {
		s.jsonError(w, "invocation tracking not enabled", http.StatusNotImplemented)
		return
	}

	id := r.PathValue("id")
	record, err := s.invocation.GetByID(r.Context(), id)
	if err != nil {
		s.jsonError(w, "invocation not found", http.StatusNotFound)
		return
	}

	s.jsonResponse(w, http.StatusOK, record)
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
