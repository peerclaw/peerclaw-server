package acp

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
)

func TestAdapterProtocol(t *testing.T) {
	a := New(nil, nil)
	defer a.Close()
	if a.Protocol() != "acp" {
		t.Errorf("Protocol() = %q", a.Protocol())
	}
}

func TestAdapterSend(t *testing.T) {
	run := &Run{
		RunID:     "run-1",
		AgentName: "echo",
		SessionID: "sess-1",
		Status:    RunStatusCompleted,
		Output: []Message{
			{Role: "agent", Parts: []MessagePart{{ContentType: "text/plain", Content: "done"}}},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/runs") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req CreateRunRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.AgentName != "echo" {
			t.Errorf("AgentName = %q", req.AgentName)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(run)
	}))
	defer server.Close()

	adapter := New(nil, server.Client())
	defer adapter.Close()

	env := envelope.New("alice", "echo", protocol.ProtocolACP, []byte("do something"))
	env.Metadata["acp.endpoint"] = server.URL
	env.Metadata["acp.agent_name"] = "echo"

	err := adapter.Send(context.Background(), env)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	// Verify run was stored.
	stored, ok := adapter.GetRun("run-1")
	if !ok {
		t.Fatal("run not stored")
	}
	if stored.Status != RunStatusCompleted {
		t.Errorf("stored status = %q", stored.Status)
	}

	// Verify response envelope was pushed to inbox.
	ch, _ := adapter.Receive(context.Background())
	select {
	case respEnv := <-ch:
		if respEnv.Metadata["acp.run_id"] != "run-1" {
			t.Errorf("response run_id = %q", respEnv.Metadata["acp.run_id"])
		}
	default:
		t.Error("expected response in inbox")
	}
}

func TestAdapterSendMissingEndpoint(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	env := envelope.New("a", "b", protocol.ProtocolACP, []byte("test"))
	err := adapter.Send(context.Background(), env)
	if err == nil {
		t.Error("expected error for missing endpoint")
	}
}

func TestAdapterHandshake(t *testing.T) {
	manifest := AgentManifest{
		Name:        "echo",
		Description: "Echo agent",
		Metadata: ManifestMetadata{
			Capabilities: []CapabilityDef{{Name: "echo"}},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/agents/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	adapter := New(nil, server.Client())
	defer adapter.Close()

	card := &agentcard.Card{
		ID:       "echo-id",
		Name:     "echo",
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

func TestAdapterTranslateToA2A(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	msg := Message{
		Role:  "agent",
		Parts: []MessagePart{{ContentType: "text/plain", Content: "result"}},
	}
	payload, _ := json.Marshal(msg)
	env := envelope.New("a", "b", protocol.ProtocolACP, payload)

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

	env := envelope.New("a", "b", protocol.ProtocolACP, []byte("test"))
	translated, err := adapter.Translate(context.Background(), env, "acp")
	if err != nil {
		t.Fatal(err)
	}
	if translated != env {
		t.Error("same-protocol translate should return original")
	}
}

func TestHandleListAgents(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	httpReq := httptest.NewRequest(http.MethodGet, "/acp/agents", nil)
	w := httptest.NewRecorder()

	adapter.HandleListAgents(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}

	var result map[string][]AgentManifest
	json.NewDecoder(w.Body).Decode(&result)
	if result["agents"] == nil {
		t.Error("agents should not be nil")
	}
}

func TestHandleGetAgent(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	httpReq := httptest.NewRequest(http.MethodGet, "/acp/agents/echo", nil)
	httpReq.SetPathValue("name", "echo")
	w := httptest.NewRecorder()

	adapter.HandleGetAgent(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}

	var manifest AgentManifest
	json.NewDecoder(w.Body).Decode(&manifest)
	if manifest.Name != "echo" {
		t.Errorf("Name = %q", manifest.Name)
	}
}

func TestHandleCreateRun(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	req := CreateRunRequest{
		AgentName: "echo",
		Input: []Message{
			{Role: "user", Parts: []MessagePart{{ContentType: "text/plain", Content: "hello"}}},
		},
		Mode: "sync",
	}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest(http.MethodPost, "/acp/runs", strings.NewReader(string(body)))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.HandleCreateRun(w, httpReq)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d", w.Code)
	}

	var run Run
	json.NewDecoder(w.Body).Decode(&run)
	if run.RunID == "" {
		t.Error("RunID should not be empty")
	}
	if run.Status != RunStatusCreated {
		t.Errorf("Status = %q", run.Status)
	}

	// Verify envelope was pushed to inbox.
	ch, _ := adapter.Receive(context.Background())
	select {
	case env := <-ch:
		if env.Metadata["acp.run_id"] != run.RunID {
			t.Errorf("inbox run_id = %q", env.Metadata["acp.run_id"])
		}
	default:
		t.Error("expected envelope in inbox")
	}
}

func TestHandleCreateRun_MissingAgent(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	req := CreateRunRequest{Input: []Message{}}
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest(http.MethodPost, "/acp/runs", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	adapter.HandleCreateRun(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleGetRun(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	run := &Run{RunID: "run-1", Status: RunStatusCompleted, AgentName: "echo"}
	adapter.runs.Store("run-1", run)

	httpReq := httptest.NewRequest(http.MethodGet, "/acp/runs/run-1", nil)
	httpReq.SetPathValue("run_id", "run-1")
	w := httptest.NewRecorder()

	adapter.HandleGetRun(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}

	var got Run
	json.NewDecoder(w.Body).Decode(&got)
	if got.RunID != "run-1" {
		t.Errorf("RunID = %q", got.RunID)
	}
}

func TestHandleGetRun_NotFound(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	httpReq := httptest.NewRequest(http.MethodGet, "/acp/runs/unknown", nil)
	httpReq.SetPathValue("run_id", "unknown")
	w := httptest.NewRecorder()

	adapter.HandleGetRun(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleCancelRun(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	run := &Run{RunID: "run-1", Status: RunStatusInProgress}
	adapter.runs.Store("run-1", run)

	httpReq := httptest.NewRequest(http.MethodPost, "/acp/runs/run-1/cancel", nil)
	httpReq.SetPathValue("run_id", "run-1")
	w := httptest.NewRecorder()

	adapter.HandleCancelRun(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}

	var got Run
	json.NewDecoder(w.Body).Decode(&got)
	if got.Status != RunStatusCancelled {
		t.Errorf("Status = %q, want cancelled", got.Status)
	}
}

func TestHandlePing(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	httpReq := httptest.NewRequest(http.MethodGet, "/acp/ping", nil)
	w := httptest.NewRecorder()

	adapter.HandlePing(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
}

func TestInjectMessage(t *testing.T) {
	adapter := New(nil, nil)
	defer adapter.Close()

	env := envelope.New("test", "dest", protocol.ProtocolACP, []byte("test"))
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
