package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-server/internal/bridge/a2a"
	"github.com/peerclaw/peerclaw-server/internal/bridge/acp"
	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
)

func TestGateway_DetectA2A(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "message/send",
		"params": map[string]any{
			"message": map[string]any{
				"role": "user",
				"parts": []map[string]any{
					{"text": "hello via gateway"},
				},
			},
		},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/agent/"+agentID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Should be a JSON-RPC response (routed to A2A handler).
	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)
	// The response should be valid JSON-RPC (either result or error from bridge).
	if resp.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q, want %q", resp.JSONRPC, "2.0")
	}
}

func TestGateway_DetectMCP(t *testing.T) {
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
				"name":    "test",
				"version": "1.0.0",
			},
		},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/agent/"+agentID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Should be routed to MCP handler — check for Mcp-Session-Id.
	sessionID := w.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Error("expected Mcp-Session-Id header (MCP route)")
	}
}

func TestGateway_DetectACP(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	acpReq := acp.CreateRunRequest{
		Input: []acp.Message{
			{
				Role: "user",
				Parts: []acp.MessagePart{
					{ContentType: "text/plain", Content: "hello via gateway"},
				},
			},
		},
	}
	body, _ := json.Marshal(acpReq)

	req := httptest.NewRequest("POST", "/agent/"+agentID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Should be routed to ACP handler — check for run_id in response.
	var run acp.Run
	json.NewDecoder(w.Body).Decode(&run)
	if run.RunID == "" {
		t.Error("expected run_id in response (ACP route)")
	}
}

func TestGateway_UnknownProtocol(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	// Unrecognizable JSON body.
	body := []byte(`{"foo": "bar", "baz": 123}`)

	req := httptest.NewRequest("POST", "/agent/"+agentID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGateway_Discover_Default(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	req := httptest.NewRequest("GET", "/agent/"+agentID, nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var card agentcard.Card
	if err := json.NewDecoder(w.Body).Decode(&card); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if card.Name != "TestA2AAgent" {
		t.Errorf("name = %q, want %q", card.Name, "TestA2AAgent")
	}
}

func TestGateway_Discover_A2A(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	req := httptest.NewRequest("GET", "/agent/"+agentID+"?format=a2a", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var card a2a.AgentCard
	if err := json.NewDecoder(w.Body).Decode(&card); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if card.Name != "TestA2AAgent" {
		t.Errorf("name = %q, want %q", card.Name, "TestA2AAgent")
	}
	if card.Endpoint == "" {
		t.Error("endpoint should not be empty")
	}
}

func TestGateway_Discover_MCP(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	req := httptest.NewRequest("GET", "/agent/"+agentID+"?format=mcp", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var info map[string]any
	if err := json.NewDecoder(w.Body).Decode(&info); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if info["name"] != "TestA2AAgent" {
		t.Errorf("name = %v, want %q", info["name"], "TestA2AAgent")
	}
	if info["protocol"] != "mcp" {
		t.Errorf("protocol = %v, want %q", info["protocol"], "mcp")
	}
}

func TestGateway_Discover_ACP(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	req := httptest.NewRequest("GET", "/agent/"+agentID+"?format=acp", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var manifest acp.AgentManifest
	if err := json.NewDecoder(w.Body).Decode(&manifest); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if manifest.Name != "TestA2AAgent" {
		t.Errorf("name = %q, want %q", manifest.Name, "TestA2AAgent")
	}
}

func TestGateway_Discover_NotFound(t *testing.T) {
	s := newTestHTTPServerWithACL(t)

	req := httptest.NewRequest("GET", "/agent/nonexistent", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDetectProtocol(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "a2a message/send",
			body:     `{"jsonrpc":"2.0","id":1,"method":"message/send","params":{}}`,
			expected: "a2a",
		},
		{
			name:     "a2a tasks/get",
			body:     `{"jsonrpc":"2.0","id":1,"method":"tasks/get","params":{"id":"123"}}`,
			expected: "a2a",
		},
		{
			name:     "mcp initialize",
			body:     `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
			expected: "mcp",
		},
		{
			name:     "mcp tools/list",
			body:     `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
			expected: "mcp",
		},
		{
			name:     "mcp tools/call",
			body:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test"}}`,
			expected: "mcp",
		},
		{
			name:     "acp with input",
			body:     `{"input":[{"role":"user","parts":[{"content":"hello"}]}]}`,
			expected: "acp",
		},
		{
			name:     "acp with agent_name",
			body:     `{"agent_name":"test","input":[]}`,
			expected: "acp",
		},
		{
			name:     "unknown",
			body:     `{"foo":"bar"}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectProtocol([]byte(tt.body))
			if got != tt.expected {
				t.Errorf("detectProtocol() = %q, want %q", got, tt.expected)
			}
		})
	}
}
