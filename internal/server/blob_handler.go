package server

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-server/internal/identity"
)

// handleBlobUpload handles POST /api/v1/blobs.
// Requires JWT authentication. Stores the uploaded file and returns blob metadata.
func (s *HTTPServer) handleBlobUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Enforce max file size at the HTTP level.
	maxSize := s.blob.MaxFileSize()
	r.Body = http.MaxBytesReader(w, r.Body, maxSize+1024) // small margin for headers

	// Parse multipart form (max memory = 32MB, rest goes to temp files).
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		s.jsonError(w, "file too large or invalid multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.jsonError(w, "missing 'file' field in multipart form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := header.Filename
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	// Sanitize content type.
	if mt, _, err := mime.ParseMediaType(contentType); err == nil {
		contentType = mt
	}

	result, err := s.blob.Upload(r.Context(), userID, filename, contentType, file)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "quota exceeded") || strings.Contains(msg, "file too large") {
			s.jsonError(w, msg, http.StatusRequestEntityTooLarge)
			return
		}
		s.jsonError(w, msg, http.StatusInternalServerError)
		return
	}

	// Set download URL.
	result.DownloadURL = fmt.Sprintf("/api/v1/blobs/%s", result.ID)

	s.jsonResponse(w, http.StatusCreated, result)
}

// handleBlobDownload handles GET /api/v1/blobs/{id}.
// Public endpoint — anyone with the blob ID can download.
func (s *HTTPServer) handleBlobDownload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		s.jsonError(w, "missing blob id", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(id); err != nil {
		s.jsonError(w, "invalid blob id", http.StatusBadRequest)
		return
	}

	meta, rc, err := s.blob.Download(r.Context(), id)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "not found") {
			s.jsonError(w, "blob not found", http.StatusNotFound)
			return
		}
		if strings.Contains(msg, "expired") {
			s.jsonError(w, "blob expired", http.StatusGone)
			return
		}
		s.jsonError(w, msg, http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(meta.Size, 10))
	if meta.Filename != "" {
		w.Header().Set("Content-Disposition",
			fmt.Sprintf("attachment; filename=%q", meta.Filename))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}
