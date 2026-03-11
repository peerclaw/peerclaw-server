package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-core/envelope"
	coreprotocol "github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
	"github.com/peerclaw/peerclaw-server/internal/bridge/mcp"
	"github.com/peerclaw/peerclaw-server/internal/invocation"
)

// mcpBridgeSessions holds in-memory MCP session state for the bridge.
type mcpBridgeSessions struct {
	sessions     sync.Map
	sessionCount atomic.Int64
}

type mcpBridgeSession struct {
	SessionID string
	AgentID   string
	CreatedAt string
}

func (s *HTTPServer) registerMCPBridgeRoutes() {
	s.mcpSessions = &mcpBridgeSessions{}
	s.mux.HandleFunc("POST /mcp/{agent_id}", s.handleMCPBridgeMessages)
	s.mux.HandleFunc("GET /mcp/{agent_id}", s.handleMCPBridgeStream)

	// Start background cleanup.
	go s.mcpBridgeCleanup(time.Hour)
}

// handleMCPBridgeMessages handles POST /mcp/{agent_id} — JSON-RPC dispatch.
func (s *HTTPServer) handleMCPBridgeMessages(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agent_id")
	if agentID == "" {
		writeMCPBridgeError(w, nil, jsonrpc.CodeInvalidParams, "missing agent_id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeMCPBridgeError(w, nil, jsonrpc.CodeParseError, "failed to read body")
		return
	}

	parsed, err := jsonrpc.ParseMessage(body)
	if err != nil {
		writeMCPBridgeError(w, nil, jsonrpc.CodeParseError, "invalid JSON-RPC: "+err.Error())
		return
	}

	if parsed.Kind == jsonrpc.KindNotification {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if parsed.Kind != jsonrpc.KindRequest {
		writeMCPBridgeError(w, nil, jsonrpc.CodeInvalidRequest, "expected JSON-RPC request")
		return
	}

	req := parsed.Request

	switch req.Method {
	case "initialize":
		s.handleMCPBridgeInitialize(w, r, req, agentID)
	case "tools/list":
		s.handleMCPBridgeToolsList(w, r, req, agentID)
	case "tools/call":
		s.handleMCPBridgeToolsCall(w, r, req, agentID)
	case "resources/list":
		s.handleMCPBridgeResourcesList(w, req)
	case "prompts/list":
		s.handleMCPBridgePromptsList(w, req)
	default:
		writeMCPBridgeError(w, req.ID, jsonrpc.CodeMethodNotFound, "unknown method: "+req.Method)
	}
}

// handleMCPBridgeInitialize handles the initialize method.
func (s *HTTPServer) handleMCPBridgeInitialize(w http.ResponseWriter, r *http.Request, req *jsonrpc.Request, agentID string) {
	card, err := s.registry.GetAgent(r.Context(), agentID)
	if err != nil {
		writeMCPBridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "agent not found: "+agentID)
		return
	}

	// Create session.
	sessionID := uuid.New().String()
	session := &mcpBridgeSession{
		SessionID: sessionID,
		AgentID:   agentID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	s.mcpSessions.sessions.Store(sessionID, session)
	s.mcpSessions.sessionCount.Add(1)

	result := mcp.InitializeResult{
		ProtocolVersion: "2025-03-26",
		Capabilities: mcp.ServerCaps{
			Tools: &mcp.ToolsCap{},
		},
		ServerInfo: mcp.ImplementInfo{
			Name:    card.Name,
			Version: card.Version,
		},
		Instructions: card.Description,
	}

	w.Header().Set("Mcp-Session-Id", sessionID)
	writeMCPBridgeResult(w, req.ID, result)
}

// handleMCPBridgeToolsList handles tools/list.
func (s *HTTPServer) handleMCPBridgeToolsList(w http.ResponseWriter, r *http.Request, req *jsonrpc.Request, agentID string) {
	card, err := s.registry.GetAgent(r.Context(), agentID)
	if err != nil {
		writeMCPBridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "agent not found: "+agentID)
		return
	}

	var tools []mcp.ToolDef
	for _, t := range card.Tools {
		tools = append(tools, mcp.ToolDef{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	if tools == nil {
		tools = []mcp.ToolDef{}
	}

	writeMCPBridgeResult(w, req.ID, mcp.ToolsListResult{Tools: tools})
}

// handleMCPBridgeToolsCall handles tools/call.
func (s *HTTPServer) handleMCPBridgeToolsCall(w http.ResponseWriter, r *http.Request, req *jsonrpc.Request, agentID string) {
	var params mcp.ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeMCPBridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	// Resolve agent.
	card, err := s.registry.GetAgent(r.Context(), agentID)
	if err != nil {
		writeMCPBridgeError(w, req.ID, jsonrpc.CodeInvalidParams, "agent not found: "+agentID)
		return
	}

	// Access control.
	flags, err := s.registry.GetAccessFlags(r.Context(), agentID)
	if err != nil {
		writeMCPBridgeError(w, req.ID, jsonrpc.CodeInternalError, "failed to check access flags")
		return
	}
	if !flags.PlaygroundEnabled {
		writeMCPBridgeError(w, req.ID, -32001, "access denied: agent does not allow external MCP access")
		return
	}

	// Rate limiting by IP.
	if s.invokeRateLimiter != nil {
		ipAddress := mcpBridgeClientIP(r)
		if !s.invokeRateLimiter.GetLimiter("mcp:"+ipAddress).Allow() {
			writeMCPBridgeError(w, req.ID, -32002, "rate limit exceeded")
			return
		}
	}

	// Determine protocol.
	proto := "mcp"
	if len(card.Protocols) > 0 {
		proto = string(card.Protocols[0])
	}

	// Build envelope.
	payload := params.Arguments
	if payload == nil {
		payload = json.RawMessage("{}")
	}

	env := envelope.New("mcp-bridge", agentID, coreprotocol.Protocol(proto), payload)
	env.WithMetadata("mcp.tool_name", params.Name)

	start := time.Now()
	ipAddress := mcpBridgeClientIP(r)

	if s.bridges == nil {
		writeMCPBridgeError(w, req.ID, jsonrpc.CodeInternalError, "bridge not available")
		return
	}

	chunks, err := s.bridges.SendStream(r.Context(), env)
	if err != nil {
		s.recordMCPBridgeInvocation(r.Context(), agentID, proto, string(payload), "", 502, time.Since(start).Milliseconds(), err.Error(), ipAddress)
		writeMCPBridgeResult(w, req.ID, mcp.ToolCallResult{
			Content: []mcp.Content{{Type: "text", Text: "error: " + err.Error()}},
			IsError: true,
		})
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
		s.recordMCPBridgeInvocation(r.Context(), agentID, proto, string(payload), respBody, 502, duration, invokeErr, ipAddress)
		writeMCPBridgeResult(w, req.ID, mcp.ToolCallResult{
			Content: []mcp.Content{{Type: "text", Text: "error: " + invokeErr}},
			IsError: true,
		})
	} else {
		s.recordMCPBridgeInvocation(r.Context(), agentID, proto, string(payload), respBody, 200, duration, "", ipAddress)
		writeMCPBridgeResult(w, req.ID, mcp.ToolCallResult{
			Content: []mcp.Content{{Type: "text", Text: respBody}},
		})
	}

	// Record reputation.
	if s.reputation != nil {
		if invokeErr == "" {
			_ = s.reputation.RecordEvent(r.Context(), agentID, "bridge_success", "")
		} else {
			_ = s.reputation.RecordEvent(r.Context(), agentID, "bridge_error", invokeErr)
		}
	}
}

// handleMCPBridgeResourcesList handles resources/list — returns empty list.
func (s *HTTPServer) handleMCPBridgeResourcesList(w http.ResponseWriter, req *jsonrpc.Request) {
	writeMCPBridgeResult(w, req.ID, mcp.ResourcesListResult{Resources: []mcp.Resource{}})
}

// handleMCPBridgePromptsList handles prompts/list — returns empty list.
func (s *HTTPServer) handleMCPBridgePromptsList(w http.ResponseWriter, req *jsonrpc.Request) {
	writeMCPBridgeResult(w, req.ID, mcp.PromptsListResult{Prompts: []mcp.Prompt{}})
}

// handleMCPBridgeStream handles GET /mcp/{agent_id} — SSE placeholder.
func (s *HTTPServer) handleMCPBridgeStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Keep connection open until client disconnects.
	<-r.Context().Done()
}

// --- Helpers ---

// mcpBridgeClientIP extracts the client IP address from the request.
func mcpBridgeClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return r.RemoteAddr
}

// recordMCPBridgeInvocation records an invocation to the invocation service.
func (s *HTTPServer) recordMCPBridgeInvocation(ctx context.Context, agentID, proto, reqBody, respBody string, statusCode int, durationMs int64, invokeErr, ipAddress string) {
	if s.invocation == nil {
		return
	}
	_ = s.invocation.Record(ctx, &invocation.InvocationRecord{
		ID:           uuid.New().String(),
		AgentID:      agentID,
		UserID:       "mcp-bridge",
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

func writeMCPBridgeResult(w http.ResponseWriter, id any, result any) {
	resp, err := jsonrpc.NewResponse(id, result)
	if err != nil {
		writeMCPBridgeError(w, id, jsonrpc.CodeInternalError, "marshal error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeMCPBridgeError(w http.ResponseWriter, id any, code int, message string) {
	resp := jsonrpc.NewErrorResponse(id, code, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors use 200
	_ = json.NewEncoder(w).Encode(resp)
}

// mcpBridgeCleanup periodically removes expired sessions.
func (s *HTTPServer) mcpBridgeCleanup(maxAge time.Duration) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().UTC()
		s.mcpSessions.sessions.Range(func(key, value any) bool {
			session := value.(*mcpBridgeSession)
			created, err := time.Parse(time.RFC3339, session.CreatedAt)
			if err != nil {
				return true
			}
			if now.Sub(created) > maxAge {
				s.mcpSessions.sessions.Delete(key)
				s.mcpSessions.sessionCount.Add(-1)
			}
			return true
		})
	}
}

// cardToMCPInfo creates MCP server info from card fields directly.
func cardToMCPInfo(name, version, description, endpoint string) map[string]any {
	return map[string]any{
		"name":        name,
		"version":     version,
		"description": description,
		"endpoint":    endpoint,
		"protocol":    "mcp",
	}
}
