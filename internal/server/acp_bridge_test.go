package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/peerclaw/peerclaw-server/internal/bridge/acp"
)

func TestACPBridge_AgentManifest(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	req := httptest.NewRequest("GET", "/acp/"+agentID+"/manifest", nil)
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
	if len(manifest.InputContentTypes) == 0 {
		t.Error("input_content_types should not be empty")
	}
}

func TestACPBridge_AgentManifest_NotFound(t *testing.T) {
	s := newTestHTTPServerWithACL(t)

	req := httptest.NewRequest("GET", "/acp/nonexistent/manifest", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestACPBridge_CreateRun(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	body, _ := json.Marshal(acp.CreateRunRequest{
		Input: []acp.Message{
			{
				Role: "user",
				Parts: []acp.MessagePart{
					{ContentType: "text/plain", Content: "hello"},
				},
			},
		},
	})

	req := httptest.NewRequest("POST", "/acp/"+agentID+"/runs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var run acp.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if run.RunID == "" {
		t.Error("run_id should not be empty")
	}
	if run.AgentName != "TestA2AAgent" {
		t.Errorf("agent_name = %q, want %q", run.AgentName, "TestA2AAgent")
	}
	// With no real bridge, the run should either fail or complete.
	// We just verify we got a valid run back.
}

func TestACPBridge_CreateRun_NoAgent(t *testing.T) {
	s := newTestHTTPServerWithACL(t)

	body, _ := json.Marshal(acp.CreateRunRequest{
		Input: []acp.Message{
			{
				Role: "user",
				Parts: []acp.MessagePart{
					{ContentType: "text/plain", Content: "hello"},
				},
			},
		},
	})

	req := httptest.NewRequest("POST", "/acp/nonexistent/runs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestACPBridge_CreateRun_MissingInput(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	body, _ := json.Marshal(acp.CreateRunRequest{
		Input: []acp.Message{},
	})

	req := httptest.NewRequest("POST", "/acp/"+agentID+"/runs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestACPBridge_GetRun(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	// Manually create a run.
	run := acp.NewRun(acp.CreateRunRequest{
		AgentName: "test",
		Input: []acp.Message{
			{Role: "user", Parts: []acp.MessagePart{{ContentType: "text/plain", Content: "test"}}},
		},
	})
	s.acpRuns.runs.Store(run.RunID, run)

	req := httptest.NewRequest("GET", "/acp/"+agentID+"/runs/"+run.RunID, nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var got acp.Run
	json.NewDecoder(w.Body).Decode(&got)
	if got.RunID != run.RunID {
		t.Errorf("run_id = %q, want %q", got.RunID, run.RunID)
	}
}

func TestACPBridge_GetRun_NotFound(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	req := httptest.NewRequest("GET", "/acp/"+agentID+"/runs/nonexistent", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestACPBridge_CancelRun(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	// Manually create a run.
	run := acp.NewRun(acp.CreateRunRequest{
		AgentName: "test",
		Input: []acp.Message{
			{Role: "user", Parts: []acp.MessagePart{{ContentType: "text/plain", Content: "test"}}},
		},
	})
	s.acpRuns.runs.Store(run.RunID, run)

	req := httptest.NewRequest("POST", "/acp/"+agentID+"/runs/"+run.RunID+"/cancel", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var got acp.Run
	json.NewDecoder(w.Body).Decode(&got)
	if got.Status != acp.RunStatusCancelled {
		t.Errorf("status = %q, want %q", got.Status, acp.RunStatusCancelled)
	}
}

func TestACPBridge_CancelRun_NotFound(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	req := httptest.NewRequest("POST", "/acp/"+agentID+"/runs/nonexistent/cancel", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestACPBridge_Ping(t *testing.T) {
	s := newTestHTTPServerWithACL(t)
	agentID := registerTestAgent(t, s)

	req := httptest.NewRequest("GET", "/acp/"+agentID+"/ping", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}
