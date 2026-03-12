package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/envelope"
	coreprotocol "github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/bridge/a2a"
	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
)

// Default bridge cleanup configuration.
const (
	defaultBridgeCleanupInterval = 10 * time.Minute
	defaultBridgeMaxAge          = time.Hour
)

// a2aBridgeTasks holds in-memory A2A task state for the bridge.
type a2aBridgeTasks struct {
	tasks     sync.Map
	taskCount atomic.Int64

	// pushConfigs stores push notification configs keyed by task ID.
	pushConfigs sync.Map // taskID → *a2a.PushNotificationConfig

	// creatorIPs tracks the IP address that created each task for authorization.
	creatorIPs sync.Map // taskID → string (IP address)
}

func (s *HTTPServer) registerA2ABridgeRoutes(ctx context.Context) {
	s.a2aTasks = &a2aBridgeTasks{}
	bridgeAuth := s.bridgeAuthMiddleware()
	s.mux.Handle("POST /a2a/{agent_id}", bridgeAuth(http.HandlerFunc(s.handleA2ABridgeMessages)))
	s.mux.HandleFunc("GET /a2a/{agent_id}/.well-known/agent.json", s.handleA2ABridgeAgentCard)
	s.mux.HandleFunc("GET /a2a/{agent_id}/tasks/{task_id}", s.handleA2ABridgeGetTaskREST)

	// Start background cleanup (stops when ctx is cancelled).
	go s.a2aBridgeCleanup(ctx, defaultBridgeCleanupInterval, defaultBridgeMaxAge)
}

// handleA2ABridgeMessages handles POST /a2a/{agent_id} — JSON-RPC dispatch.
func (s *HTTPServer) handleA2ABridgeMessages(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agent_id")
	if agentID == "" {
		writeA2ABridgeError(w, nil, jsonrpc.CodeInvalidParams, "missing agent_id")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		writeA2ABridgeError(w, nil, jsonrpc.CodeParseError, "failed to read body")
		return
	}

	parsed, err := jsonrpc.ParseMessage(body)
	if err != nil {
		writeA2ABridgeError(w, nil, jsonrpc.CodeParseError, "invalid JSON-RPC: "+err.Error())
		return
	}

	if parsed.Kind == jsonrpc.KindNotification {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if parsed.Kind != jsonrpc.KindRequest {
		writeA2ABridgeError(w, nil, jsonrpc.CodeInvalidRequest, "expected JSON-RPC request")
		return
	}

	req := parsed.Request

	switch req.Method {
	case "message/send":
		// Check Accept header for SSE streaming preference.
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			s.handleA2ABridgeSendSubscribe(w, r, req, agentID)
		} else {
			s.handleA2ABridgeSendMessage(w, r, req, agentID)
		}
	case "message/send/subscribe":
		s.handleA2ABridgeSendSubscribe(w, r, req, agentID)
	case "tasks/get":
		s.handleA2ABridgeGetTask(w, r, req)
	case "tasks/cancel":
		s.handleA2ABridgeCancelTask(w, r, req)
	case "tasks/pushNotification/set":
		s.handleA2ABridgePushNotification(w, req, "set")
	case "tasks/pushNotification/get":
		s.handleA2ABridgePushNotification(w, req, "get")
	default:
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeMethodNotFound, "unknown method: "+req.Method)
	}
}

// handleA2ABridgeSendMessage handles synchronous message/send.
func (s *HTTPServer) handleA2ABridgeSendMessage(w http.ResponseWriter, r *http.Request, req *jsonrpc.Request, agentID string) {
	var params a2a.SendMessageParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	// Resolve agent.
	card, err := s.registry.GetAgent(r.Context(), agentID)
	if err != nil {
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "agent not found: "+agentID)
		return
	}

	// Access control: treat external A2A clients as anonymous users.
	flags, err := s.registry.GetAccessFlags(r.Context(), agentID)
	if err != nil {
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInternalError, "failed to check access flags")
		return
	}
	if !bridgeAccessAllowed(r, flags.PlaygroundEnabled) {
		writeA2ABridgeError(w, req.ID, -32001, "access denied: authentication required or enable playground access")
		return
	}

	// Rate limiting by IP.
	if s.invokeRateLimiter != nil {
		ipAddress := BridgeClientIP(r)
		if !s.invokeRateLimiter.GetLimiter("a2a:"+ipAddress).Allow() {
			writeA2ABridgeError(w, req.ID, -32002, "rate limit exceeded")
			return
		}
	}

	// Reject if too many in-flight tasks.
	const maxA2ATasks = 10000
	if s.a2aTasks.taskCount.Load() >= maxA2ATasks {
		writeA2ABridgeError(w, req.ID, -32003, "too many in-flight tasks")
		return
	}

	// Create task.
	contextID := params.Message.ContextID
	if contextID == "" {
		contextID = uuid.New().String()
	}
	task := a2a.NewTask(contextID, params.Message)
	s.a2aTasks.tasks.Store(task.ID, task)
	s.a2aTasks.taskCount.Add(1)

	ipAddress := BridgeClientIP(r)
	s.a2aTasks.creatorIPs.Store(task.ID, ipAddress)

	// Determine protocol.
	proto := "a2a"
	if len(card.Protocols) > 0 {
		proto = string(card.Protocols[0])
	}

	// Build envelope.
	var textParts []string
	for _, p := range params.Message.Parts {
		if p.Text != "" {
			textParts = append(textParts, p.Text)
		}
	}
	payload := strings.Join(textParts, "\n")

	env := envelope.New("a2a-bridge", agentID, coreprotocol.Protocol(proto), []byte(payload))
	env.WithSessionID(contextID)
	env.WithMetadata("a2a.task_id", task.ID)
	env.WithMetadata("a2a.context_id", contextID)

	start := time.Now()

	// Synchronous: collect all chunks from stream.
	if s.bridges == nil {
		task = s.updateTaskState(task, a2a.TaskStateFailed, "bridge not available")
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInternalError, "bridge not available")
		return
	}

	// Update task to working.
	task = s.updateTaskState(task, a2a.TaskStateWorking, "")

	chunks, err := s.bridges.SendStream(r.Context(), env)
	if err != nil {
		task = s.updateTaskState(task, a2a.TaskStateFailed, err.Error())
		writeA2ABridgeResult(w, req.ID, task)
		s.recordA2ABridgeInvocation(r.Context(), agentID, proto, payload, "", 502, time.Since(start).Milliseconds(), err.Error(), ipAddress)
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
		task = s.updateTaskState(task, a2a.TaskStateFailed, invokeErr)
		s.recordA2ABridgeInvocation(r.Context(), agentID, proto, payload, respBody, 502, duration, invokeErr, ipAddress)
	} else {
		// Add response as artifact (clone before mutation).
		cp := cloneA2ATask(task)
		cp.Artifacts = append(cp.Artifacts, a2a.Artifact{
			ID: uuid.New().String(),
			Parts: []a2a.Part{
				{Text: respBody},
			},
		})
		task = s.updateTaskState(cp, a2a.TaskStateCompleted, "")
		s.recordA2ABridgeInvocation(r.Context(), agentID, proto, payload, respBody, 200, duration, "", ipAddress)
	}

	// Record reputation.
	if s.reputation != nil {
		if invokeErr == "" {
			if err := s.reputation.RecordEvent(r.Context(), agentID, "bridge_success", ""); err != nil {
				s.logger.Debug("failed to record reputation event", "agent_id", agentID, "error", err)
			}
		} else {
			if err := s.reputation.RecordEvent(r.Context(), agentID, "bridge_error", invokeErr); err != nil {
				s.logger.Debug("failed to record reputation event", "agent_id", agentID, "error", err)
			}
		}
	}

	writeA2ABridgeResult(w, req.ID, task)
}

// handleA2ABridgeSendSubscribe handles SSE streaming for message/send.
func (s *HTTPServer) handleA2ABridgeSendSubscribe(w http.ResponseWriter, r *http.Request, req *jsonrpc.Request, agentID string) {
	var params a2a.SendMessageParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	// Resolve agent.
	card, err := s.registry.GetAgent(r.Context(), agentID)
	if err != nil {
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "agent not found: "+agentID)
		return
	}

	// Access control.
	flags, err := s.registry.GetAccessFlags(r.Context(), agentID)
	if err != nil {
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInternalError, "failed to check access flags")
		return
	}
	if !bridgeAccessAllowed(r, flags.PlaygroundEnabled) {
		writeA2ABridgeError(w, req.ID, -32001, "access denied: authentication required or enable playground access")
		return
	}

	// Rate limiting.
	if s.invokeRateLimiter != nil {
		ipAddress := BridgeClientIP(r)
		if !s.invokeRateLimiter.GetLimiter("a2a:"+ipAddress).Allow() {
			writeA2ABridgeError(w, req.ID, -32002, "rate limit exceeded")
			return
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInternalError, "streaming not supported")
		return
	}

	// Create task.
	contextID := params.Message.ContextID
	if contextID == "" {
		contextID = uuid.New().String()
	}
	task := a2a.NewTask(contextID, params.Message)
	s.a2aTasks.tasks.Store(task.ID, task)
	s.a2aTasks.taskCount.Add(1)

	ipAddress := BridgeClientIP(r)
	s.a2aTasks.creatorIPs.Store(task.ID, ipAddress)

	// Determine protocol.
	proto := "a2a"
	if len(card.Protocols) > 0 {
		proto = string(card.Protocols[0])
	}

	// Build envelope.
	var textParts []string
	for _, p := range params.Message.Parts {
		if p.Text != "" {
			textParts = append(textParts, p.Text)
		}
	}
	payload := strings.Join(textParts, "\n")

	env := envelope.New("a2a-bridge", agentID, coreprotocol.Protocol(proto), []byte(payload))
	env.WithSessionID(contextID)
	env.WithMetadata("a2a.task_id", task.ID)
	env.WithMetadata("a2a.context_id", contextID)

	start := time.Now()

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Send initial working state.
	task = s.updateTaskState(task, a2a.TaskStateWorking, "")
	sendA2ASSEEvent(w, flusher, req.ID, task)

	if s.bridges == nil {
		task = s.updateTaskState(task, a2a.TaskStateFailed, "bridge not available")
		sendA2ASSEEvent(w, flusher, req.ID, task)
		return
	}

	chunks, err := s.bridges.SendStream(r.Context(), env)
	if err != nil {
		task = s.updateTaskState(task, a2a.TaskStateFailed, err.Error())
		sendA2ASSEEvent(w, flusher, req.ID, task)
		s.recordA2ABridgeInvocation(r.Context(), agentID, proto, payload, "", 502, time.Since(start).Milliseconds(), err.Error(), ipAddress)
		return
	}

	var sb strings.Builder
	var invokeErr string
	artifactIndex := 0
	for chunk := range chunks {
		if chunk.Error != nil {
			invokeErr = chunk.Error.Error()
			break
		}
		if chunk.Data != "" {
			sb.WriteString(chunk.Data)
			// Send artifact update event (clone before mutation).
			cp := cloneA2ATask(task)
			cp.Artifacts = []a2a.Artifact{
				{
					ID: fmt.Sprintf("artifact-%d", artifactIndex),
					Parts: []a2a.Part{
						{Text: sb.String()},
					},
				},
			}
			cp.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			s.a2aTasks.tasks.Store(cp.ID, cp)
			task = cp
			sendA2ASSEEvent(w, flusher, req.ID, task)
		}
		if chunk.Done {
			break
		}
	}

	duration := time.Since(start).Milliseconds()
	respBody := sb.String()

	if invokeErr != "" {
		task = s.updateTaskState(task, a2a.TaskStateFailed, invokeErr)
		s.recordA2ABridgeInvocation(r.Context(), agentID, proto, payload, respBody, 502, duration, invokeErr, ipAddress)
	} else {
		cp := cloneA2ATask(task)
		cp.Artifacts = []a2a.Artifact{
			{
				ID: "artifact-final",
				Parts: []a2a.Part{
					{Text: respBody},
				},
			},
		}
		task = s.updateTaskState(cp, a2a.TaskStateCompleted, "")
		s.recordA2ABridgeInvocation(r.Context(), agentID, proto, payload, respBody, 200, duration, "", ipAddress)
	}

	// Send final state.
	sendA2ASSEEvent(w, flusher, req.ID, task)

	// Record reputation.
	if s.reputation != nil {
		if invokeErr == "" {
			if err := s.reputation.RecordEvent(r.Context(), agentID, "bridge_success", ""); err != nil {
				s.logger.Debug("failed to record reputation event", "agent_id", agentID, "error", err)
			}
		} else {
			if err := s.reputation.RecordEvent(r.Context(), agentID, "bridge_error", invokeErr); err != nil {
				s.logger.Debug("failed to record reputation event", "agent_id", agentID, "error", err)
			}
		}
	}
}

// handleA2ABridgeGetTask handles tasks/get.
func (s *HTTPServer) handleA2ABridgeGetTask(w http.ResponseWriter, r *http.Request, req *jsonrpc.Request) {
	var params a2a.GetTaskParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	v, ok := s.a2aTasks.tasks.Load(params.ID)
	if !ok {
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "task not found: "+params.ID)
		return
	}

	// Verify requester is the task creator.
	if creatorIP, ok := s.a2aTasks.creatorIPs.Load(params.ID); ok {
		if BridgeClientIP(r) != creatorIP.(string) {
			writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "task not found: "+params.ID)
			return
		}
	}

	writeA2ABridgeResult(w, req.ID, v.(*a2a.Task))
}

// handleA2ABridgeCancelTask handles tasks/cancel.
func (s *HTTPServer) handleA2ABridgeCancelTask(w http.ResponseWriter, r *http.Request, req *jsonrpc.Request) {
	var params a2a.CancelTaskParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	v, ok := s.a2aTasks.tasks.Load(params.ID)
	if !ok {
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "task not found: "+params.ID)
		return
	}

	// Verify requester is the task creator.
	if creatorIP, ok := s.a2aTasks.creatorIPs.Load(params.ID); ok {
		if BridgeClientIP(r) != creatorIP.(string) {
			writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "task not found: "+params.ID)
			return
		}
	}

	task := v.(*a2a.Task)
	task = s.updateTaskState(task, a2a.TaskStateCanceled, "")

	writeA2ABridgeResult(w, req.ID, task)
}

// handleA2ABridgePushNotification handles tasks/pushNotification/set and get.
func (s *HTTPServer) handleA2ABridgePushNotification(w http.ResponseWriter, req *jsonrpc.Request, method string) {
	switch method {
	case "set":
		var params a2a.SetPushNotificationParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
			return
		}
		// Verify task exists.
		if _, ok := s.a2aTasks.tasks.Load(params.ID); !ok {
			writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "task not found: "+params.ID)
			return
		}
		s.a2aTasks.pushConfigs.Store(params.ID, &params.PushNotificationConfig)
		writeA2ABridgeResult(w, req.ID, map[string]any{
			"id":                     params.ID,
			"pushNotificationConfig": params.PushNotificationConfig,
		})
	case "get":
		var params a2a.GetPushNotificationParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
			return
		}
		if _, ok := s.a2aTasks.tasks.Load(params.ID); !ok {
			writeA2ABridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "task not found: "+params.ID)
			return
		}
		v, ok := s.a2aTasks.pushConfigs.Load(params.ID)
		if !ok {
			writeA2ABridgeResult(w, req.ID, map[string]any{
				"id":                     params.ID,
				"pushNotificationConfig": nil,
			})
			return
		}
		writeA2ABridgeResult(w, req.ID, map[string]any{
			"id":                     params.ID,
			"pushNotificationConfig": v,
		})
	}
}

// handleA2ABridgeAgentCard handles GET /a2a/{agent_id}/.well-known/agent.json.
func (s *HTTPServer) handleA2ABridgeAgentCard(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agent_id")
	if agentID == "" {
		s.jsonError(w, "missing agent_id", http.StatusBadRequest)
		return
	}

	card, err := s.registry.GetAgent(r.Context(), agentID)
	if err != nil {
		s.jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	baseURL := requestBaseURL(r)
	a2aCard := cardToA2AAgentCard(card, baseURL)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(a2aCard)
}

// handleA2ABridgeGetTaskREST handles GET /a2a/{agent_id}/tasks/{task_id}.
func (s *HTTPServer) handleA2ABridgeGetTaskREST(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("task_id")
	if taskID == "" {
		s.jsonError(w, "missing task_id", http.StatusBadRequest)
		return
	}

	v, ok := s.a2aTasks.tasks.Load(taskID)
	if !ok {
		s.jsonError(w, "task not found", http.StatusNotFound)
		return
	}

	// Verify requester is the task creator.
	if creatorIP, ok := s.a2aTasks.creatorIPs.Load(taskID); ok {
		if BridgeClientIP(r) != creatorIP.(string) {
			s.jsonError(w, "task not found", http.StatusNotFound)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v.(*a2a.Task))
}

// --- Helpers ---

// cardToA2AAgentCard converts a PeerClaw Card to an A2A AgentCard.
func cardToA2AAgentCard(card *agentcard.Card, baseURL string) a2a.AgentCard {
	ac := a2a.AgentCard{
		Name:        card.Name,
		Description: card.Description,
		Version:     card.Version,
		Endpoint:    baseURL + "/a2a/" + card.ID,
		Capabilities: a2a.A2ACaps{
			Streaming: true,
			MultiTurn: true,
		},
	}

	for _, s := range card.Skills {
		ac.Skills = append(ac.Skills, a2a.Skill{
			Name:        s.Name,
			Description: s.Description,
			InputModes:  s.InputModes,
			OutputModes: s.OutputModes,
		})
	}

	return ac
}

// requestBaseURL derives the base URL from the request.
// Only trusts X-Forwarded-Proto/Host from loopback/private IPs.
// Validates forwarded values to prevent host header injection.
func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	host := r.Host

	// Only trust forwarded headers from trusted proxies (loopback/private).
	remoteHost, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip := net.ParseIP(remoteHost); ip != nil && (ip.IsLoopback() || ip.IsPrivate()) {
		if proto := r.Header.Get("X-Forwarded-Proto"); proto == "http" || proto == "https" {
			scheme = proto
		}
		if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
			fwd = strings.TrimSpace(fwd)
			// Reject values containing path separators or whitespace to prevent injection.
			if !strings.ContainsAny(fwd, "/\\? \t\n\r") {
				// Validate host:port format if port is present.
				if h, p, err := net.SplitHostPort(fwd); err == nil {
					// Has port — validate hostname and port are reasonable.
					if h != "" && p != "" && (net.ParseIP(h) != nil || !strings.ContainsAny(h, "/:")) {
						host = fwd
					}
				} else {
					// No port — just a hostname; validate no colons (except IPv6 which would have port).
					if !strings.Contains(fwd, ":") || net.ParseIP(fwd) != nil {
						host = fwd
					}
				}
			}
		}
	}

	return scheme + "://" + host
}


// cloneA2ATask returns a shallow copy of the task with a fresh Artifacts slice.
func cloneA2ATask(task *a2a.Task) *a2a.Task {
	cp := *task
	if task.Artifacts != nil {
		cp.Artifacts = make([]a2a.Artifact, len(task.Artifacts))
		copy(cp.Artifacts, task.Artifacts)
	}
	return &cp
}

// updateTaskState creates a copy of the task with updated status and stores it.
// Returns the new copy so callers can continue working with it.
func (s *HTTPServer) updateTaskState(task *a2a.Task, state a2a.TaskState, message string) *a2a.Task {
	cp := cloneA2ATask(task)
	now := time.Now().UTC().Format(time.RFC3339)
	cp.Status = a2a.TaskStatus{
		State:     state,
		Message:   message,
		Timestamp: now,
	}
	cp.UpdatedAt = now
	s.a2aTasks.tasks.Store(cp.ID, cp)
	return cp
}

// recordA2ABridgeInvocation records an A2A bridge invocation.
func (s *HTTPServer) recordA2ABridgeInvocation(ctx context.Context, agentID, proto, reqBody, respBody string, statusCode int, durationMs int64, invokeErr, ipAddress string) {
	s.recordBridgeInvocation(ctx, "a2a-bridge", agentID, proto, reqBody, respBody, statusCode, durationMs, invokeErr, ipAddress)
}

// sendA2ASSEEvent sends a JSON-RPC response wrapped in an SSE event.
func sendA2ASSEEvent(w http.ResponseWriter, flusher http.Flusher, reqID any, task *a2a.Task) {
	resp, err := jsonrpc.NewResponse(reqID, task)
	if err != nil {
		return
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(data))
	flusher.Flush()
}

func writeA2ABridgeResult(w http.ResponseWriter, id any, result any) {
	resp, err := jsonrpc.NewResponse(id, result)
	if err != nil {
		writeA2ABridgeError(w, id, jsonrpc.CodeInternalError, "marshal error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeA2ABridgeError(w http.ResponseWriter, id any, code int, message string) {
	resp := jsonrpc.NewErrorResponse(id, code, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors use 200
	_ = json.NewEncoder(w).Encode(resp)
}

// a2aBridgeCleanup periodically removes expired tasks. Stops when ctx is cancelled.
func (s *HTTPServer) a2aBridgeCleanup(ctx context.Context, cleanupInterval, maxAge time.Duration) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().UTC()
			s.a2aTasks.tasks.Range(func(key, value any) bool {
				task := value.(*a2a.Task)
				created, err := time.Parse(time.RFC3339, task.CreatedAt)
				if err != nil {
					return true
				}
				if now.Sub(created) > maxAge {
					s.a2aTasks.tasks.Delete(key)
					s.a2aTasks.pushConfigs.Delete(key)
					s.a2aTasks.creatorIPs.Delete(key)
					s.a2aTasks.taskCount.Add(-1)
				}
				return true
			})
		}
	}
}
