package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/envelope"
	coreprotocol "github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/bridge/acp"
	"github.com/peerclaw/peerclaw-server/internal/invocation"
)

// acpBridgeRuns holds in-memory ACP run state for the bridge.
type acpBridgeRuns struct {
	runs     sync.Map
	runCount atomic.Int64
}

func (s *HTTPServer) registerACPBridgeRoutes() {
	s.acpRuns = &acpBridgeRuns{}
	s.mux.HandleFunc("POST /acp/{agent_id}/runs", s.handleACPBridgeCreateRun)
	s.mux.HandleFunc("GET /acp/{agent_id}/runs/{run_id}", s.handleACPBridgeGetRun)
	s.mux.HandleFunc("POST /acp/{agent_id}/runs/{run_id}/cancel", s.handleACPBridgeCancelRun)
	s.mux.HandleFunc("GET /acp/{agent_id}/agents", s.handleACPBridgeAgentManifest)
	s.mux.HandleFunc("GET /acp/{agent_id}/ping", s.handleACPBridgePing)

	// Start background cleanup.
	go s.acpBridgeCleanup(time.Hour)
}

// handleACPBridgeCreateRun handles POST /acp/{agent_id}/runs.
func (s *HTTPServer) handleACPBridgeCreateRun(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agent_id")
	if agentID == "" {
		writeACPBridgeError(w, http.StatusBadRequest, "missing agent_id")
		return
	}

	var req acp.CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeACPBridgeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if len(req.Input) == 0 {
		writeACPBridgeError(w, http.StatusBadRequest, "input is required")
		return
	}

	// Resolve agent.
	card, err := s.registry.GetAgent(r.Context(), agentID)
	if err != nil {
		writeACPBridgeError(w, http.StatusNotFound, "agent not found: "+agentID)
		return
	}

	// Access control: treat external ACP clients as anonymous users.
	flags, err := s.registry.GetAccessFlags(r.Context(), agentID)
	if err != nil {
		writeACPBridgeError(w, http.StatusInternalServerError, "failed to check access flags")
		return
	}
	if !flags.PlaygroundEnabled {
		writeACPBridgeError(w, http.StatusForbidden, "access denied: agent does not allow external ACP access")
		return
	}

	// Rate limiting by IP.
	if s.invokeRateLimiter != nil {
		ipAddress := BridgeClientIP(r)
		if !s.invokeRateLimiter.GetLimiter("acp:"+ipAddress).Allow() {
			writeACPBridgeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
	}

	// Force agent name from card.
	req.AgentName = card.Name

	// Create run.
	run := acp.NewRun(req)
	s.acpRuns.runs.Store(run.RunID, run)
	s.acpRuns.runCount.Add(1)

	// Determine protocol.
	proto := "acp"
	if len(card.Protocols) > 0 {
		proto = string(card.Protocols[0])
	}

	// Build envelope from input messages.
	payload := acpInputToPayload(req.Input)
	env := envelope.New("acp-bridge", agentID, coreprotocol.Protocol(proto), []byte(payload))
	env.WithSessionID(run.SessionID)
	env.WithMetadata("acp.run_id", run.RunID)
	env.WithMetadata("acp.session_id", run.SessionID)

	start := time.Now()
	ipAddress := BridgeClientIP(r)

	mode := req.Mode
	if mode == "" {
		mode = "sync"
	}

	switch mode {
	case "stream":
		s.handleACPBridgeStream(w, r, run, env, card, proto, payload, start, ipAddress)
	case "async":
		s.handleACPBridgeAsync(w, r, run, env, card, proto, payload, start, ipAddress)
	default: // sync
		s.handleACPBridgeSync(w, r, run, env, card, proto, payload, start, ipAddress)
	}
}

// handleACPBridgeSync handles synchronous run execution.
func (s *HTTPServer) handleACPBridgeSync(w http.ResponseWriter, r *http.Request, run *acp.Run, env *envelope.Envelope, card *agentcard.Card, proto, payload string, start time.Time, ipAddress string) {
	if s.bridges == nil {
		run = updateACPRunState(run, acp.RunStatusFailed, "bridge not available")
		s.acpRuns.runs.Store(run.RunID, run)
		writeACPBridgeError(w, http.StatusBadGateway, "bridge not available")
		return
	}

	// Update run to in-progress.
	run = updateACPRunState(run, acp.RunStatusInProgress, "")
	s.acpRuns.runs.Store(run.RunID, run)

	chunks, err := s.bridges.SendStream(r.Context(), env)
	if err != nil {
		run = updateACPRunState(run, acp.RunStatusFailed, err.Error())
		s.acpRuns.runs.Store(run.RunID, run)
		s.recordACPBridgeInvocation(r.Context(), card.ID, proto, payload, "", 502, time.Since(start).Milliseconds(), err.Error(), ipAddress)
		writeACPBridgeJSON(w, http.StatusOK, run)
		return
	}

	var sb strings.Builder
	var invokeErr string
	for chunk := range chunks {
		if chunk.Error != nil {
			invokeErr = chunk.Error.Error()
			break
		}
		if chunk.Data != "" {
			sb.WriteString(chunk.Data)
		}
		if chunk.Done {
			break
		}
	}

	duration := time.Since(start).Milliseconds()
	respBody := sb.String()

	if invokeErr != "" {
		run = updateACPRunState(run, acp.RunStatusFailed, invokeErr)
		s.recordACPBridgeInvocation(r.Context(), card.ID, proto, payload, respBody, 502, duration, invokeErr, ipAddress)
	} else {
		cp := cloneACPRun(run)
		cp.Output = []acp.Message{
			{
				Role: "agent",
				Parts: []acp.MessagePart{
					{ContentType: "text/plain", Content: respBody},
				},
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			},
		}
		run = updateACPRunState(cp, acp.RunStatusCompleted, "")
		s.recordACPBridgeInvocation(r.Context(), card.ID, proto, payload, respBody, 200, duration, "", ipAddress)
	}
	s.acpRuns.runs.Store(run.RunID, run)

	// Record reputation.
	if s.reputation != nil {
		if invokeErr == "" {
			_ = s.reputation.RecordEvent(r.Context(), card.ID, "bridge_success", "")
		} else {
			_ = s.reputation.RecordEvent(r.Context(), card.ID, "bridge_error", invokeErr)
		}
	}

	writeACPBridgeJSON(w, http.StatusOK, run)
}

// handleACPBridgeStream handles SSE streaming run execution.
func (s *HTTPServer) handleACPBridgeStream(w http.ResponseWriter, r *http.Request, run *acp.Run, env *envelope.Envelope, card *agentcard.Card, proto, payload string, start time.Time, ipAddress string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeACPBridgeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Send initial in-progress state.
	run = updateACPRunState(run, acp.RunStatusInProgress, "")
	s.acpRuns.runs.Store(run.RunID, run)
	sendACPSSEEvent(w, flusher, run)

	if s.bridges == nil {
		run = updateACPRunState(run, acp.RunStatusFailed, "bridge not available")
		s.acpRuns.runs.Store(run.RunID, run)
		sendACPSSEEvent(w, flusher, run)
		return
	}

	chunks, err := s.bridges.SendStream(r.Context(), env)
	if err != nil {
		run = updateACPRunState(run, acp.RunStatusFailed, err.Error())
		s.acpRuns.runs.Store(run.RunID, run)
		sendACPSSEEvent(w, flusher, run)
		s.recordACPBridgeInvocation(r.Context(), card.ID, proto, payload, "", 502, time.Since(start).Milliseconds(), err.Error(), ipAddress)
		return
	}

	var sb strings.Builder
	var invokeErr string
	for chunk := range chunks {
		if chunk.Error != nil {
			invokeErr = chunk.Error.Error()
			break
		}
		if chunk.Data != "" {
			sb.WriteString(chunk.Data)
			// Send partial output update (clone before mutation).
			cp := cloneACPRun(run)
			cp.Output = []acp.Message{
				{
					Role: "agent",
					Parts: []acp.MessagePart{
						{ContentType: "text/plain", Content: sb.String()},
					},
				},
			}
			cp.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			s.acpRuns.runs.Store(cp.RunID, cp)
			run = cp
			sendACPSSEEvent(w, flusher, run)
		}
		if chunk.Done {
			break
		}
	}

	duration := time.Since(start).Milliseconds()
	respBody := sb.String()

	if invokeErr != "" {
		run = updateACPRunState(run, acp.RunStatusFailed, invokeErr)
		s.recordACPBridgeInvocation(r.Context(), card.ID, proto, payload, respBody, 502, duration, invokeErr, ipAddress)
	} else {
		cp := cloneACPRun(run)
		cp.Output = []acp.Message{
			{
				Role: "agent",
				Parts: []acp.MessagePart{
					{ContentType: "text/plain", Content: respBody},
				},
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			},
		}
		run = updateACPRunState(cp, acp.RunStatusCompleted, "")
		s.recordACPBridgeInvocation(r.Context(), card.ID, proto, payload, respBody, 200, duration, "", ipAddress)
	}
	s.acpRuns.runs.Store(run.RunID, run)

	// Send final state.
	sendACPSSEEvent(w, flusher, run)

	// Record reputation.
	if s.reputation != nil {
		if invokeErr == "" {
			_ = s.reputation.RecordEvent(r.Context(), card.ID, "bridge_success", "")
		} else {
			_ = s.reputation.RecordEvent(r.Context(), card.ID, "bridge_error", invokeErr)
		}
	}
}

// handleACPBridgeAsync handles async run execution.
func (s *HTTPServer) handleACPBridgeAsync(w http.ResponseWriter, r *http.Request, run *acp.Run, env *envelope.Envelope, card *agentcard.Card, proto, payload string, start time.Time, ipAddress string) {
	// Update run to in-progress.
	run = updateACPRunState(run, acp.RunStatusInProgress, "")
	s.acpRuns.runs.Store(run.RunID, run)

	// Return 202 immediately.
	writeACPBridgeJSON(w, http.StatusAccepted, run)

	// Execute in background.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if s.bridges == nil {
			run = updateACPRunState(run, acp.RunStatusFailed, "bridge not available")
			s.acpRuns.runs.Store(run.RunID, run)
			return
		}

		chunks, err := s.bridges.SendStream(ctx, env)
		if err != nil {
			run = updateACPRunState(run, acp.RunStatusFailed, err.Error())
			s.acpRuns.runs.Store(run.RunID, run)
			s.recordACPBridgeInvocation(ctx, card.ID, proto, payload, "", 502, time.Since(start).Milliseconds(), err.Error(), ipAddress)
			return
		}

		var sb strings.Builder
		var invokeErr string
		for chunk := range chunks {
			if chunk.Error != nil {
				invokeErr = chunk.Error.Error()
				break
			}
			if chunk.Data != "" {
				sb.WriteString(chunk.Data)
			}
			if chunk.Done {
				break
			}
		}

		duration := time.Since(start).Milliseconds()
		respBody := sb.String()

		if invokeErr != "" {
			run = updateACPRunState(run, acp.RunStatusFailed, invokeErr)
			s.recordACPBridgeInvocation(ctx, card.ID, proto, payload, respBody, 502, duration, invokeErr, ipAddress)
		} else {
			cp := cloneACPRun(run)
			cp.Output = []acp.Message{
				{
					Role: "agent",
					Parts: []acp.MessagePart{
						{ContentType: "text/plain", Content: respBody},
					},
					CreatedAt: time.Now().UTC().Format(time.RFC3339),
				},
			}
			run = updateACPRunState(cp, acp.RunStatusCompleted, "")
			s.recordACPBridgeInvocation(ctx, card.ID, proto, payload, respBody, 200, duration, "", ipAddress)
		}
		s.acpRuns.runs.Store(run.RunID, run)

		// Record reputation.
		if s.reputation != nil {
			if invokeErr == "" {
				_ = s.reputation.RecordEvent(ctx, card.ID, "bridge_success", "")
			} else {
				_ = s.reputation.RecordEvent(ctx, card.ID, "bridge_error", invokeErr)
			}
		}
	}()
}

// handleACPBridgeGetRun handles GET /acp/{agent_id}/runs/{run_id}.
func (s *HTTPServer) handleACPBridgeGetRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("run_id")
	if runID == "" {
		writeACPBridgeError(w, http.StatusBadRequest, "missing run_id")
		return
	}

	v, ok := s.acpRuns.runs.Load(runID)
	if !ok {
		writeACPBridgeError(w, http.StatusNotFound, "run not found: "+runID)
		return
	}

	writeACPBridgeJSON(w, http.StatusOK, v.(*acp.Run))
}

// handleACPBridgeCancelRun handles POST /acp/{agent_id}/runs/{run_id}/cancel.
func (s *HTTPServer) handleACPBridgeCancelRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("run_id")
	if runID == "" {
		writeACPBridgeError(w, http.StatusBadRequest, "missing run_id")
		return
	}

	v, ok := s.acpRuns.runs.Load(runID)
	if !ok {
		writeACPBridgeError(w, http.StatusNotFound, "run not found: "+runID)
		return
	}

	run := v.(*acp.Run)
	run = updateACPRunState(run, acp.RunStatusCancelled, "")
	s.acpRuns.runs.Store(run.RunID, run)

	writeACPBridgeJSON(w, http.StatusOK, run)
}

// handleACPBridgeAgentManifest handles GET /acp/{agent_id}/agents.
func (s *HTTPServer) handleACPBridgeAgentManifest(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agent_id")
	if agentID == "" {
		writeACPBridgeError(w, http.StatusBadRequest, "missing agent_id")
		return
	}

	card, err := s.registry.GetAgent(r.Context(), agentID)
	if err != nil {
		writeACPBridgeError(w, http.StatusNotFound, "agent not found: "+agentID)
		return
	}

	manifest := cardToACPManifest(card)
	writeACPBridgeJSON(w, http.StatusOK, manifest)
}

// handleACPBridgePing handles GET /acp/{agent_id}/ping.
func (s *HTTPServer) handleACPBridgePing(w http.ResponseWriter, r *http.Request) {
	writeACPBridgeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Helpers ---

// cardToACPManifest converts a PeerClaw Card to an ACP AgentManifest.
func cardToACPManifest(card *agentcard.Card) acp.AgentManifest {
	manifest := acp.AgentManifest{
		Name:               card.Name,
		Description:        card.Description,
		InputContentTypes:  []string{"text/plain"},
		OutputContentTypes: []string{"text/plain"},
	}

	var caps []acp.CapabilityDef
	for _, c := range card.Capabilities {
		caps = append(caps, acp.CapabilityDef{Name: c})
	}
	if len(caps) > 0 {
		manifest.Metadata.Capabilities = caps
	}

	if ext := card.PeerClaw; len(ext.Tags) > 0 {
		manifest.Metadata.Tags = ext.Tags
	}

	return manifest
}

// acpInputToPayload extracts text content from ACP input messages.
func acpInputToPayload(input []acp.Message) string {
	var parts []string
	for _, msg := range input {
		for _, p := range msg.Parts {
			if p.Content != "" {
				parts = append(parts, p.Content)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// cloneACPRun returns a shallow copy of the run with fresh Output slice.
func cloneACPRun(run *acp.Run) *acp.Run {
	cp := *run
	if run.Output != nil {
		cp.Output = make([]acp.Message, len(run.Output))
		copy(cp.Output, run.Output)
	}
	return &cp
}

// updateACPRunState creates a copy with updated status and returns it.
func updateACPRunState(run *acp.Run, status acp.RunStatus, errMsg string) *acp.Run {
	cp := cloneACPRun(run)
	now := time.Now().UTC().Format(time.RFC3339)
	cp.Status = status
	cp.UpdatedAt = now
	if errMsg != "" {
		cp.Error = &acp.RunError{
			Code:    string(status),
			Message: errMsg,
		}
	}
	return cp
}

// recordACPBridgeInvocation records an invocation to the invocation service.
func (s *HTTPServer) recordACPBridgeInvocation(ctx context.Context, agentID, proto, reqBody, respBody string, statusCode int, durationMs int64, invokeErr, ipAddress string) {
	if s.invocation == nil {
		return
	}
	_ = s.invocation.Record(ctx, &invocation.InvocationRecord{
		ID:           uuid.New().String(),
		AgentID:      agentID,
		UserID:       "acp-bridge",
		Protocol:     proto,
		RequestBody:  reqBody,
		ResponseBody: respBody,
		StatusCode:   statusCode,
		DurationMs:   durationMs,
		Error:        invokeErr,
		IPAddress:    ipAddress,
		CreatedAt:    time.Now().UTC(),
	})
}

// sendACPSSEEvent sends a run update as an SSE event.
func sendACPSSEEvent(w http.ResponseWriter, flusher http.Flusher, run *acp.Run) {
	data, err := json.Marshal(run)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: run_update\ndata: %s\n\n", string(data))
	flusher.Flush()
}


func writeACPBridgeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeACPBridgeError(w http.ResponseWriter, status int, message string) {
	writeACPBridgeJSON(w, status, map[string]string{"error": message})
}

// acpBridgeCleanup periodically removes expired runs.
func (s *HTTPServer) acpBridgeCleanup(maxAge time.Duration) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().UTC()
		s.acpRuns.runs.Range(func(key, value any) bool {
			run := value.(*acp.Run)
			created, err := time.Parse(time.RFC3339, run.CreatedAt)
			if err != nil {
				return true
			}
			if now.Sub(created) > maxAge {
				s.acpRuns.runs.Delete(key)
				s.acpRuns.runCount.Add(-1)
			}
			return true
		})
	}
}
