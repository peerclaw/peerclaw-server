package a2a

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
)

// ServerInfo holds information about this PeerClaw server for A2A agent card.
type ServerInfo struct {
	Name        string
	Description string
	Version     string
	Endpoint    string
}

// HandleMessages handles POST /a2a — the JSON-RPC endpoint for A2A messages.
func (a *Adapter) HandleMessages(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONRPCError(w, nil, jsonrpc.CodeParseError, "failed to read body")
		return
	}

	parsed, err := jsonrpc.ParseMessage(body)
	if err != nil {
		writeJSONRPCError(w, nil, jsonrpc.CodeParseError, "invalid JSON-RPC: "+err.Error())
		return
	}

	if parsed.Kind == KindNotification {
		// Notifications don't require a response.
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if parsed.Kind != KindRequest {
		writeJSONRPCError(w, nil, jsonrpc.CodeInvalidRequest, "expected JSON-RPC request")
		return
	}

	req := parsed.Request

	switch req.Method {
	case "message/send":
		a.handleSendMessage(w, req)
	case "tasks/get":
		a.handleGetTask(w, req)
	case "tasks/cancel":
		a.handleCancelTask(w, req)
	default:
		writeJSONRPCError(w, req.ID, jsonrpc.CodeMethodNotFound, "unknown method: "+req.Method)
	}
}

func (a *Adapter) handleSendMessage(w http.ResponseWriter, req *jsonrpc.Request) {
	var params SendMessageParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeJSONRPCError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	contextID := params.Message.ContextID
	if contextID == "" {
		contextID = params.Message.MessageID
	}

	// Create a task for this message.
	task := NewTask(contextID, params.Message)

	// Store the task.
	a.tasks.Store(task.ID, task)

	// Convert to Envelope and push into inbox for bridge processing.
	payload, _ := json.Marshal(params.Message)
	env := envelope.New("external", "", protocol.ProtocolA2A, payload)
	env.Metadata["a2a.task_id"] = task.ID
	env.Metadata["a2a.context_id"] = contextID
	env.Metadata["a2a.state"] = string(task.Status.State)

	select {
	case a.inbox <- env:
	default:
		a.logger.Warn("a2a inbox full, dropping inbound message")
		writeJSONRPCError(w, req.ID, jsonrpc.CodeInternalError, "server busy")
		return
	}

	// Return the task as the response.
	writeJSONRPCResult(w, req.ID, task)
}

func (a *Adapter) handleGetTask(w http.ResponseWriter, req *jsonrpc.Request) {
	var params GetTaskParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeJSONRPCError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	task, ok := a.GetTask(params.ID)
	if !ok {
		writeJSONRPCError(w, req.ID, jsonrpc.CodeInternalError, "task not found: "+params.ID)
		return
	}

	writeJSONRPCResult(w, req.ID, task)
}

func (a *Adapter) handleCancelTask(w http.ResponseWriter, req *jsonrpc.Request) {
	var params CancelTaskParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeJSONRPCError(w, req.ID, jsonrpc.CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	v, ok := a.tasks.Load(params.ID)
	if !ok {
		writeJSONRPCError(w, req.ID, jsonrpc.CodeInternalError, "task not found: "+params.ID)
		return
	}

	task := v.(*Task)
	task.Status = TaskStatus{
		State:     TaskStateCanceled,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	a.tasks.Store(params.ID, task)

	writeJSONRPCResult(w, req.ID, task)
}

// HandleAgentCard handles GET /.well-known/agent.json.
func (a *Adapter) HandleAgentCard(w http.ResponseWriter, r *http.Request) {
	card := AgentCard{
		Name:    "PeerClaw Gateway",
		Version: a.version,
		Capabilities: A2ACaps{
			Streaming: false,
			MultiTurn: true,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

// HandleGetTask handles GET /a2a/tasks/{id}.
func (a *Adapter) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, `{"error":"missing task id"}`, http.StatusBadRequest)
		return
	}

	task, ok := a.GetTask(taskID)
	if !ok {
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// KindNotification is used for checking parsed message kind.
const KindNotification = jsonrpc.KindNotification
const KindRequest = jsonrpc.KindRequest

func writeJSONRPCResult(w http.ResponseWriter, id any, result any) {
	resp, err := jsonrpc.NewResponse(id, result)
	if err != nil {
		writeJSONRPCError(w, id, jsonrpc.CodeInternalError, "marshal error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeJSONRPCError(w http.ResponseWriter, id any, code int, message string) {
	resp := jsonrpc.NewErrorResponse(id, code, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors still use 200
	json.NewEncoder(w).Encode(resp)
}
