package a2a

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/bridge"
	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
	"github.com/peerclaw/peerclaw-server/internal/security"
)

const maxResponseBodySize = 10 << 20 // 10MB

// MaxTasks is the maximum number of tasks tracked by the adapter.
const MaxTasks = 10000

// DefaultTaskMaxAge is the default TTL for tracked tasks.
const DefaultTaskMaxAge = 1 * time.Hour

// Adapter implements the ProtocolBridge interface for Google A2A protocol.
// A2A uses HTTP with JSON-RPC 2.0 message format.
type Adapter struct {
	logger    *slog.Logger
	inbox     chan *envelope.Envelope
	client    *http.Client
	tasks     sync.Map // taskID → *Task
	taskCount atomic.Int64
	version   string
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

	// SSRF validation.
	if err := security.ValidateURL(endpoint); err != nil {
		return fmt.Errorf("a2a: SSRF blocked: %w", err)
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
	defer func() { _ = httpResp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxResponseBodySize))
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
	if _, loaded := a.tasks.LoadOrStore(taskID, &task); loaded {
		a.tasks.Store(taskID, &task) // update existing
	} else {
		a.taskCount.Add(1)
	}

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

// SendStream delivers an envelope and returns a channel of response chunks.
// If the remote agent returns SSE (text/event-stream), chunks are streamed in real time.
// Otherwise, falls back to buffered Send and returns one chunk.
func (a *Adapter) SendStream(ctx context.Context, env *envelope.Envelope) (<-chan bridge.StreamChunk, error) {
	endpoint := env.Metadata["a2a.endpoint"]
	if endpoint == "" {
		return nil, fmt.Errorf("a2a: missing a2a.endpoint in metadata")
	}
	if err := security.ValidateURL(endpoint); err != nil {
		return nil, fmt.Errorf("a2a: SSRF blocked: %w", err)
	}

	msg := EnvelopeToMessage(env)
	params := SendMessageParams{
		Message: msg,
		Config:  &SendMessageConfig{Blocking: false}, // request streaming
	}
	req, err := jsonrpc.NewRequest("message/send", params)
	if err != nil {
		return nil, fmt.Errorf("a2a: build request: %w", err)
	}

	url := strings.TrimRight(endpoint, "/")
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("a2a: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("a2a: new http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream, application/json")

	httpResp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("a2a: http post: %w", err)
	}

	ch := make(chan bridge.StreamChunk, 32)
	ct := httpResp.Header.Get("Content-Type")

	if strings.HasPrefix(ct, "text/event-stream") {
		// SSE streaming response — parse and relay chunks.
		go a.readSSEStream(httpResp, ch)
	} else {
		// Non-streaming response — read fully and send as single chunk.
		go a.readBufferedResponse(httpResp, ch)
	}

	return ch, nil
}

func (a *Adapter) readSSEStream(resp *http.Response, ch chan<- bridge.StreamChunk) {
	defer close(ch)
	defer func() { _ = resp.Body.Close() }()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	var eventType string
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			if eventType == "error" {
				ch <- bridge.StreamChunk{Data: data, Error: fmt.Errorf("remote error: %s", data)}
				return
			}
			if eventType == "done" || eventType == "close" {
				ch <- bridge.StreamChunk{Data: data, Done: true}
				return
			}
			ch <- bridge.StreamChunk{Data: data}
			eventType = ""
			continue
		}
		// Empty line = event boundary, reset event type.
		if line == "" {
			eventType = ""
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- bridge.StreamChunk{Error: fmt.Errorf("sse read: %w", err)}
		return
	}
	ch <- bridge.StreamChunk{Done: true}
}

func (a *Adapter) readBufferedResponse(resp *http.Response, ch chan<- bridge.StreamChunk) {
	defer close(ch)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		ch <- bridge.StreamChunk{Error: fmt.Errorf("read response: %w", err)}
		return
	}

	if resp.StatusCode != http.StatusOK {
		ch <- bridge.StreamChunk{Error: fmt.Errorf("status %d: %s", resp.StatusCode, string(body))}
		return
	}

	ch <- bridge.StreamChunk{Data: string(body), Done: true}
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
	if err := security.ValidateURL(url); err != nil {
		return fmt.Errorf("a2a: SSRF blocked: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("a2a: create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("a2a: fetch agent card: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("a2a: agent card status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
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

	var marshalErr error
	switch targetProtocol {
	case string(protocol.ProtocolMCP):
		// A2A message text → MCP tool call arguments.
		var msg Message
		if err := json.Unmarshal(env.Payload, &msg); err == nil && len(msg.Parts) > 0 {
			toolCall := map[string]any{
				"name":      env.Metadata["mcp.tool_name"],
				"arguments": map[string]string{"text": msg.Parts[0].Text},
			}
			translated.Payload, marshalErr = json.Marshal(toolCall)
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
			translated.Payload, marshalErr = json.Marshal(acpMsg)
		} else {
			translated.Payload = env.Payload
		}

	default:
		wrapper := map[string]any{
			"original_protocol": protocol.ProtocolA2A,
			"payload":           json.RawMessage(env.Payload),
		}
		translated.Payload, marshalErr = json.Marshal(wrapper)
	}

	if marshalErr != nil {
		return nil, fmt.Errorf("a2a translate: marshal payload: %w", marshalErr)
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

// StartCleanup launches a background goroutine that periodically removes expired tasks.
func (a *Adapter) StartCleanup(ctx context.Context, maxAge time.Duration) {
	if maxAge == 0 {
		maxAge = DefaultTaskMaxAge
	}
	go func() {
		ticker := time.NewTicker(maxAge / 2)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.cleanExpired(maxAge)
			}
		}
	}()
}

func (a *Adapter) cleanExpired(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	a.tasks.Range(func(key, value any) bool {
		task := value.(*Task)
		if t, err := time.Parse(time.RFC3339, task.CreatedAt); err == nil && t.Before(cutoff) {
			a.tasks.Delete(key)
			a.taskCount.Add(-1)
		}
		return true
	})
}

// TaskCount returns the current number of tracked tasks.
func (a *Adapter) TaskCount() int64 {
	return a.taskCount.Load()
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
