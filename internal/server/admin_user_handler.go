package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// --- User Management ---

// handleAdminListUsers handles GET /api/v1/admin/users.
func (s *HTTPServer) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	if s.userAuth == nil {
		s.jsonError(w, "user authentication not enabled", http.StatusNotImplemented)
		return
	}

	search := r.URL.Query().Get("search")
	role := r.URL.Query().Get("role")
	sortBy := r.URL.Query().Get("sort")
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	users, total, err := s.userAuth.ListUsers(r.Context(), search, role, sortBy, limit, offset)
	if err != nil {
		s.internalError(w, r, "list users", err)
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
		s.jsonError(w, "invalid role update", http.StatusBadRequest)
		return
	}

	s.recordAdminAudit(r, "user.role_change", "user", id, fmt.Sprintf(`{"role":"%s"}`, req.Role))
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
		s.jsonError(w, "not found", http.StatusNotFound)
		return
	}

	s.recordAdminAudit(r, "user.delete", "user", id, "")
	w.WriteHeader(http.StatusNoContent)
}

// handleAdminBulkUsers handles POST /api/v1/admin/users/bulk.
func (s *HTTPServer) handleAdminBulkUsers(w http.ResponseWriter, r *http.Request) {
	if s.userAuth == nil {
		s.jsonError(w, "user authentication not enabled", http.StatusNotImplemented)
		return
	}

	var req struct {
		Action string   `json:"action"`
		IDs    []string `json:"ids"`
		Role   string   `json:"role,omitempty"`
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
		case "delete":
			err = s.userAuth.DeleteUser(r.Context(), id)
		case "set_role":
			if req.Role == "" {
				s.jsonError(w, "role is required for set_role action", http.StatusBadRequest)
				return
			}
			_, err = s.userAuth.UpdateRole(r.Context(), id, req.Role)
		default:
			s.jsonError(w, "invalid action: must be delete or set_role", http.StatusBadRequest)
			return
		}
		if err != nil {
			errorCount++
		} else {
			successCount++
			s.recordAdminAudit(r, "user."+req.Action, "user", id, "")
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"success": successCount,
		"errors":  errorCount,
	})
}
