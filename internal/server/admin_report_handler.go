package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/peerclaw/peerclaw-server/internal/review"
)

// --- Report Moderation ---

// handleAdminListReports handles GET /api/v1/admin/reports.
func (s *HTTPServer) handleAdminListReports(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "review service not enabled", http.StatusNotImplemented)
		return
	}

	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	sortBy := r.URL.Query().Get("sort")
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	reports, total, err := s.reviewService.ListReports(r.Context(), status, search, sortBy, limit, offset)
	if err != nil {
		s.internalError(w, r, "list reports", err)
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
		s.jsonError(w, "invalid report status update", http.StatusBadRequest)
		return
	}

	s.recordAdminAudit(r, "report.update", "report", id, fmt.Sprintf(`{"status":"%s"}`, req.Status))
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
		s.jsonError(w, "not found", http.StatusNotFound)
		return
	}

	s.recordAdminAudit(r, "report.delete", "report", id, "")
	w.WriteHeader(http.StatusNoContent)
}

// handleAdminBulkReports handles POST /api/v1/admin/reports/bulk.
func (s *HTTPServer) handleAdminBulkReports(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "review service not enabled", http.StatusNotImplemented)
		return
	}

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
		case "review":
			err = s.reviewService.UpdateReportStatus(r.Context(), id, "reviewed")
		case "dismiss":
			err = s.reviewService.UpdateReportStatus(r.Context(), id, "dismissed")
		case "action":
			err = s.reviewService.UpdateReportStatus(r.Context(), id, "actioned")
		case "delete":
			err = s.reviewService.DeleteReport(r.Context(), id)
		default:
			s.jsonError(w, "invalid action: must be review, dismiss, action, or delete", http.StatusBadRequest)
			return
		}
		if err != nil {
			errorCount++
		} else {
			successCount++
			s.recordAdminAudit(r, "report."+req.Action, "report", id, "")
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"success": successCount,
		"errors":  errorCount,
	})
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
		s.jsonError(w, "failed to create category", http.StatusBadRequest)
		return
	}

	s.recordAdminAudit(r, "category.create", "category", cat.ID, "")
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
		s.jsonError(w, "not found", http.StatusNotFound)
		return
	}

	s.recordAdminAudit(r, "category.update", "category", id, "")
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
		s.jsonError(w, "not found", http.StatusNotFound)
		return
	}

	s.recordAdminAudit(r, "category.delete", "category", id, "")
	w.WriteHeader(http.StatusNoContent)
}
