package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
	"github.com/peerclaw/peerclaw-server/internal/bridge/mcp"
)

func TestMCPBridge_Initialize(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/mcp/"+agentID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp jsonrpc.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result mcp.InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if result.ServerInfo.Name != "TestA2AAgent" {
		t.Errorf("server name = %q, want %q", result.ServerInfo.Name, "TestA2AAgent")
	}

	sessionID := w.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Error("Mcp-Session-Id header should not be empty")
	}
}

func TestMCPBridge_Initialize_NotFound(t *testing.T) {
	s := newTestHTTPServer(t)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/mcp/nonexistent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error == nil {
		t.Error("expected error for nonexistent agent")
	}
	if resp.Error != nil && resp.Error.Code != jsonrpc.CodeInvalidParams {
		t.Errorf("error code = %d, want %d", resp.Error.Code, jsonrpc.CodeInvalidParams)
	}
}

func TestMCPBridge_ToolsList(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/mcp/"+agentID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result mcp.ToolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	// Test agent has no tools, so expect empty list.
	if result.Tools == nil {
		t.Error("tools should not be nil")
	}
}

func TestMCPBridge_ToolsCall(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "test_tool",
			"arguments": map[string]any{"query": "hello"},
		},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/mcp/"+agentID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)

	// With no real bridge, we expect a ToolCallResult (possibly error).
	if resp.Result != nil {
		var result mcp.ToolCallResult
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			t.Fatalf("unmarshal result: %v", err)
		}
		if len(result.Content) == 0 {
			t.Error("content should not be empty")
		}
	}
}

func TestMCPBridge_ToolsCall_NoAgent(t *testing.T) {
	s := newTestHTTPServerWithACL(t)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "test_tool",
			"arguments": map[string]any{},
		},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/mcp/nonexistent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error == nil {
		t.Error("expected error for nonexistent agent")
	}
}

func TestMCPBridge_UnknownMethod(t *testing.T) {
	s := newTestHTTPServer(t)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "foo/bar",
		"params":  map[string]any{},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/mcp/any-agent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error == nil {
		t.Error("expected error for unknown method")
	}
	if resp.Error != nil && resp.Error.Code != jsonrpc.CodeMethodNotFound {
		t.Errorf("error code = %d, want %d", resp.Error.Code, jsonrpc.CodeMethodNotFound)
	}
}

func TestMCPBridge_InvalidJSON(t *testing.T) {
	s := newTestHTTPServer(t)

	req := httptest.NewRequest("POST", "/mcp/any-agent", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error == nil {
		t.Error("expected error for invalid JSON")
	}
	if resp.Error != nil && resp.Error.Code != jsonrpc.CodeParseError {
		t.Errorf("error code = %d, want %d", resp.Error.Code, jsonrpc.CodeParseError)
	}
}

func TestMCPBridge_Notification(t *testing.T) {
	s := newTestHTTPServer(t)

	// Notification: no "id" field.
	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
		"params":  map[string]any{},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/mcp/any-agent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestMCPBridge_ResourcesList(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "resources/list",
		"params":  map[string]any{},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/mcp/"+agentID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result mcp.ResourcesListResult
	json.Unmarshal(resp.Result, &result)
	if len(result.Resources) != 0 {
		t.Errorf("resources count = %d, want 0", len(result.Resources))
	}
}

func TestMCPBridge_PromptsList(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  "prompts/list",
		"params":  map[string]any{},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/mcp/"+agentID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result mcp.PromptsListResult
	json.Unmarshal(resp.Result, &result)
	if len(result.Prompts) != 0 {
		t.Errorf("prompts count = %d, want 0", len(result.Prompts))
	}
}
