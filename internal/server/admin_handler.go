package server

import (
	"net/http"
	"strconv"
)

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

	// Dashboard & analytics.
	s.mux.Handle("GET /api/v1/admin/dashboard", wrapAdmin(s.handleAdminDashboard))
	s.mux.Handle("GET /api/v1/admin/analytics", wrapAdmin(s.handleAdminGlobalAnalytics))
	s.mux.Handle("GET /api/v1/admin/sdk-version", wrapAdmin(s.handleAdminSDKVersion))

	// User management.
	s.mux.Handle("GET /api/v1/admin/users", wrapAdmin(s.handleAdminListUsers))
	s.mux.Handle("GET /api/v1/admin/users/{id}", wrapAdmin(s.handleAdminGetUser))
	s.mux.Handle("PUT /api/v1/admin/users/{id}/role", wrapAdmin(s.handleAdminUpdateUserRole))
	s.mux.Handle("DELETE /api/v1/admin/users/{id}", wrapAdmin(s.handleAdminDeleteUser))
	s.mux.Handle("POST /api/v1/admin/users/bulk", wrapAdmin(s.handleAdminBulkUsers))

	// Agent management.
	s.mux.Handle("GET /api/v1/admin/agents", wrapAdmin(s.handleAdminListAgents))
	s.mux.Handle("GET /api/v1/admin/agents/{id}", wrapAdmin(s.handleAdminGetAgent))
	s.mux.Handle("DELETE /api/v1/admin/agents/{id}", wrapAdmin(s.handleAdminDeleteAgent))
	s.mux.Handle("POST /api/v1/admin/agents/{id}/verify", wrapAdmin(s.handleAdminVerifyAgent))
	s.mux.Handle("DELETE /api/v1/admin/agents/{id}/verify", wrapAdmin(s.handleAdminUnverifyAgent))
	s.mux.Handle("POST /api/v1/admin/agents/bulk", wrapAdmin(s.handleAdminBulkAgents))

	// Report & category moderation.
	s.mux.Handle("GET /api/v1/admin/reports", wrapAdmin(s.handleAdminListReports))
	s.mux.Handle("GET /api/v1/admin/reports/{id}", wrapAdmin(s.handleAdminGetReport))
	s.mux.Handle("PUT /api/v1/admin/reports/{id}", wrapAdmin(s.handleAdminUpdateReport))
	s.mux.Handle("DELETE /api/v1/admin/reports/{id}", wrapAdmin(s.handleAdminDeleteReport))
	s.mux.Handle("POST /api/v1/admin/reports/bulk", wrapAdmin(s.handleAdminBulkReports))
	s.mux.Handle("POST /api/v1/admin/categories", wrapAdmin(s.handleAdminCreateCategory))
	s.mux.Handle("PUT /api/v1/admin/categories/{id}", wrapAdmin(s.handleAdminUpdateCategory))
	s.mux.Handle("DELETE /api/v1/admin/categories/{id}", wrapAdmin(s.handleAdminDeleteCategory))

	// Invocation log.
	s.mux.Handle("GET /api/v1/admin/invocations", wrapAdmin(s.handleAdminListInvocations))
	s.mux.Handle("GET /api/v1/admin/invocations/{id}", wrapAdmin(s.handleAdminGetInvocation))

	// Audit log.
	s.mux.Handle("GET /api/v1/admin/audit", wrapAdmin(s.handleAdminListAudit))
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
