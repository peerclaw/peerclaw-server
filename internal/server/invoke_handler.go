package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-core/envelope"
	coreprotocol "github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/identity"
	"github.com/peerclaw/peerclaw-server/internal/invocation"
)

type invokeRequest struct {
	Message   string            `json:"message"`
	Protocol  string            `json:"protocol,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Stream    bool              `json:"stream,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
}

type invokeResponse struct {
	ID        string `json:"id"`
	AgentID   string `json:"agent_id"`
	Response  string `json:"response"`
	Protocol  string `json:"protocol"`
	Duration  int64  `json:"duration_ms"`
	SessionID string `json:"session_id,omitempty"`
}

// handleInvoke handles POST /api/v1/invoke/{agent_id}.
func (s *HTTPServer) handleInvoke(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agent_id")
	if agentID == "" {
		s.jsonError(w, "agent_id is required", http.StatusBadRequest)
		return
	}

	var req invokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		s.jsonError(w, "message is required", http.StatusBadRequest)
		return
	}

	// Extract caller identity.
	callerAgentID, isAgent := identity.AgentIDFromContext(r.Context())
	userID, _ := identity.UserIDFromContext(r.Context())

	// Check contacts whitelist for agent-to-agent invocations.
	if isAgent && s.contacts != nil {
		allowed, err := s.contacts.IsAllowed(r.Context(), callerAgentID, agentID)
		if err != nil {
			s.jsonError(w, "failed to check contacts", http.StatusInternalServerError)
			return
		}
		if !allowed {
			s.jsonError(w, "not in destination agent's contact list", http.StatusForbidden)
			return
		}
	}

	// User access check: require playground_enabled (P2 adds ACL fallback).
	if !isAgent {
		if userID == "" {
			s.jsonError(w, "authentication required", http.StatusUnauthorized)
			return
		}
		flags, err := s.registry.GetAccessFlags(r.Context(), agentID)
		if err != nil {
			s.jsonError(w, "agent not found", http.StatusNotFound)
			return
		}
		if !flags.PlaygroundEnabled {
			// Check user ACL (P2).
			if s.useracl != nil {
				allowed, err := s.useracl.IsAllowed(r.Context(), agentID, userID)
				if err != nil {
					s.jsonError(w, "failed to check access", http.StatusInternalServerError)
					return
				}
				if !allowed {
					s.jsonError(w, "access denied: request access from the agent provider", http.StatusForbidden)
					return
				}
			} else {
				s.jsonError(w, "access denied: agent does not allow playground access", http.StatusForbidden)
				return
			}
		}
		// Set userID for later use (already in context from middleware).
	}

	// Look up agent.
	card, err := s.registry.GetAgent(r.Context(), agentID)
	if err != nil {
		s.jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	// Determine protocol.
	proto := req.Protocol
	if proto == "" && len(card.Protocols) > 0 {
		proto = string(card.Protocols[0])
	}
	if proto == "" {
		proto = "a2a"
	}

	// Get IP.
	ipAddress := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ipAddress = fwd
	}

	// Rate limiting: skip for agent-to-agent (controlled by whitelist).
	// For user invocations: keyed by user ID.
	if !isAgent && s.invokeRateLimiter != nil {
		key := ipAddress
		if userID != "" {
			key = "user:" + userID
		}
		if !s.invokeRateLimiter.GetLimiter(key).Allow() {
			s.jsonError(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
	}

	// Build envelope.
	env := envelope.New("gateway", agentID, coreprotocol.Protocol(proto), []byte(req.Message))
	for k, v := range req.Metadata {
		env.WithMetadata(k, v)
	}
	if req.SessionID != "" {
		env.WithSessionID(req.SessionID)
	}

	start := time.Now()

	if req.Stream {
		s.handleInvokeSSE(w, r, env, agentID, userID, proto, ipAddress, start)
		return
	}

	// Synchronous invocation via bridge.
	var respBody string
	var statusCode int
	var invokeErr string

	if s.bridges != nil {
		err := s.bridges.Send(r.Context(), env)
		if err != nil {
			invokeErr = err.Error()
			statusCode = 502
		} else {
			respBody = "Message delivered to agent " + agentID
			statusCode = 200
		}
	} else {
		invokeErr = "bridge not available"
		statusCode = 503
	}

	duration := time.Since(start).Milliseconds()

	// Record invocation.
	invID := uuid.New().String()
	if s.invocation != nil {
		_ = s.invocation.Record(r.Context(), &invocation.InvocationRecord{
			ID:           invID,
			AgentID:      agentID,
			UserID:       userID,
			Protocol:     proto,
			RequestBody:  req.Message,
			ResponseBody: respBody,
			StatusCode:   statusCode,
			DurationMs:   duration,
			Error:        invokeErr,
			IPAddress:    ipAddress,
			CreatedAt:    time.Now().UTC(),
		})
	}

	// Record reputation event.
	if s.reputation != nil {
		if invokeErr == "" {
			_ = s.reputation.RecordEvent(r.Context(), agentID, "bridge_success", "")
		} else {
			_ = s.reputation.RecordEvent(r.Context(), agentID, "bridge_error", invokeErr)
		}
	}

	if invokeErr != "" {
		s.jsonError(w, invokeErr, statusCode)
		return
	}

	// Use the session ID from the request, or generate one for the client.
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	s.jsonResponse(w, http.StatusOK, invokeResponse{
		ID:        invID,
		AgentID:   agentID,
		Response:  respBody,
		Protocol:  proto,
		Duration:  duration,
		SessionID: sessionID,
	})
}

// handleInvokeSSE handles streaming invocation with SSE.
// When the bridge supports streaming, chunks are piped to the client in real time.
func (s *HTTPServer) handleInvokeSSE(w http.ResponseWriter, r *http.Request, env *envelope.Envelope, agentID, userID, proto, ipAddress string, start time.Time) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.jsonError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	invID := uuid.New().String()
	sessionID := env.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Send initial event.
	_, _ = fmt.Fprintf(w, "event: start\ndata: {\"id\":\"%s\",\"agent_id\":\"%s\",\"session_id\":\"%s\"}\n\n", invID, agentID, sessionID)
	flusher.Flush()

	var respBody string
	var statusCode int
	var invokeErr string

	if s.bridges != nil {
		chunks, err := s.bridges.SendStream(r.Context(), env)
		if err != nil {
			invokeErr = err.Error()
			statusCode = 502
		} else {
			statusCode = 200
			var sb strings.Builder
			for chunk := range chunks {
				if chunk.Error != nil {
					invokeErr = chunk.Error.Error()
					statusCode = 502
					_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n",
						escapeSSEData(invokeErr))
					flusher.Flush()
					break
				}
				if chunk.Data != "" {
					sb.WriteString(chunk.Data)
					msgData, _ := json.Marshal(map[string]any{
						"content":  chunk.Data,
						"protocol": proto,
					})
					_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(msgData))
					flusher.Flush()
				}
				if chunk.Done {
					break
				}
			}
			if invokeErr == "" {
				respBody = sb.String()
			}
		}
	} else {
		invokeErr = "bridge not available"
		statusCode = 503
	}

	duration := time.Since(start).Milliseconds()

	// Send error event only if not already sent by the streaming loop.
	if invokeErr != "" && statusCode >= 500 && s.bridges == nil {
		_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", escapeSSEData(invokeErr))
		flusher.Flush()
	}

	_, _ = fmt.Fprintf(w, "event: done\ndata: {\"duration_ms\":%d}\n\n", duration)
	flusher.Flush()

	// Record invocation.
	if s.invocation != nil {
		_ = s.invocation.Record(r.Context(), &invocation.InvocationRecord{
			ID:           invID,
			AgentID:      agentID,
			UserID:       userID,
			Protocol:     proto,
			RequestBody:  string(env.Payload),
			ResponseBody: respBody,
			StatusCode:   statusCode,
			DurationMs:   duration,
			Error:        invokeErr,
			IPAddress:    ipAddress,
			CreatedAt:    time.Now().UTC(),
		})
	}

	if s.reputation != nil {
		if invokeErr == "" {
			_ = s.reputation.RecordEvent(r.Context(), agentID, "bridge_success", "")
		} else {
			_ = s.reputation.RecordEvent(r.Context(), agentID, "bridge_error", invokeErr)
		}
	}
}

// escapeSSEData escapes a string for use in SSE data fields.
func escapeSSEData(s string) string {
	data, _ := json.Marshal(map[string]string{"error": s})
	return string(data)
}

// handleListInvocations handles GET /api/v1/invocations.
func (s *HTTPServer) handleListInvocations(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

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

	records, total, err := s.invocation.ListByUser(r.Context(), userID, limit, offset)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if records == nil {
		records = []invocation.InvocationRecord{}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"invocations": records,
		"total":       total,
	})
}

// handleGetInvocation handles GET /api/v1/invocations/{id}.
func (s *HTTPServer) handleGetInvocation(w http.ResponseWriter, r *http.Request) {
	userID, ok := identity.UserIDFromContext(r.Context())
	if !ok {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	invID := r.PathValue("id")
	record, err := s.invocation.GetByID(r.Context(), invID)
	if err != nil {
		s.jsonError(w, "invocation not found", http.StatusNotFound)
		return
	}

	if record.UserID != userID {
		s.jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	s.jsonResponse(w, http.StatusOK, record)
}
