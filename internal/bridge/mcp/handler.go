package mcp

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
)

// HandleMCP handles POST /mcp — the Streamable HTTP endpoint for MCP.
func (a *Adapter) HandleMCP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeMCPError(w, nil, jsonrpc.CodeParseError, "failed to read body")
		return
	}

	parsed, err := jsonrpc.ParseMessage(body)
	if err != nil {
		writeMCPError(w, nil, jsonrpc.CodeParseError, "invalid JSON-RPC: "+err.Error())
		return
	}

	if parsed.Kind == jsonrpc.KindNotification {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if parsed.Kind != jsonrpc.KindRequest {
		writeMCPError(w, nil, jsonrpc.CodeInvalidRequest, "expected JSON-RPC request")
		return
	}

	req := parsed.Request

	// Generate or reuse session ID.
	sessionID := r.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	switch req.Method {
	case "initialize":
		a.handleInitialize(w, req, sessionID)
	case "tools/list":
		a.handleToolsList(w, req, sessionID)
	case "tools/call":
		a.handleToolsCall(w, req, sessionID)
	case "resources/list":
		a.handleResourcesList(w, req, sessionID)
	case "resources/read":
		a.handleResourcesRead(w, req, sessionID)
	case "prompts/list":
		a.handlePromptsList(w, req, sessionID)
	case "prompts/get":
		a.handlePromptsGet(w, req, sessionID)
	default:
		writeMCPError(w, req.ID, jsonrpc.CodeMethodNotFound, "unknown method: "+req.Method)
	}
}

// HandleMCPStream handles GET /mcp — SSE stream for server-initiated messages.
func (a *Adapter) HandleMCPStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Keep connection open; close when client disconnects.
	<-r.Context().Done()
	_ = flusher // ensure flusher is used
}

func (a *Adapter) handleInitialize(w http.ResponseWriter, req *jsonrpc.Request, sessionID string) {
	result := InitializeResult{
		ProtocolVersion: a.version,
		Capabilities: ServerCaps{
			Tools:     &ToolsCap{},
			Resources: &ResourcesCap{},
			Prompts:   &PromptsCap{},
		},
		ServerInfo: ImplementInfo{
			Name:    "PeerClaw",
			Version: "1.0",
		},
	}

	w.Header().Set("Mcp-Session-Id", sessionID)
	writeMCPResult(w, req.ID, result)
}

func (a *Adapter) handleToolsList(w http.ResponseWriter, req *jsonrpc.Request, sessionID string) {
	// Return an empty tools list; agents register their tools dynamically.
	result := ToolsListResult{
		Tools: []ToolDef{},
	}
	w.Header().Set("Mcp-Session-Id", sessionID)
	writeMCPResult(w, req.ID, result)
}

func (a *Adapter) handleToolsCall(w http.ResponseWriter, req *jsonrpc.Request, sessionID string) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeMCPError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	// Convert to Envelope and push into inbox for bridge processing.
	payload, _ := json.Marshal(params.Arguments)
	env := envelope.New("external", "", protocol.ProtocolMCP, payload)
	env.Metadata["mcp.method"] = "tools/call"
	env.Metadata["mcp.tool_name"] = params.Name
	env.Metadata["mcp.session_id"] = sessionID

	select {
	case a.inbox <- env:
	default:
		a.logger.Warn("mcp inbox full, dropping tools/call")
		writeMCPError(w, req.ID, jsonrpc.CodeInternalError, "server busy")
		return
	}

	// For now, return a placeholder result since actual processing is async.
	result := ToolCallResult{
		Content: []Content{
			{Type: "text", Text: "Tool call accepted: " + params.Name},
		},
	}
	w.Header().Set("Mcp-Session-Id", sessionID)
	writeMCPResult(w, req.ID, result)
}

func (a *Adapter) handleResourcesList(w http.ResponseWriter, req *jsonrpc.Request, sessionID string) {
	result := ResourcesListResult{
		Resources: []Resource{},
	}
	w.Header().Set("Mcp-Session-Id", sessionID)
	writeMCPResult(w, req.ID, result)
}

func (a *Adapter) handleResourcesRead(w http.ResponseWriter, req *jsonrpc.Request, sessionID string) {
	var params ResourceReadParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeMCPError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	// Push to inbox for processing.
	env := envelope.New("external", "", protocol.ProtocolMCP, []byte(params.URI))
	env.Metadata["mcp.method"] = "resources/read"
	env.Metadata["mcp.resource_uri"] = params.URI
	env.Metadata["mcp.session_id"] = sessionID

	select {
	case a.inbox <- env:
	default:
		a.logger.Warn("mcp inbox full, dropping resources/read")
	}

	result := ResourceReadResult{
		Contents: []ResourceContent{},
	}
	w.Header().Set("Mcp-Session-Id", sessionID)
	writeMCPResult(w, req.ID, result)
}

func (a *Adapter) handlePromptsList(w http.ResponseWriter, req *jsonrpc.Request, sessionID string) {
	result := PromptsListResult{
		Prompts: []Prompt{},
	}
	w.Header().Set("Mcp-Session-Id", sessionID)
	writeMCPResult(w, req.ID, result)
}

func (a *Adapter) handlePromptsGet(w http.ResponseWriter, req *jsonrpc.Request, sessionID string) {
	var params PromptGetParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeMCPError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	result := PromptGetResult{
		Messages: []PromptMessage{},
	}
	w.Header().Set("Mcp-Session-Id", sessionID)
	writeMCPResult(w, req.ID, result)
}

func writeMCPResult(w http.ResponseWriter, id any, result any) {
	resp, err := jsonrpc.NewResponse(id, result)
	if err != nil {
		writeMCPError(w, id, jsonrpc.CodeInternalError, "marshal error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeMCPError(w http.ResponseWriter, id any, code int, message string) {
	resp := jsonrpc.NewErrorResponse(id, code, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
