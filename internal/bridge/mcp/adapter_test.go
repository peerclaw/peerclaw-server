package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
	"github.com/peerclaw/peerclaw-server/internal/security"
)

func init() {
	security.AllowLocalhost = true
}

func TestAdapterProtocol(t *testing.T) {
	a := New(nil, nil)
	defer a.Close()
	if a.Protocol() != "mcp" {
		t.Errorf("Protocol() = %q", a.Protocol())
	}
}

func mockMCPServer(t *testing.T) *httptest.Server {
	t.Helper()
	initialized := false
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonrpc.Request
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Mcp-Session-Id", "test-session")

		switch req.Method {
		case "initialize":
			initialized = true
			result := InitializeResult{
				ProtocolVersion: "2025-11-25",
				Capabilities: ServerCaps{
					Tools: &ToolsCap{},
				},
				ServerInfo: ImplementInfo{Name: "mock-server", Version: "1.0"},
			}
			resp, _ := jsonrpc.NewResponse(req.ID, result)
			json.NewEncoder(w).Encode(resp)

		case "notifications/initialized":
			w.WriteHeader(http.StatusOK)

		case "tools/call":
			if !initialized {
				t.Error("tools/call before initialize")
			}
			result := ToolCallResult{
				Content: []Content{{Type: "text", Text: "tool result"}},
			}
			resp, _ := jsonrpc.NewResponse(req.ID, result)
			json.NewEncoder(w).Encode(resp)

		case "tools/list":
			result := ToolsListResult{
				Tools: []ToolDef{
					{Name: "search", Description: "Search the web"},
				},
			}
			resp, _ := jsonrpc.NewResponse(req.ID, result)
			json.NewEncoder(w).Encode(resp)

		case "resources/list":
			result := ResourcesListResult{Resources: []Resource{}}
			resp, _ := jsonrpc.NewResponse(req.ID, result)
			json.NewEncoder(w).Encode(resp)

		case "resources/read":
			result := ResourceReadResult{
				Contents: []ResourceContent{
					{URI: "file:///test", Text: "content"},
				},
			}
			resp, _ := jsonrpc.NewResponse(req.ID, result)
			json.NewEncoder(w).Encode(resp)

		default:
			resp := jsonrpc.NewErrorResponse(req.ID, jsonrpc.CodeMethodNotFound, "unknown")
			json.NewEncoder(w).Encode(resp)
		}
	}))
}

func TestAdapterSend_ToolsCall(t *testing.T) {
	server := mockMCPServer(t)
	defer server.Close()

	adapter := New(nil, server.Client())
	defer adapter.Close()

	args, _ := json.Marshal(map[string]string{"query": "hello"})
	env := envelope.New("alice", "bob", protocol.ProtocolMCP, args)
	env.Metadata["mcp.endpoint"] = server.URL
	env.Metadata["mcp.method"] = "tools/call"
	env.Metadata["mcp.tool_name"] = "search"

	err := adapter.Send(context.Background(), env)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	// Verify response envelope in inbox.
	ch, _ := adapter.Receive(context.Background())
	select {
	case respEnv := <-ch:
		if respEnv.Metadata["mcp.method"] != "tools/call" {
			t.Errorf("method = %q", respEnv.Metadata["mcp.method"])
		}
	default:
		t.Error("expected response in inbox")
	}
}

func TestAdapterSend_ToolsList(t *testing.T) {
	server := mockMCPServer(t)
	defer server.Close()

	adapter := New(nil, server.Client())
	defer adapter.Close()

	env := envelope.New("alice", "bob", protocol.ProtocolMCP, nil)
	env.Metadata["mcp.endpoint"] = server.URL
	env.Metadata["mcp.method"] = "tools/list"

	err := adapter.Send(context.Background(), env)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	ch, _ := adapter.Receive(context.Background())
	select {
	case respEnv := <-ch:
		var result ToolsListResult
		json.Unmarshal(respEnv.Payload, &result)
		if len(result.Tools) != 1 || result.Tools[0].Name != "search" {
			t.Errorf("tools = %+v", result.Tools)
		}
	default:
		t.Error("expected response in inbox")
	}
}

func TestAdapterSendMissingEndpoint(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	env := envelope.New("a", "b", protocol.ProtocolMCP, nil)
	err := adapter.Send(context.Background(), env)
	if err == nil {
		t.Error("expected error for missing endpoint")
	}
}

func TestAdapterHandshake(t *testing.T) {
	server := mockMCPServer(t)
	defer server.Close()

	adapter := New(nil, server.Client())
	defer adapter.Close()

	card := &agentcard.Card{
		ID:       "test-agent",
		Endpoint: agentcard.Endpoint{URL: server.URL},
	}

	err := adapter.Handshake(context.Background(), card)
	if err != nil {
		t.Fatalf("Handshake: %v", err)
	}

	// Verify session was stored.
	session, ok := adapter.GetSession(server.URL)
	if !ok {
		t.Fatal("session not stored")
	}
	if !session.Initialized {
		t.Error("session should be initialized")
	}
}

func TestAdapterHandshakeNoURL(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	card := &agentcard.Card{ID: "test"}
	err := adapter.Handshake(context.Background(), card)
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestAdapterTranslateToA2A(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	result := ToolCallResult{Content: []Content{{Type: "text", Text: "answer"}}}
	payload, _ := json.Marshal(result)
	env := envelope.New("a", "b", protocol.ProtocolMCP, payload)

	translated, err := adapter.Translate(context.Background(), env, string(protocol.ProtocolA2A))
	if err != nil {
		t.Fatal(err)
	}
	if translated.Protocol != protocol.ProtocolA2A {
		t.Errorf("Protocol = %q", translated.Protocol)
	}
}

func TestAdapterTranslateSame(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	env := envelope.New("a", "b", protocol.ProtocolMCP, []byte("test"))
	translated, err := adapter.Translate(context.Background(), env, "mcp")
	if err != nil {
		t.Fatal(err)
	}
	if translated != env {
		t.Error("same-protocol translate should return original")
	}
}

func TestHandleMCP_Initialize(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	params := InitializeParams{
		ProtocolVersion: "2025-11-25",
		ClientInfo:      ImplementInfo{Name: "test", Version: "1.0"},
	}
	req, _ := jsonrpc.NewRequestWithID(1, "initialize", params)
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	adapter.HandleMCP(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}

	if sid := w.Header().Get("Mcp-Session-Id"); sid == "" {
		t.Error("missing session ID header")
	}

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != nil {
		t.Fatalf("error: %v", resp.Error)
	}

	var result InitializeResult
	json.Unmarshal(resp.Result, &result)
	if result.ServerInfo.Name != "PeerClaw" {
		t.Errorf("ServerInfo.Name = %q", result.ServerInfo.Name)
	}
}

func TestHandleMCP_ToolsList(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	req, _ := jsonrpc.NewRequestWithID(1, "tools/list", nil)
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	adapter.HandleMCP(w, httpReq)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != nil {
		t.Fatalf("error: %v", resp.Error)
	}

	var result ToolsListResult
	json.Unmarshal(resp.Result, &result)
	// Empty list is valid.
	if result.Tools == nil {
		t.Error("tools should not be nil")
	}
}

func TestHandleMCP_ToolsCall(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	params := ToolCallParams{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	}
	req, _ := jsonrpc.NewRequestWithID(1, "tools/call", params)
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	adapter.HandleMCP(w, httpReq)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != nil {
		t.Fatalf("error: %v", resp.Error)
	}

	// Verify envelope was pushed to inbox.
	ch, _ := adapter.Receive(context.Background())
	select {
	case env := <-ch:
		if env.Metadata["mcp.tool_name"] != "search" {
			t.Errorf("tool_name = %q", env.Metadata["mcp.tool_name"])
		}
	default:
		t.Error("expected envelope in inbox")
	}
}

func TestHandleMCP_UnknownMethod(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	req, _ := jsonrpc.NewRequestWithID(1, "unknown/method", nil)
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	adapter.HandleMCP(w, httpReq)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error == nil {
		t.Error("expected error for unknown method")
	}
}

func TestHandleMCP_Notification(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	notif, _ := jsonrpc.NewNotification("notifications/initialized", nil)
	body, _ := json.Marshal(notif)

	httpReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	adapter.HandleMCP(w, httpReq)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestInjectMessage(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	env := envelope.New("test", "dest", protocol.ProtocolMCP, []byte("test"))
	if err := adapter.InjectMessage(env); err != nil {
		t.Fatal(err)
	}

	ch, _ := adapter.Receive(context.Background())
	select {
	case got := <-ch:
		if got.ID != env.ID {
			t.Errorf("ID = %q", got.ID)
		}
	default:
		t.Error("no message in inbox")
	}
}
