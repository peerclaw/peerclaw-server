package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/peerclaw/peerclaw-server/internal/identity"
	"github.com/peerclaw/peerclaw-server/internal/review"
)

// handleListReviews handles GET /api/v1/directory/{id}/reviews.
func (s *HTTPServer) handleListReviews(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "reviews not available", http.StatusServiceUnavailable)
		return
	}

	agentID := r.PathValue("id")
	limit := 50
	offset := 0
	if ls := r.URL.Query().Get("limit"); ls != "" {
		if l, err := strconv.Atoi(ls); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	if os := r.URL.Query().Get("offset"); os != "" {
		if o, err := strconv.Atoi(os); err == nil && o >= 0 {
			offset = o
		}
	}

	reviews, total, err := s.reviewService.ListReviews(r.Context(), agentID, limit, offset)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if reviews == nil {
		reviews = []review.Review{}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"reviews": reviews,
		"total":   total,
	})
}

// handleGetReviewSummary handles GET /api/v1/directory/{id}/reviews/summary.
func (s *HTTPServer) handleGetReviewSummary(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "reviews not available", http.StatusServiceUnavailable)
		return
	}

	agentID := r.PathValue("id")
	summary, err := s.reviewService.GetSummary(r.Context(), agentID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, summary)
}

// handleSubmitReview handles POST /api/v1/directory/{id}/reviews.
func (s *HTTPServer) handleSubmitReview(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "reviews not available", http.StatusServiceUnavailable)
		return
	}

	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	agentID := r.PathValue("id")

	var req struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	rev, err := s.reviewService.SubmitReview(r.Context(), agentID, userID, req.Rating, req.Comment)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusCreated, rev)
}

// handleDeleteReview handles DELETE /api/v1/directory/{id}/reviews.
func (s *HTTPServer) handleDeleteReview(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "reviews not available", http.StatusServiceUnavailable)
		return
	}

	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	agentID := r.PathValue("id")
	if err := s.reviewService.DeleteReview(r.Context(), agentID, userID); err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListCategories handles GET /api/v1/categories.
func (s *HTTPServer) handleListCategories(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "reviews not available", http.StatusServiceUnavailable)
		return
	}

	categories, err := s.reviewService.ListCategories(r.Context())
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if categories == nil {
		categories = []review.Category{}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"categories": categories})
}

// handleSubmitReport handles POST /api/v1/reports.
func (s *HTTPServer) handleSubmitReport(w http.ResponseWriter, r *http.Request) {
	if s.reviewService == nil {
		s.jsonError(w, "reviews not available", http.StatusServiceUnavailable)
		return
	}

	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		TargetType string `json:"target_type"` // "agent" or "review"
		TargetID   string `json:"target_id"`
		Reason     string `json:"reason"`
		Details    string `json:"details"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.reviewService.SubmitReport(r.Context(), userID, req.TargetType, req.TargetID, req.Reason, req.Details); err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]string{"status": "reported"})
}
