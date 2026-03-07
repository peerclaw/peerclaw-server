package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dashboard_dist
var dashboardFS embed.FS

// DashboardHandler serves the embedded SPA frontend.
// For any path that does not match a real file, it returns index.html (SPA fallback).
func DashboardHandler() http.Handler {
	sub, err := fs.Sub(dashboardFS, "dashboard_dist")
	if err != nil {
		panic("dashboard_dist embed failed: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip API and protocol endpoints — those are handled by other routes.
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/a2a") ||
			strings.HasPrefix(path, "/mcp") ||
			strings.HasPrefix(path, "/acp") ||
			strings.HasPrefix(path, "/.well-known/") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the requested file.
		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		if _, err := fs.Stat(sub, cleanPath); err == nil {
			// File exists — serve it with caching for assets.
			if strings.HasPrefix(cleanPath, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback — serve index.html for client-side routing.
		indexData, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(indexData)
	})
}
