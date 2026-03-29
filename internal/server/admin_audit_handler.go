package server

import (
	"net/http"
	"time"

	"github.com/peerclaw/peerclaw-server/internal/identity"
)

// --- Admin Audit Log ---

// recordAdminAudit is a helper that records an admin audit event if the service is available.
func (s *HTTPServer) recordAdminAudit(r *http.Request, action, targetType, targetID, details string) {
	if s.adminAudit == nil {
		return
	}
	adminUserID, _ := identity.UserIDFromContext(r.Context())
	s.adminAudit.Record(r.Context(), adminUserID, action, targetType, targetID, details, r.RemoteAddr)
}

// handleAdminListAudit handles GET /api/v1/admin/audit.
func (s *HTTPServer) handleAdminListAudit(w http.ResponseWriter, r *http.Request) {
	if s.adminAudit == nil {
		s.jsonError(w, "admin audit not enabled", http.StatusNotImplemented)
		return
	}

	adminUserID := r.URL.Query().Get("admin_user_id")
	action := r.URL.Query().Get("action")
	targetType := r.URL.Query().Get("target_type")
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	var since time.Time
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}

	events, total, err := s.adminAudit.List(r.Context(), adminUserID, action, targetType, since, limit, offset)
	if err != nil {
		s.internalError(w, r, "list audit events", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"events": events,
		"total":  total,
	})
}
