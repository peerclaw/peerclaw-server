package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-core/envelope"
	coreprotocol "github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/identity"
	"github.com/peerclaw/peerclaw-server/internal/invocation"
)

type invokeRequest struct {
	Message  string            `json:"message"`
	Protocol string            `json:"protocol,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Stream   bool              `json:"stream,omitempty"`
}

type invokeResponse struct {
	ID       string `json:"id"`
	AgentID  string `json:"agent_id"`
	Response string `json:"response"`
	Protocol string `json:"protocol"`
	Duration int64  `json:"duration_ms"`
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

	// Get user ID if authenticated.
	userID, _ := identity.UserIDFromContext(r.Context())

	// Get IP.
	ipAddress := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ipAddress = fwd
	}

	// Rate limiting: keyed by IP for anonymous, by user ID for authenticated.
	if s.invokeRateLimiter != nil {
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

	s.jsonResponse(w, http.StatusOK, invokeResponse{
		ID:       invID,
		AgentID:  agentID,
		Response: respBody,
		Protocol: proto,
		Duration: duration,
	})
}

// handleInvokeSSE handles streaming invocation with SSE.
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

	// Send initial event.
	_, _ = fmt.Fprintf(w, "event: start\ndata: {\"id\":\"%s\",\"agent_id\":\"%s\"}\n\n", invID, agentID)
	flusher.Flush()

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

	if invokeErr != "" {
		_, _ = fmt.Fprintf(w, "event: error\ndata: {\"error\":\"%s\"}\n\n", invokeErr)
	} else {
		msgData, _ := json.Marshal(map[string]any{
			"content":  respBody,
			"protocol": proto,
		})
		_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(msgData))
	}
	flusher.Flush()

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
