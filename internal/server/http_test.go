package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/peerclaw/peerclaw-server/internal/bridge"
	"github.com/peerclaw/peerclaw-server/internal/registry"
	"github.com/peerclaw/peerclaw-server/internal/router"
	"github.com/peerclaw/peerclaw-server/internal/signaling"
)

func newTestHTTPServer(t *testing.T) *HTTPServer {
	t.Helper()
	store, err := registry.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	reg := registry.NewService(store, nil)
	table := router.NewTable()
	eng := router.NewEngine(table, nil)
	brg := bridge.NewManager(nil)
	sigHub := signaling.NewHub(nil, nil, 0)

	return NewHTTPServer(":0", reg, eng, brg, sigHub, nil, nil)
}

func TestHTTP_Health(t *testing.T) {
	s := newTestHTTPServer(t)
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHTTP_RegisterAndGetAgent(t *testing.T) {
	s := newTestHTTPServer(t)

	body := `{
		"name": "TestAgent",
		"description": "A test agent",
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

	// Get the agent.
	req = httptest.NewRequest("GET", "/api/v1/agents/"+agentID, nil)
	w = httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("get status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHTTP_ListAgents(t *testing.T) {
	s := newTestHTTPServer(t)

	// Register an agent first.
	body := `{"name": "Agent1", "endpoint": {"url": "http://localhost:3000"}, "protocols": ["a2a"]}`
	req := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	// List agents.
	req = httptest.NewRequest("GET", "/api/v1/agents", nil)
	w = httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)
	count := int(result["total_count"].(float64))
	if count != 1 {
		t.Errorf("TotalCount = %d, want 1", count)
	}
}

func TestHTTP_DeleteAgent(t *testing.T) {
	s := newTestHTTPServer(t)

	body := `{"name": "Agent1", "endpoint": {"url": "http://localhost:3000"}, "protocols": ["a2a"]}`
	req := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var card map[string]any
	json.NewDecoder(w.Body).Decode(&card)
	agentID := card["id"].(string)

	// Delete.
	req = httptest.NewRequest("DELETE", "/api/v1/agents/"+agentID, nil)
	w = httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestHTTP_Register_BadRequest(t *testing.T) {
	s := newTestHTTPServer(t)

	req := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHTTP_GetRoutes(t *testing.T) {
	s := newTestHTTPServer(t)

	req := httptest.NewRequest("GET", "/api/v1/routes", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHTTP_Discover(t *testing.T) {
	s := newTestHTTPServer(t)

	// Register an agent with capabilities.
	body := `{"name": "SearchBot", "capabilities": ["search"], "endpoint": {"url": "http://localhost:3000"}, "protocols": ["a2a"]}`
	req := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	// Discover.
	discoverBody := `{"capabilities": ["search"], "max_results": 10}`
	req = httptest.NewRequest("POST", "/api/v1/discover", bytes.NewBufferString(discoverBody))
	w = httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
