package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
)

// Adapter implements the ProtocolBridge interface for Google A2A protocol.
// A2A uses HTTP with JSON-RPC 2.0 message format.
type Adapter struct {
	logger  *slog.Logger
	inbox   chan *envelope.Envelope
	client  *http.Client
	tasks   sync.Map // taskID → *Task
	version string
}

// New creates a new A2A adapter.
func New(logger *slog.Logger, client *http.Client) *Adapter {
	if logger == nil {
		logger = slog.Default()
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Adapter{
		logger:  logger,
		inbox:   make(chan *envelope.Envelope, 100),
		client:  client,
		version: "0.3",
	}
}

func (a *Adapter) Protocol() string {
	return string(protocol.ProtocolA2A)
}

// Send delivers an envelope to a remote A2A agent.
func (a *Adapter) Send(ctx context.Context, env *envelope.Envelope) error {
	endpoint := env.Metadata["a2a.endpoint"]
	if endpoint == "" {
		return fmt.Errorf("a2a: missing a2a.endpoint in metadata")
	}

	// Build A2A message from envelope.
	msg := EnvelopeToMessage(env)

	// Determine task ID: reuse existing or create new.
	taskID := env.Metadata["a2a.task_id"]
	contextID := env.Metadata["a2a.context_id"]
	if contextID == "" {
		contextID = env.TraceID
	}

	params := SendMessageParams{
		Message: msg,
		Config:  &SendMessageConfig{Blocking: true},
	}

	req, err := jsonrpc.NewRequest("message/send", params)
	if err != nil {
		return fmt.Errorf("a2a: build request: %w", err)
	}

	// POST to the endpoint.
	url := strings.TrimRight(endpoint, "/")
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("a2a: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("a2a: new http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := a.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("a2a: http post: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("a2a: read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("a2a: unexpected status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Parse JSON-RPC response.
	var rpcResp jsonrpc.Response
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return fmt.Errorf("a2a: unmarshal response: %w", err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("a2a: remote error: %s", rpcResp.Error.Error())
	}

	// Parse task from result.
	var task Task
	if err := json.Unmarshal(rpcResp.Result, &task); err != nil {
		return fmt.Errorf("a2a: unmarshal task: %w", err)
	}

	if taskID == "" {
		taskID = task.ID
	}
	a.tasks.Store(taskID, &task)

	a.logger.Info("a2a send complete",
		"dest", env.Destination,
		"task_id", taskID,
		"state", task.Status.State,
	)

	// If task has artifacts or is completed, push response envelope into inbox.
	if task.Status.State == TaskStateCompleted || len(task.Artifacts) > 0 {
		respEnv := TaskToEnvelope(&task, env.Destination, env.Source)
		respEnv.TraceID = env.TraceID
		if contextID != "" {
			respEnv.Metadata["a2a.context_id"] = contextID
		}
		select {
		case a.inbox <- respEnv:
		default:
			a.logger.Warn("a2a inbox full, dropping response")
		}
	}

	return nil
}

func (a *Adapter) Receive(ctx context.Context) (<-chan *envelope.Envelope, error) {
	return a.inbox, nil
}

// Handshake fetches /.well-known/agent.json from the agent's endpoint.
func (a *Adapter) Handshake(ctx context.Context, card *agentcard.Card) error {
	if card.Endpoint.URL == "" {
		return fmt.Errorf("a2a: agent has no endpoint URL")
	}

	url := strings.TrimRight(card.Endpoint.URL, "/") + "/.well-known/agent.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("a2a: create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("a2a: fetch agent card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("a2a: agent card status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("a2a: read agent card: %w", err)
	}

	var agentCard AgentCard
	if err := json.Unmarshal(body, &agentCard); err != nil {
		return fmt.Errorf("a2a: unmarshal agent card: %w", err)
	}

	a.logger.Info("a2a handshake complete",
		"agent", card.ID,
		"remote_name", agentCard.Name,
		"skills", len(agentCard.Skills),
	)
	return nil
}

// Translate converts an envelope from A2A format to another protocol.
func (a *Adapter) Translate(ctx context.Context, env *envelope.Envelope, targetProtocol string) (*envelope.Envelope, error) {
	if targetProtocol == string(protocol.ProtocolA2A) {
		return env, nil
	}

	translated := &envelope.Envelope{
		ID:          env.ID,
		Source:      env.Source,
		Destination: env.Destination,
		Protocol:    protocol.Protocol(targetProtocol),
		MessageType: env.MessageType,
		ContentType: "application/json",
		Metadata:    copyMetadata(env.Metadata),
		Timestamp:   env.Timestamp,
		TTL:         env.TTL,
		TraceID:     env.TraceID,
	}

	switch targetProtocol {
	case string(protocol.ProtocolMCP):
		// A2A message text → MCP tool call arguments.
		var msg Message
		if err := json.Unmarshal(env.Payload, &msg); err == nil && len(msg.Parts) > 0 {
			toolCall := map[string]any{
				"name":      env.Metadata["mcp.tool_name"],
				"arguments": map[string]string{"text": msg.Parts[0].Text},
			}
			translated.Payload, _ = json.Marshal(toolCall)
			translated.Metadata["mcp.method"] = "tools/call"
		} else {
			translated.Payload = env.Payload
		}

	case string(protocol.ProtocolACP):
		// A2A message → ACP message format.
		var msg Message
		if err := json.Unmarshal(env.Payload, &msg); err == nil {
			acpMsg := map[string]any{
				"role":  msg.Role,
				"parts": msg.Parts,
			}
			translated.Payload, _ = json.Marshal(acpMsg)
		} else {
			translated.Payload = env.Payload
		}

	default:
		wrapper := map[string]any{
			"original_protocol": protocol.ProtocolA2A,
			"payload":           json.RawMessage(env.Payload),
		}
		translated.Payload, _ = json.Marshal(wrapper)
	}

	return translated, nil
}

// GetTask retrieves a tracked task by ID.
func (a *Adapter) GetTask(taskID string) (*Task, bool) {
	v, ok := a.tasks.Load(taskID)
	if !ok {
		return nil, false
	}
	return v.(*Task), true
}

func (a *Adapter) Close() error {
	close(a.inbox)
	return nil
}

// InjectMessage is a helper for testing: pushes an envelope into the receive channel.
func (a *Adapter) InjectMessage(env *envelope.Envelope) error {
	select {
	case a.inbox <- env:
		return nil
	default:
		return fmt.Errorf("a2a inbox full")
	}
}

func copyMetadata(m map[string]string) map[string]string {
	if m == nil {
		return make(map[string]string)
	}
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
