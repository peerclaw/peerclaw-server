package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/peerclaw/peerclaw-server/internal/bridge"
	"github.com/peerclaw/peerclaw-server/internal/bridge/a2a"
	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
	"github.com/peerclaw/peerclaw-server/internal/registry"
	"github.com/peerclaw/peerclaw-server/internal/router"
	"github.com/peerclaw/peerclaw-server/internal/signaling"
)

// newTestHTTPServerWithACL creates a test server with playground_enabled column.
func newTestHTTPServerWithACL(t *testing.T) *HTTPServer {
	t.Helper()
	store, err := registry.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	// Add ACL columns that are normally added by migration.
	if db, ok := store.GetDB().(*sql.DB); ok {
		_, _ = db.Exec("ALTER TABLE agents ADD COLUMN playground_enabled INTEGER DEFAULT 0")
		_, _ = db.Exec("ALTER TABLE agents ADD COLUMN visibility TEXT DEFAULT 'public'")
	}

	reg := registry.NewService(store, nil)
	table := router.NewTable()
	eng := router.NewEngine(table, nil)
	brg := bridge.NewManager(nil)
	sigHub := signaling.NewHub(nil, nil, 0)

	return NewHTTPServer(":0", reg, eng, brg, sigHub, nil, nil)
}

// registerTestAgent registers a test agent and enables playground access.
func registerTestAgent(t *testing.T, s *HTTPServer) string {
	t.Helper()
	body := `{
		"name": "TestA2AAgent",
		"description": "A test agent for A2A bridge",
		"version": "1.0.0",
		"capabilities": ["chat"],
		"endpoint": {"url": "http://localhost:3000"},
		"protocols": ["a2a"]
	}`
	req := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var card map[string]any
	json.NewDecoder(w.Body).Decode(&card)
	agentID := card["id"].(string)

	// Enable playground access.
	err := s.registry.SetAccessFlags(req.Context(), agentID, &registry.AccessFlags{
		PlaygroundEnabled: true,
		Visibility:        "public",
	})
	if err != nil {
		t.Fatalf("failed to set access flags: %v", err)
	}

	return agentID
}

func TestA2ABridge_AgentCard(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	req := httptest.NewRequest("GET", "/a2a/"+agentID+"/.well-known/agent.json", nil)
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
	if !card.Capabilities.Streaming {
		t.Error("capabilities.streaming should be true")
	}
	if !card.Capabilities.MultiTurn {
		t.Error("capabilities.multiTurn should be true")
	}
}

func TestA2ABridge_AgentCard_NotFound(t *testing.T) {
	s := newTestHTTPServer(t)

	req := httptest.NewRequest("GET", "/a2a/nonexistent/.well-known/agent.json", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestA2ABridge_SendMessage(t *testing.T) {
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
					{"text": "hello"},
				},
			},
		},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/a2a/"+agentID, bytes.NewBuffer(body))
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
		// With no real bridge, we expect the task to be created but fail.
		// The bridge.SendStream will fail since there's no actual adapter connected.
		// Check we still get a valid JSON-RPC response.
		t.Logf("expected bridge error: %s", resp.Error.Message)
	}

	if resp.Result != nil {
		var task a2a.Task
		if err := json.Unmarshal(resp.Result, &task); err != nil {
			t.Fatalf("unmarshal task: %v", err)
		}
		if task.ID == "" {
			t.Error("task ID should not be empty")
		}
	}
}

func TestA2ABridge_SendMessage_NotFound(t *testing.T) {
	s := newTestHTTPServer(t)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "message/send",
		"params": map[string]any{
			"message": map[string]any{
				"role": "user",
				"parts": []map[string]any{
					{"text": "hello"},
				},
			},
		},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/a2a/nonexistent", bytes.NewBuffer(body))
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

func TestA2ABridge_GetTask(t *testing.T) {
	s := newTestHTTPServer(t)

	// Manually create a task.
	task := a2a.NewTask("ctx-1", a2a.Message{
		Role:  "user",
		Parts: []a2a.Part{{Text: "test"}},
	})
	s.a2aTasks.tasks.Store(task.ID, task)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tasks/get",
		"params": map[string]any{
			"id": task.ID,
		},
	}
	body, _ := json.Marshal(rpcReq)

	// agent_id is required in the path but not used by tasks/get.
	req := httptest.NewRequest("POST", "/a2a/any-agent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var got a2a.Task
	json.Unmarshal(resp.Result, &got)
	if got.ID != task.ID {
		t.Errorf("task ID = %q, want %q", got.ID, task.ID)
	}
}

func TestA2ABridge_CancelTask(t *testing.T) {
	s := newTestHTTPServer(t)

	task := a2a.NewTask("ctx-1", a2a.Message{
		Role:  "user",
		Parts: []a2a.Part{{Text: "test"}},
	})
	s.a2aTasks.tasks.Store(task.ID, task)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tasks/cancel",
		"params": map[string]any{
			"id": task.ID,
		},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/a2a/any-agent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var resp jsonrpc.Response
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var got a2a.Task
	json.Unmarshal(resp.Result, &got)
	if got.Status.State != a2a.TaskStateCanceled {
		t.Errorf("state = %q, want %q", got.Status.State, a2a.TaskStateCanceled)
	}
}

func TestA2ABridge_UnknownMethod(t *testing.T) {
	s := newTestHTTPServer(t)

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "foo/bar",
		"params":  map[string]any{},
	}
	body, _ := json.Marshal(rpcReq)

	req := httptest.NewRequest("POST", "/a2a/any-agent", bytes.NewBuffer(body))
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

func TestA2ABridge_InvalidJSON(t *testing.T) {
	s := newTestHTTPServer(t)

	req := httptest.NewRequest("POST", "/a2a/any-agent", bytes.NewBufferString("{invalid json"))
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

func TestA2ABridge_GetTaskREST(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	task := a2a.NewTask("ctx-2", a2a.Message{
		Role:  "user",
		Parts: []a2a.Part{{Text: "rest test"}},
	})
	s.a2aTasks.tasks.Store(task.ID, task)

	req := httptest.NewRequest("GET", "/a2a/"+agentID+"/tasks/"+task.ID, nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var got a2a.Task
	json.NewDecoder(w.Body).Decode(&got)
	if got.ID != task.ID {
		t.Errorf("task ID = %q, want %q", got.ID, task.ID)
	}
}

func TestA2ABridge_GetTaskREST_NotFound(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	req := httptest.NewRequest("GET", "/a2a/"+agentID+"/tasks/nonexistent", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestA2ABridgeTasks_ConcurrentAccess(t *testing.T) {
	tasks := &a2aBridgeTasks{}
	const goroutines = 10
	const opsPerGoroutine = 100

	// Run concurrent writes, reads, and deletes on the task store.
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Writers: store tasks.
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				task := a2a.NewTask("ctx", a2a.Message{
					Role:  "user",
					Parts: []a2a.Part{{Text: "concurrent test"}},
				})
				tasks.tasks.Store(task.ID, task)
				tasks.taskCount.Add(1)
				tasks.creatorIPs.Store(task.ID, "127.0.0.1")
			}
		}(g)
	}

	// Readers: iterate and load tasks.
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				tasks.tasks.Range(func(key, value any) bool {
					_ = value.(*a2a.Task).ID
					return true
				})
				_ = tasks.taskCount.Load()
			}
		}()
	}

	// Deleters: try to delete tasks found during iteration.
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				tasks.tasks.Range(func(key, value any) bool {
					tasks.tasks.Delete(key)
					tasks.creatorIPs.Delete(key)
					tasks.taskCount.Add(-1)
					return false // delete one per iteration
				})
			}
		}()
	}

	wg.Wait()
	// No panic or race = success.
}

func TestA2ABridgeCleanup_ExitsOnCancel(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	s.a2aTasks = &a2aBridgeTasks{}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.a2aBridgeCleanup(ctx, 50*time.Millisecond, time.Hour)
		close(done)
	}()

	// Let the cleanup goroutine start and tick at least once.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Goroutine exited — success.
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup goroutine did not exit after context cancellation")
	}
}
