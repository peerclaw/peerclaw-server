package acp

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
)

// HandleListAgents handles GET /acp/agents.
func (a *Adapter) HandleListAgents(w http.ResponseWriter, r *http.Request) {
	// Return an empty list; agents register dynamically.
	agents := []AgentManifest{}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"agents": agents})
}

// HandleGetAgent handles GET /acp/agents/{name}.
func (a *Adapter) HandleGetAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, `{"error":"missing agent name"}`, http.StatusBadRequest)
		return
	}

	// Return a placeholder manifest for the PeerClaw gateway.
	manifest := AgentManifest{
		Name:               name,
		Description:        "PeerClaw Gateway Agent",
		InputContentTypes:  []string{"text/plain", "application/json"},
		OutputContentTypes: []string{"text/plain", "application/json"},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(manifest)
}

// HandleCreateRun handles POST /acp/runs.
func (a *Adapter) HandleCreateRun(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"failed to read body"}`, http.StatusBadRequest)
		return
	}

	var req CreateRunRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, `{"error":"invalid request: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	if req.AgentName == "" {
		http.Error(w, `{"error":"agent_name is required"}`, http.StatusBadRequest)
		return
	}

	// Create the run.
	run := NewRun(req)
	a.runs.Store(run.RunID, run)
	if run.SessionID != "" {
		a.trackSession(run.SessionID, run.RunID)
	}

	// Convert to Envelope and push into inbox for bridge processing.
	payload, _ := json.Marshal(req)
	env := envelope.New("external", req.AgentName, protocol.ProtocolACP, payload)
	env.Metadata["acp.run_id"] = run.RunID
	env.Metadata["acp.session_id"] = run.SessionID
	env.Metadata["acp.agent_name"] = req.AgentName

	select {
	case a.inbox <- env:
	default:
		a.logger.Warn("acp inbox full, dropping inbound run")
		http.Error(w, `{"error":"server busy"}`, http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(run)
}

// HandleGetRun handles GET /acp/runs/{run_id}.
func (a *Adapter) HandleGetRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("run_id")
	if runID == "" {
		http.Error(w, `{"error":"missing run_id"}`, http.StatusBadRequest)
		return
	}

	run, ok := a.GetRun(runID)
	if !ok {
		http.Error(w, `{"error":"run not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(run)
}

// HandleCancelRun handles POST /acp/runs/{run_id}/cancel.
func (a *Adapter) HandleCancelRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("run_id")
	if runID == "" {
		http.Error(w, `{"error":"missing run_id"}`, http.StatusBadRequest)
		return
	}

	v, ok := a.runs.Load(runID)
	if !ok {
		http.Error(w, `{"error":"run not found"}`, http.StatusNotFound)
		return
	}

	run := v.(*Run)
	run.Status = RunStatusCancelled
	run.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	a.runs.Store(runID, run)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(run)
}

// HandlePing handles GET /acp/ping.
func (a *Adapter) HandlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
