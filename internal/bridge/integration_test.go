package bridge_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/bridge"
	"github.com/peerclaw/peerclaw-server/internal/bridge/a2a"
	"github.com/peerclaw/peerclaw-server/internal/bridge/acp"
	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
	"github.com/peerclaw/peerclaw-server/internal/bridge/mcp"
	"github.com/peerclaw/peerclaw-server/internal/security"
)

func init() {
	security.AllowLocalhost = true
}

// mockA2AServer simulates an external A2A agent.
func mockA2AServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/agent.json":
			card := a2a.AgentCard{
				Name:     "mock-a2a-agent",
				Endpoint: "http://mock",
				Skills:   []a2a.Skill{{Name: "echo"}},
			}
			json.NewEncoder(w).Encode(card)
		default:
			var req jsonrpc.Request
			json.NewDecoder(r.Body).Decode(&req)

			task := a2a.Task{
				ID:        "task-1",
				ContextID: "ctx-1",
				Status: a2a.TaskStatus{
					State:     a2a.TaskStateCompleted,
					Timestamp: "2025-01-01T00:00:00Z",
				},
				Artifacts: []a2a.Artifact{
					{ID: "art-1", Parts: []a2a.Part{{Text: "echo result"}}},
				},
			}
			resp, _ := jsonrpc.NewResponse(req.ID, task)
			json.NewEncoder(w).Encode(resp)
		}
	}))
}

// mockMCPServer simulates an external MCP server.
func mockMCPServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonrpc.Request
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Mcp-Session-Id", "test-session")

		switch req.Method {
		case "initialize":
			result := mcp.InitializeResult{
				ProtocolVersion: "2025-11-25",
				Capabilities: mcp.ServerCaps{
					Tools: &mcp.ToolsCap{},
				},
				ServerInfo: mcp.ImplementInfo{Name: "mock-mcp", Version: "1.0"},
			}
			resp, _ := jsonrpc.NewResponse(req.ID, result)
			json.NewEncoder(w).Encode(resp)
		case "notifications/initialized":
			w.WriteHeader(http.StatusOK)
		case "tools/call":
			result := mcp.ToolCallResult{
				Content: []mcp.Content{{Type: "text", Text: "tool result"}},
			}
			resp, _ := jsonrpc.NewResponse(req.ID, result)
			json.NewEncoder(w).Encode(resp)
		default:
			resp := jsonrpc.NewErrorResponse(req.ID, jsonrpc.CodeMethodNotFound, "unknown")
			json.NewEncoder(w).Encode(resp)
		}
	}))
}

// mockACPServer simulates an external ACP agent.
func mockACPServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/agents/echo":
			manifest := acp.AgentManifest{
				Name:              "echo",
				InputContentTypes: []string{"text/plain"},
			}
			json.NewEncoder(w).Encode(manifest)
		case r.URL.Path == "/runs":
			run := acp.Run{
				RunID:     "run-1",
				AgentName: "echo",
				SessionID: "sess-1",
				Status:    acp.RunStatusCompleted,
				Output: []acp.Message{
					{Role: "agent", Parts: []acp.MessagePart{{ContentType: "text/plain", Content: "echo done"}}},
				},
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(run)
		}
	}))
}

func TestIntegration_A2ASendAndReceive(t *testing.T) {
	server := mockA2AServer(t)
	defer server.Close()

	a2aAdapter := a2a.New(nil, server.Client())
	defer a2aAdapter.Close()

	mgr := bridge.NewManager(nil)
	mgr.RegisterBridge(a2aAdapter)

	// Send an envelope to the mock A2A agent.
	env := envelope.New("peerclaw", "external-a2a", protocol.ProtocolA2A, []byte("hello"))
	env.Metadata["a2a.endpoint"] = server.URL
	env.Metadata["a2a.context_id"] = "ctx-1"

	err := mgr.Send(context.Background(), env)
	if err != nil {
		t.Fatalf("A2A Send: %v", err)
	}

	// Verify response came back through inbox.
	ch, _ := a2aAdapter.Receive(context.Background())
	select {
	case resp := <-ch:
		if resp.Metadata["a2a.task_id"] != "task-1" {
			t.Errorf("task_id = %q", resp.Metadata["a2a.task_id"])
		}
		if resp.MessageType != envelope.MessageTypeResponse {
			t.Errorf("type = %q", resp.MessageType)
		}
	default:
		t.Error("expected response in A2A inbox")
	}
}

func TestIntegration_MCPSendAndReceive(t *testing.T) {
	server := mockMCPServer(t)
	defer server.Close()

	mcpAdapter := mcp.New(nil, server.Client())
	defer mcpAdapter.Close()

	mgr := bridge.NewManager(nil)
	mgr.RegisterBridge(mcpAdapter)

	args, _ := json.Marshal(map[string]string{"query": "test"})
	env := envelope.New("peerclaw", "external-mcp", protocol.ProtocolMCP, args)
	env.Metadata["mcp.endpoint"] = server.URL
	env.Metadata["mcp.method"] = "tools/call"
	env.Metadata["mcp.tool_name"] = "search"

	err := mgr.Send(context.Background(), env)
	if err != nil {
		t.Fatalf("MCP Send: %v", err)
	}

	ch, _ := mcpAdapter.Receive(context.Background())
	select {
	case resp := <-ch:
		if resp.MessageType != envelope.MessageTypeResponse {
			t.Errorf("type = %q", resp.MessageType)
		}
		var result mcp.ToolCallResult
		json.Unmarshal(resp.Payload, &result)
		if len(result.Content) == 0 || result.Content[0].Text != "tool result" {
			t.Errorf("result = %+v", result)
		}
	default:
		t.Error("expected response in MCP inbox")
	}
}

func TestIntegration_ACPSendAndReceive(t *testing.T) {
	server := mockACPServer(t)
	defer server.Close()

	acpAdapter := acp.New(nil, server.Client())
	defer acpAdapter.Close()

	mgr := bridge.NewManager(nil)
	mgr.RegisterBridge(acpAdapter)

	env := envelope.New("peerclaw", "echo", protocol.ProtocolACP, []byte("hello"))
	env.Metadata["acp.endpoint"] = server.URL
	env.Metadata["acp.agent_name"] = "echo"

	err := mgr.Send(context.Background(), env)
	if err != nil {
		t.Fatalf("ACP Send: %v", err)
	}

	ch, _ := acpAdapter.Receive(context.Background())
	select {
	case resp := <-ch:
		if resp.Metadata["acp.run_id"] != "run-1" {
			t.Errorf("run_id = %q", resp.Metadata["acp.run_id"])
		}
	default:
		t.Error("expected response in ACP inbox")
	}
}

func TestIntegration_InboundA2A(t *testing.T) {
	a2aAdapter := a2a.New(nil, nil)
	defer a2aAdapter.Close()

	// Simulate inbound SendMessage via handler.
	params := a2a.SendMessageParams{
		Message: a2a.Message{
			Role:  "user",
			Parts: []a2a.Part{{Text: "inbound test"}},
		},
	}
	req, _ := jsonrpc.NewRequestWithID(1, "message/send", params)
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest(http.MethodPost, "/a2a", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	a2aAdapter.HandleMessages(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}

	// Verify envelope in inbox.
	ch, _ := a2aAdapter.Receive(context.Background())
	select {
	case env := <-ch:
		if env.Protocol != protocol.ProtocolA2A {
			t.Errorf("protocol = %q", env.Protocol)
		}
	default:
		t.Error("expected envelope from inbound A2A handler")
	}
}

func TestIntegration_InboundMCP(t *testing.T) {
	mcpAdapter := mcp.New(nil, nil)
	defer mcpAdapter.Close()

	params := mcp.ToolCallParams{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	}
	req, _ := jsonrpc.NewRequestWithID(1, "tools/call", params)
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	mcpAdapter.HandleMCP(w, httpReq)

	ch, _ := mcpAdapter.Receive(context.Background())
	select {
	case env := <-ch:
		if env.Metadata["mcp.tool_name"] != "search" {
			t.Errorf("tool_name = %q", env.Metadata["mcp.tool_name"])
		}
	default:
		t.Error("expected envelope from inbound MCP handler")
	}
}

func TestIntegration_InboundACP(t *testing.T) {
	acpAdapter := acp.New(nil, nil)
	defer acpAdapter.Close()

	createReq := acp.CreateRunRequest{
		AgentName: "echo",
		Input: []acp.Message{
			{Role: "user", Parts: []acp.MessagePart{{ContentType: "text/plain", Content: "hello"}}},
		},
		Mode: "sync",
	}
	body, _ := json.Marshal(createReq)

	httpReq := httptest.NewRequest(http.MethodPost, "/acp/runs", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	acpAdapter.HandleCreateRun(w, httpReq)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d", w.Code)
	}

	ch, _ := acpAdapter.Receive(context.Background())
	select {
	case env := <-ch:
		if env.Metadata["acp.agent_name"] != "echo" {
			t.Errorf("agent_name = %q", env.Metadata["acp.agent_name"])
		}
	default:
		t.Error("expected envelope from inbound ACP handler")
	}
}

func TestIntegration_CrossProtocolTranslation_A2AToMCP(t *testing.T) {
	a2aAdapter := a2a.New(nil, nil)
	defer a2aAdapter.Close()

	mgr := bridge.NewManager(nil)
	mgr.RegisterBridge(a2aAdapter)

	msg := a2a.Message{
		Role:  "user",
		Parts: []a2a.Part{{Text: "translate me"}},
	}
	payload, _ := json.Marshal(msg)
	env := envelope.New("alice", "bob", protocol.ProtocolA2A, payload)
	env.Metadata["mcp.tool_name"] = "analyze"

	translated, err := mgr.Translate(context.Background(), env, string(protocol.ProtocolMCP))
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}

	if translated.Protocol != protocol.ProtocolMCP {
		t.Errorf("Protocol = %q", translated.Protocol)
	}
	if translated.Metadata["mcp.method"] != "tools/call" {
		t.Errorf("mcp.method = %q", translated.Metadata["mcp.method"])
	}
}

func TestIntegration_CrossProtocolTranslation_MCPToA2A(t *testing.T) {
	mcpAdapter := mcp.New(nil, nil)
	defer mcpAdapter.Close()

	result := mcp.ToolCallResult{
		Content: []mcp.Content{{Type: "text", Text: "result"}},
	}
	payload, _ := json.Marshal(result)
	env := envelope.New("server", "client", protocol.ProtocolMCP, payload)

	translated, err := mcpAdapter.Translate(context.Background(), env, string(protocol.ProtocolA2A))
	if err != nil {
		t.Fatal(err)
	}
	if translated.Protocol != protocol.ProtocolA2A {
		t.Errorf("Protocol = %q", translated.Protocol)
	}
}

