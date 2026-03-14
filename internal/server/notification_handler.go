package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/coder/websocket"
	"github.com/peerclaw/peerclaw-server/internal/identity"
	"github.com/peerclaw/peerclaw-server/internal/notification"
)

func (s *HTTPServer) registerNotificationRoutes() {
	if s.notificationSvc == nil {
		return
	}

	s.mux.Handle("GET /api/v1/provider/notifications", s.wrapUserAuth(s.handleListNotifications))
	s.mux.Handle("GET /api/v1/provider/notifications/count", s.wrapUserAuth(s.handleNotificationCount))
	s.mux.Handle("PUT /api/v1/provider/notifications/{id}/read", s.wrapUserAuth(s.handleMarkNotificationRead))
	s.mux.Handle("PUT /api/v1/provider/notifications/read-all", s.wrapUserAuth(s.handleMarkAllNotificationsRead))
	s.mux.Handle("GET /api/v1/provider/notifications/ws", s.wrapUserAuth(s.handleNotificationWS))
}

// handleListNotifications handles GET /api/v1/provider/notifications.
func (s *HTTPServer) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 20
	offset := 0
	unreadOnly := false

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	if r.URL.Query().Get("unread_only") == "true" {
		unreadOnly = true
	}

	notifications, total, err := s.notificationSvc.List(r.Context(), userID, unreadOnly, limit, offset)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if notifications == nil {
		notifications = []notification.Notification{}
	}

	unreadCount, _ := s.notificationSvc.CountUnread(r.Context(), userID)

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"notifications": notifications,
		"total":         total,
		"unread_count":  unreadCount,
	})
}

// handleNotificationCount handles GET /api/v1/provider/notifications/count.
func (s *HTTPServer) handleNotificationCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	count, err := s.notificationSvc.CountUnread(r.Context(), userID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"unread_count": count})
}

// handleMarkNotificationRead handles PUT /api/v1/provider/notifications/{id}/read.
func (s *HTTPServer) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	notifID := r.PathValue("id")
	if err := s.notificationSvc.MarkRead(r.Context(), notifID, userID); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleMarkAllNotificationsRead handles PUT /api/v1/provider/notifications/read-all.
func (s *HTTPServer) handleMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := s.notificationSvc.MarkAllRead(r.Context(), userID); err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleNotificationWS handles GET /api/v1/provider/notifications/ws.
func (s *HTTPServer) handleNotificationWS(w http.ResponseWriter, r *http.Request) {
	if s.notifHub == nil {
		s.jsonError(w, "notifications not enabled", http.StatusNotImplemented)
		return
	}

	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // CORS handled by middleware
	})
	if err != nil {
		s.logger.Debug("notification ws accept failed", "error", err)
		return
	}

	s.notifHub.AddConn(userID, conn)
	defer s.notifHub.RemoveConn(userID, conn)

	s.logger.Debug("notification ws connected", "user_id", userID)

	// Read loop to keep the connection alive (detect disconnects).
	for {
		_, _, err := conn.Read(r.Context())
		if err != nil {
			break
		}
	}
}

// emitNotification creates a notification for the owner of the given agent.
func (s *HTTPServer) emitNotification(ctx context.Context, agentID string, ntype notification.NotificationType, severity notification.Severity, title, body string, metadata map[string]string) {
	if s.notificationSvc == nil {
		return
	}
	card, err := s.registry.GetAgent(ctx, agentID)
	if err != nil {
		return
	}
	if card.Metadata == nil {
		return
	}
	ownerUserID, ok := card.Metadata["owner_user_id"]
	if !ok || ownerUserID == "" {
		return
	}
	s.emitNotificationToUser(ctx, ownerUserID, agentID, ntype, severity, title, body, metadata)
}

// emitNotificationToUser creates a notification for a specific user.
func (s *HTTPServer) emitNotificationToUser(ctx context.Context, userID, agentID string, ntype notification.NotificationType, severity notification.Severity, title, body string, metadata map[string]string) {
	if s.notificationSvc == nil {
		return
	}
	_, err := s.notificationSvc.Notify(ctx, userID, agentID, ntype, severity, title, body, metadata)
	if err != nil {
		s.logger.Debug("failed to emit notification", "user_id", userID, "type", ntype, "error", err)
	}
}
