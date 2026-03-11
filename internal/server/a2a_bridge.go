package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/peerclaw/peerclaw-server/internal/invocation"
)

// a2aBridgeTasks holds in-memory A2A task state for the bridge.
type a2aBridgeTasks struct {
	tasks     sync.Map
	taskCount atomic.Int64

	// pushConfigs stores push notification configs keyed by task ID.
	pushConfigs sync.Map // taskID → *a2a.PushNotificationConfig
}

func (s *HTTPServer) registerA2ABridgeRoutes() {
	s.a2aTasks = &a2aBridgeTasks{}
	s.mux.HandleFunc("POST /a2a/{agent_id}", s.handleA2ABridgeMessages)
	s.mux.HandleFunc("GET /a2a/{agent_id}/.well-known/agent.json", s.handleA2ABridgeAgentCard)
	s.mux.HandleFunc("GET /a2a/{agent_id}/tasks/{task_id}", s.handleA2ABridgeGetTaskREST)

	// Start background cleanup.
	go s.a2aBridgeCleanup(time.Hour)
}

// handleA2ABridgeMessages handles POST /a2a/{agent_id} — JSON-RPC dispatch.
func (s *HTTPServer) handleA2ABridgeMessages(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agent_id")
	if agentID == "" {
		writeA2ABridgeError(w, nil, jsonrpc.CodeInvalidParams, "missing agent_id")
		return
	}

	body, err := io.ReadAll(r.Body)
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
		s.handleA2ABridgeGetTask(w, req)
	case "tasks/cancel":
		s.handleA2ABridgeCancelTask(w, req)
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
	if !flags.PlaygroundEnabled {
		writeA2ABridgeError(w, req.ID, -32001, "access denied: agent does not allow external A2A access")
		return
	}

	// Rate limiting by IP.
	if s.invokeRateLimiter != nil {
		ipAddress := a2aBridgeClientIP(r)
		if !s.invokeRateLimiter.GetLimiter("a2a:"+ipAddress).Allow() {
			writeA2ABridgeError(w, req.ID, -32002, "rate limit exceeded")
			return
		}
	}

	// Create task.
	contextID := params.Message.ContextID
	if contextID == "" {
		contextID = uuid.New().String()
	}
	task := a2a.NewTask(contextID, params.Message)
	s.a2aTasks.tasks.Store(task.ID, task)
	s.a2aTasks.taskCount.Add(1)

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
	ipAddress := a2aBridgeClientIP(r)

	// Synchronous: collect all chunks from stream.
	if s.bridges == nil {
		s.updateTaskState(task, a2a.TaskStateFailed, "bridge not available")
		writeA2ABridgeError(w, req.ID, jsonrpc.CodeInternalError, "bridge not available")
		return
	}

	// Update task to working.
	s.updateTaskState(task, a2a.TaskStateWorking, "")

	chunks, err := s.bridges.SendStream(r.Context(), env)
	if err != nil {
		s.updateTaskState(task, a2a.TaskStateFailed, err.Error())
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
		s.updateTaskState(task, a2a.TaskStateFailed, invokeErr)
		s.recordA2ABridgeInvocation(r.Context(), agentID, proto, payload, respBody, 502, duration, invokeErr, ipAddress)
	} else {
		// Add response as artifact.
		task.Artifacts = append(task.Artifacts, a2a.Artifact{
			ID: uuid.New().String(),
			Parts: []a2a.Part{
				{Text: respBody},
			},
		})
		s.updateTaskState(task, a2a.TaskStateCompleted, "")
		s.recordA2ABridgeInvocation(r.Context(), agentID, proto, payload, respBody, 200, duration, "", ipAddress)
	}

	// Record reputation.
	if s.reputation != nil {
		if invokeErr == "" {
			_ = s.reputation.RecordEvent(r.Context(), agentID, "bridge_success", "")
		} else {
			_ = s.reputation.RecordEvent(r.Context(), agentID, "bridge_error", invokeErr)
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
	if !flags.PlaygroundEnabled {
		writeA2ABridgeError(w, req.ID, -32001, "access denied: agent does not allow external A2A access")
		return
	}

	// Rate limiting.
	if s.invokeRateLimiter != nil {
		ipAddress := a2aBridgeClientIP(r)
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
	ipAddress := a2aBridgeClientIP(r)

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Send initial working state.
	s.updateTaskState(task, a2a.TaskStateWorking, "")
	sendA2ASSEEvent(w, flusher, req.ID, task)

	if s.bridges == nil {
		s.updateTaskState(task, a2a.TaskStateFailed, "bridge not available")
		sendA2ASSEEvent(w, flusher, req.ID, task)
		return
	}

	chunks, err := s.bridges.SendStream(r.Context(), env)
	if err != nil {
		s.updateTaskState(task, a2a.TaskStateFailed, err.Error())
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
			// Send artifact update event.
			task.Artifacts = []a2a.Artifact{
				{
					ID: fmt.Sprintf("artifact-%d", artifactIndex),
					Parts: []a2a.Part{
						{Text: sb.String()},
					},
				},
			}
			task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			sendA2ASSEEvent(w, flusher, req.ID, task)
		}
		if chunk.Done {
			break
		}
	}

	duration := time.Since(start).Milliseconds()
	respBody := sb.String()

	if invokeErr != "" {
		s.updateTaskState(task, a2a.TaskStateFailed, invokeErr)
		s.recordA2ABridgeInvocation(r.Context(), agentID, proto, payload, respBody, 502, duration, invokeErr, ipAddress)
	} else {
		task.Artifacts = []a2a.Artifact{
			{
				ID: "artifact-final",
				Parts: []a2a.Part{
					{Text: respBody},
				},
			},
		}
		s.updateTaskState(task, a2a.TaskStateCompleted, "")
		s.recordA2ABridgeInvocation(r.Context(), agentID, proto, payload, respBody, 200, duration, "", ipAddress)
	}

	// Send final state.
	sendA2ASSEEvent(w, flusher, req.ID, task)

	// Record reputation.
	if s.reputation != nil {
		if invokeErr == "" {
			_ = s.reputation.RecordEvent(r.Context(), agentID, "bridge_success", "")
		} else {
			_ = s.reputation.RecordEvent(r.Context(), agentID, "bridge_error", invokeErr)
		}
	}
}

// handleA2ABridgeGetTask handles tasks/get.
func (s *HTTPServer) handleA2ABridgeGetTask(w http.ResponseWriter, req *jsonrpc.Request) {
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

	writeA2ABridgeResult(w, req.ID, v.(*a2a.Task))
}

// handleA2ABridgeCancelTask handles tasks/cancel.
func (s *HTTPServer) handleA2ABridgeCancelTask(w http.ResponseWriter, req *jsonrpc.Request) {
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

	task := v.(*a2a.Task)
	s.updateTaskState(task, a2a.TaskStateCanceled, "")

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
		http.Error(w, `{"error":"missing agent_id"}`, http.StatusBadRequest)
		return
	}

	card, err := s.registry.GetAgent(r.Context(), agentID)
	if err != nil {
		http.Error(w, `{"error":"agent not found"}`, http.StatusNotFound)
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
		http.Error(w, `{"error":"missing task_id"}`, http.StatusBadRequest)
		return
	}

	v, ok := s.a2aTasks.tasks.Load(taskID)
	if !ok {
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
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

// requestBaseURL derives the base URL from the request (X-Forwarded-Proto/Host take priority).
func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}

	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		host = fwd
	}

	return scheme + "://" + host
}

// a2aBridgeClientIP extracts the client IP address from the request.
func a2aBridgeClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return r.RemoteAddr
}

// updateTaskState updates a task's status and timestamp.
func (s *HTTPServer) updateTaskState(task *a2a.Task, state a2a.TaskState, message string) {
	now := time.Now().UTC().Format(time.RFC3339)
	task.Status = a2a.TaskStatus{
		State:     state,
		Message:   message,
		Timestamp: now,
	}
	task.UpdatedAt = now
	s.a2aTasks.tasks.Store(task.ID, task)
}

// recordA2ABridgeInvocation records an invocation to the invocation service.
func (s *HTTPServer) recordA2ABridgeInvocation(ctx context.Context, agentID, proto, reqBody, respBody string, statusCode int, durationMs int64, invokeErr, ipAddress string) {
	if s.invocation == nil {
		return
	}
	_ = s.invocation.Record(ctx, &invocation.InvocationRecord{
		ID:           uuid.New().String(),
		AgentID:      agentID,
		UserID:       "a2a-bridge",
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

// a2aBridgeCleanup periodically removes expired tasks.
func (s *HTTPServer) a2aBridgeCleanup(maxAge time.Duration) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
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
				s.a2aTasks.taskCount.Add(-1)
			}
			return true
		})
	}
}
