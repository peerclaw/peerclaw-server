package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/peerclaw/peerclaw-server/internal/security"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
)

func init() {
	security.AllowLocalhost = true
}

func TestAdapterProtocol(t *testing.T) {
	a := New(nil, nil)
	defer func() { _ = a.Close() }()
	if a.Protocol() != "a2a" {
		t.Errorf("Protocol() = %q", a.Protocol())
	}
}

func TestAdapterSend(t *testing.T) {
	// Mock A2A server.
	task := &Task{
		ID:        "task-1",
		ContextID: "ctx-1",
		Status: TaskStatus{
			State:     TaskStateCompleted,
			Timestamp: "2025-01-01T00:00:00Z",
		},
		Artifacts: []Artifact{
			{ID: "art-1", Parts: []Part{{Text: "result"}}},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected json content type")
		}

		// Parse and validate JSON-RPC request.
		var req jsonrpc.Request
		json.NewDecoder(r.Body).Decode(&req)
		if req.Method != "message/send" {
			t.Errorf("method = %q", req.Method)
		}

		resp, _ := jsonrpc.NewResponse(req.ID, task)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := New(nil, server.Client())
	defer adapter.Close()

	env := envelope.New("alice", "bob", protocol.ProtocolA2A, []byte("hello"))
	env.Metadata["a2a.endpoint"] = server.URL
	env.Metadata["a2a.context_id"] = "ctx-1"

	err := adapter.Send(context.Background(), env)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	// Verify task was stored.
	stored, ok := adapter.GetTask("task-1")
	if !ok {
		t.Fatal("task not stored")
	}
	if stored.Status.State != TaskStateCompleted {
		t.Errorf("stored state = %q", stored.Status.State)
	}

	// Verify response envelope was pushed to inbox.
	ch, _ := adapter.Receive(context.Background())
	select {
	case respEnv := <-ch:
		if respEnv.Metadata["a2a.task_id"] != "task-1" {
			t.Errorf("response task_id = %q", respEnv.Metadata["a2a.task_id"])
		}
	default:
		t.Error("expected response envelope in inbox")
	}
}

func TestAdapterSendMissingEndpoint(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	env := envelope.New("alice", "bob", protocol.ProtocolA2A, []byte("hello"))
	err := adapter.Send(context.Background(), env)
	if err == nil {
		t.Fatal("expected error for missing endpoint")
	}
}

func TestAdapterHandshake(t *testing.T) {
	agentCard := AgentCard{
		Name:     "remote-agent",
		Endpoint: "https://example.com",
		Capabilities: A2ACaps{
			Streaming: true,
		},
		Skills: []Skill{
			{Name: "summarize"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/agent.json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(agentCard)
	}))
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

func TestAdapterTranslateToMCP(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	msg := Message{Role: "user", Parts: []Part{{Text: "analyze this"}}}
	payload, _ := json.Marshal(msg)
	env := envelope.New("alice", "bob", protocol.ProtocolA2A, payload)
	env.Metadata["mcp.tool_name"] = "analyze"

	translated, err := adapter.Translate(context.Background(), env, string(protocol.ProtocolMCP))
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

func TestAdapterTranslateSameProtocol(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	env := envelope.New("a", "b", protocol.ProtocolA2A, []byte("test"))
	translated, err := adapter.Translate(context.Background(), env, "a2a")
	if err != nil {
		t.Fatal(err)
	}
	if translated != env {
		t.Error("same-protocol translate should return original")
	}
}

func TestHandleMessages_SendMessage(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	params := SendMessageParams{
		Message: Message{
			Role:      "user",
			Parts:     []Part{{Text: "hello"}},
			ContextID: "ctx-1",
		},
	}
	req, _ := jsonrpc.NewRequestWithID(1, "message/send", params)
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest(http.MethodPost, "/a2a", strings.NewReader(string(body)))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.HandleMessages(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var task Task
	json.Unmarshal(resp.Result, &task)
	if task.ID == "" {
		t.Error("task ID should not be empty")
	}
	if task.Status.State != TaskStateAccepted {
		t.Errorf("state = %q", task.Status.State)
	}

	// Verify envelope was pushed to inbox.
	ch, _ := adapter.Receive(context.Background())
	select {
	case env := <-ch:
		if env.Metadata["a2a.task_id"] != task.ID {
			t.Errorf("inbox task_id = %q", env.Metadata["a2a.task_id"])
		}
	default:
		t.Error("expected envelope in inbox")
	}
}

func TestHandleMessages_GetTask(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	// Pre-store a task.
	task := &Task{
		ID:     "task-1",
		Status: TaskStatus{State: TaskStateWorking},
	}
	adapter.tasks.Store("task-1", task)

	req, _ := jsonrpc.NewRequestWithID(1, "tasks/get", GetTaskParams{ID: "task-1"})
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest(http.MethodPost, "/a2a", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	adapter.HandleMessages(w, httpReq)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != nil {
		t.Fatalf("error: %v", resp.Error)
	}

	var got Task
	json.Unmarshal(resp.Result, &got)
	if got.ID != "task-1" {
		t.Errorf("task ID = %q", got.ID)
	}
}

func TestHandleMessages_CancelTask(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	task := &Task{
		ID:     "task-1",
		Status: TaskStatus{State: TaskStateWorking},
	}
	adapter.tasks.Store("task-1", task)

	req, _ := jsonrpc.NewRequestWithID(1, "tasks/cancel", CancelTaskParams{ID: "task-1"})
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest(http.MethodPost, "/a2a", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	adapter.HandleMessages(w, httpReq)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != nil {
		t.Fatalf("error: %v", resp.Error)
	}

	var got Task
	json.Unmarshal(resp.Result, &got)
	if got.Status.State != TaskStateCanceled {
		t.Errorf("state = %q, want canceled", got.Status.State)
	}
}

func TestHandleMessages_UnknownMethod(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	req, _ := jsonrpc.NewRequestWithID(1, "unknown/method", nil)
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest(http.MethodPost, "/a2a", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	adapter.HandleMessages(w, httpReq)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error == nil {
		t.Error("expected error for unknown method")
	}
	if resp.Error.Code != jsonrpc.CodeMethodNotFound {
		t.Errorf("code = %d", resp.Error.Code)
	}
}

func TestHandleAgentCard(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	httpReq := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
	w := httptest.NewRecorder()

	adapter.HandleAgentCard(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}

	var card AgentCard
	json.NewDecoder(w.Body).Decode(&card)
	if card.Name != "PeerClaw Gateway" {
		t.Errorf("Name = %q", card.Name)
	}
}

func TestHandleGetTask_REST(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	task := &Task{
		ID:     "task-1",
		Status: TaskStatus{State: TaskStateCompleted},
	}
	adapter.tasks.Store("task-1", task)

	httpReq := httptest.NewRequest(http.MethodGet, "/a2a/tasks/task-1", nil)
	httpReq.SetPathValue("id", "task-1")
	w := httptest.NewRecorder()

	adapter.HandleGetTask(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}

	var got Task
	json.NewDecoder(w.Body).Decode(&got)
	if got.ID != "task-1" {
		t.Errorf("task ID = %q", got.ID)
	}
}

func TestHandleGetTask_NotFound(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	httpReq := httptest.NewRequest(http.MethodGet, "/a2a/tasks/unknown", nil)
	httpReq.SetPathValue("id", "unknown")
	w := httptest.NewRecorder()

	adapter.HandleGetTask(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestTaskLifecycle(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	// 1. Create task via SendMessage.
	params := SendMessageParams{
		Message: Message{
			Role:  "user",
			Parts: []Part{{Text: "process this"}},
		},
	}
	req, _ := jsonrpc.NewRequestWithID(1, "message/send", params)
	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/a2a", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	adapter.HandleMessages(w, httpReq)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)
	var task Task
	json.Unmarshal(resp.Result, &task)

	// 2. Verify initial state.
	if task.Status.State != TaskStateAccepted {
		t.Errorf("initial state = %q", task.Status.State)
	}

	// 3. Get task.
	getReq, _ := jsonrpc.NewRequestWithID(2, "tasks/get", GetTaskParams{ID: task.ID})
	getBody, _ := json.Marshal(getReq)
	httpReq2 := httptest.NewRequest(http.MethodPost, "/a2a", strings.NewReader(string(getBody)))
	w2 := httptest.NewRecorder()
	adapter.HandleMessages(w2, httpReq2)

	var resp2 jsonrpc.Response
	json.NewDecoder(w2.Body).Decode(&resp2)
	var gotTask Task
	json.Unmarshal(resp2.Result, &gotTask)
	if gotTask.ID != task.ID {
		t.Errorf("get task ID = %q", gotTask.ID)
	}

	// 4. Cancel task.
	cancelReq, _ := jsonrpc.NewRequestWithID(3, "tasks/cancel", CancelTaskParams{ID: task.ID})
	cancelBody, _ := json.Marshal(cancelReq)
	httpReq3 := httptest.NewRequest(http.MethodPost, "/a2a", strings.NewReader(string(cancelBody)))
	w3 := httptest.NewRecorder()
	adapter.HandleMessages(w3, httpReq3)

	var resp3 jsonrpc.Response
	json.NewDecoder(w3.Body).Decode(&resp3)
	var canceled Task
	json.Unmarshal(resp3.Result, &canceled)
	if canceled.Status.State != TaskStateCanceled {
		t.Errorf("canceled state = %q", canceled.Status.State)
	}
}

func TestInjectMessage(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	env := envelope.New("test", "dest", protocol.ProtocolA2A, []byte("test"))
	if err := adapter.InjectMessage(env); err != nil {
		t.Fatalf("InjectMessage: %v", err)
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
