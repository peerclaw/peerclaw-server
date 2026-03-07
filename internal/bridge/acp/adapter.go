package acp

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
	"sync/atomic"
	"time"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/security"
)

const maxResponseBodySize = 10 << 20 // 10MB

// DefaultRunMaxAge is the default TTL for tracked runs.
const DefaultRunMaxAge = 1 * time.Hour

// Adapter implements the ProtocolBridge interface for IBM ACP protocol.
// ACP uses REST/HTTP-based communication.
type Adapter struct {
	logger   *slog.Logger
	inbox    chan *envelope.Envelope
	client   *http.Client
	runs     sync.Map // runID → *Run
	runCount atomic.Int64
	sessions sync.Map // sessionID → *Session
}

// New creates a new ACP adapter.
func New(logger *slog.Logger, client *http.Client) *Adapter {
	if logger == nil {
		logger = slog.Default()
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Adapter{
		logger: logger,
		inbox:  make(chan *envelope.Envelope, 100),
		client: client,
	}
}

func (a *Adapter) Protocol() string {
	return string(protocol.ProtocolACP)
}

// Send delivers an envelope to a remote ACP agent.
func (a *Adapter) Send(ctx context.Context, env *envelope.Envelope) error {
	endpoint := env.Metadata["acp.endpoint"]
	if endpoint == "" {
		return fmt.Errorf("acp: missing acp.endpoint in metadata")
	}

	createReq := EnvelopeToCreateRun(env)

	body, err := json.Marshal(createReq)
	if err != nil {
		return fmt.Errorf("acp: marshal request: %w", err)
	}

	url := strings.TrimRight(endpoint, "/") + "/runs"
	if err := security.ValidateURL(url); err != nil {
		return fmt.Errorf("acp: SSRF blocked: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("acp: new http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := a.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("acp: http post: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxResponseBodySize))
	if err != nil {
		return fmt.Errorf("acp: read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("acp: unexpected status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var run Run
	if err := json.Unmarshal(respBody, &run); err != nil {
		return fmt.Errorf("acp: unmarshal run: %w", err)
	}

	// Store run and track session.
	if _, loaded := a.runs.LoadOrStore(run.RunID, &run); loaded {
		a.runs.Store(run.RunID, &run) // update existing
	} else {
		a.runCount.Add(1)
	}
	if run.SessionID != "" {
		a.trackSession(run.SessionID, run.RunID)
	}

	a.logger.Info("acp send complete",
		"dest", env.Destination,
		"run_id", run.RunID,
		"status", run.Status,
	)

	// Push response envelope if status is terminal or awaiting input.
	switch run.Status {
	case RunStatusCompleted, RunStatusFailed, RunStatusAwaiting:
		respEnv := RunToEnvelope(&run, env.Destination, env.Source)
		respEnv.TraceID = env.TraceID
		select {
		case a.inbox <- respEnv:
		default:
			a.logger.Warn("acp inbox full, dropping response")
		}
	}

	return nil
}

func (a *Adapter) Receive(ctx context.Context) (<-chan *envelope.Envelope, error) {
	return a.inbox, nil
}

// Handshake fetches the agent manifest from the ACP endpoint.
func (a *Adapter) Handshake(ctx context.Context, card *agentcard.Card) error {
	if card.Endpoint.URL == "" {
		return fmt.Errorf("acp: agent has no endpoint URL")
	}

	agentName := card.Name
	url := strings.TrimRight(card.Endpoint.URL, "/") + "/agents/" + agentName
	if err := security.ValidateURL(url); err != nil {
		return fmt.Errorf("acp: SSRF blocked: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("acp: create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("acp: fetch manifest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("acp: manifest status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		return fmt.Errorf("acp: read manifest: %w", err)
	}

	var manifest AgentManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return fmt.Errorf("acp: unmarshal manifest: %w", err)
	}

	a.logger.Info("acp handshake complete",
		"agent", card.ID,
		"remote_name", manifest.Name,
		"capabilities", len(manifest.Metadata.Capabilities),
	)
	return nil
}

// Translate converts an envelope from ACP format to another protocol.
func (a *Adapter) Translate(ctx context.Context, env *envelope.Envelope, targetProtocol string) (*envelope.Envelope, error) {
	if targetProtocol == string(protocol.ProtocolACP) {
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
	case string(protocol.ProtocolA2A):
		// ACP message → A2A message format.
		var msg Message
		if err := json.Unmarshal(env.Payload, &msg); err == nil && len(msg.Parts) > 0 {
			a2aMsg := map[string]any{
				"role": msg.Role,
				"parts": []map[string]string{
					{"text": msg.Parts[0].Content},
				},
			}
			translated.Payload, marshalErr = json.Marshal(a2aMsg)
		} else {
			translated.Payload = env.Payload
		}

	case string(protocol.ProtocolMCP):
		// ACP message → MCP tool call.
		var msg Message
		if err := json.Unmarshal(env.Payload, &msg); err == nil && len(msg.Parts) > 0 {
			toolCall := map[string]any{
				"name":      env.Metadata["mcp.tool_name"],
				"arguments": map[string]string{"text": msg.Parts[0].Content},
			}
			translated.Payload, marshalErr = json.Marshal(toolCall)
			translated.Metadata["mcp.method"] = "tools/call"
		} else {
			translated.Payload = env.Payload
		}

	default:
		wrapper := map[string]any{
			"original_protocol": protocol.ProtocolACP,
			"payload":           json.RawMessage(env.Payload),
		}
		translated.Payload, marshalErr = json.Marshal(wrapper)
	}

	if marshalErr != nil {
		return nil, fmt.Errorf("acp translate: marshal payload: %w", marshalErr)
	}

	return translated, nil
}

// GetRun retrieves a tracked run by ID.
func (a *Adapter) GetRun(runID string) (*Run, bool) {
	v, ok := a.runs.Load(runID)
	if !ok {
		return nil, false
	}
	return v.(*Run), true
}

func (a *Adapter) Close() error {
	close(a.inbox)
	return nil
}

// InjectMessage pushes an envelope into the receive channel for testing.
func (a *Adapter) InjectMessage(env *envelope.Envelope) error {
	select {
	case a.inbox <- env:
		return nil
	default:
		return fmt.Errorf("acp inbox full")
	}
}

func (a *Adapter) trackSession(sessionID, runID string) {
	v, _ := a.sessions.LoadOrStore(sessionID, &Session{ID: sessionID})
	sess := v.(*Session)
	sess.RunIDs = append(sess.RunIDs, runID)
}

// StartCleanup launches a background goroutine that periodically removes expired runs and sessions.
func (a *Adapter) StartCleanup(ctx context.Context, maxAge time.Duration) {
	if maxAge == 0 {
		maxAge = DefaultRunMaxAge
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
	a.runs.Range(func(key, value any) bool {
		run := value.(*Run)
		if t, err := time.Parse(time.RFC3339, run.CreatedAt); err == nil && t.Before(cutoff) {
			a.runs.Delete(key)
			a.runCount.Add(-1)
		}
		return true
	})
	// Clean orphaned sessions by checking if all their runs have been removed.
	a.sessions.Range(func(key, value any) bool {
		sess := value.(*Session)
		hasActiveRun := false
		for _, rid := range sess.RunIDs {
			if _, ok := a.runs.Load(rid); ok {
				hasActiveRun = true
				break
			}
		}
		if !hasActiveRun {
			a.sessions.Delete(key)
		}
		return true
	})
}

// RunCount returns the current number of tracked runs.
func (a *Adapter) RunCount() int64 {
	return a.runCount.Load()
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
