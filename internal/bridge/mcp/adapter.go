package mcp

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

	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/bridge/jsonrpc"
	"github.com/peerclaw/peerclaw-server/internal/security"
)

const maxResponseBodySize = 10 << 20 // 10MB

// DefaultSessionMaxAge is the default TTL for MCP sessions.
const DefaultSessionMaxAge = 2 * time.Hour

// MCPSession holds state for an active MCP session.
type MCPSession struct {
	SessionID   string
	ServerCaps  ServerCaps
	Endpoint    string
	Initialized bool
	CreatedAt   time.Time
}

// Adapter implements the ProtocolBridge interface for MCP (Streamable HTTP).
type Adapter struct {
	logger       *slog.Logger
	inbox        chan *envelope.Envelope
	client       *http.Client
	sessions     sync.Map // endpoint → *MCPSession
	sessionCount atomic.Int64
	version      string
}

// New creates a new MCP adapter.
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
		version: "2025-11-25",
	}
}

func (a *Adapter) Protocol() string {
	return string(protocol.ProtocolMCP)
}

// Send delivers an envelope to a remote MCP server.
func (a *Adapter) Send(ctx context.Context, env *envelope.Envelope) error {
	endpoint := env.Metadata["mcp.endpoint"]
	if endpoint == "" {
		return fmt.Errorf("mcp: missing mcp.endpoint in metadata")
	}

	// Ensure session is initialized.
	session, err := a.getOrInitSession(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("mcp: session init: %w", err)
	}

	method := env.Metadata["mcp.method"]
	if method == "" {
		method = "tools/call"
	}

	// Build params based on method.
	var params any
	switch method {
	case "tools/call":
		tc, err := EnvelopeToToolCall(env)
		if err != nil || tc == nil {
			params = json.RawMessage(env.Payload)
		} else {
			params = tc
		}
	case "tools/list", "resources/list", "prompts/list":
		params = nil
	case "resources/read":
		uri := env.Metadata["mcp.resource_uri"]
		params = ResourceReadParams{URI: uri}
	case "prompts/get":
		var pgp PromptGetParams
		if err := json.Unmarshal(env.Payload, &pgp); err != nil {
			params = json.RawMessage(env.Payload)
		} else {
			params = pgp
		}
	default:
		params = json.RawMessage(env.Payload)
	}

	req, err := jsonrpc.NewRequest(method, params)
	if err != nil {
		return fmt.Errorf("mcp: build request: %w", err)
	}

	respBody, err := a.doPost(ctx, endpoint, req, session.SessionID)
	if err != nil {
		return err
	}

	// Parse JSON-RPC response.
	var rpcResp jsonrpc.Response
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return fmt.Errorf("mcp: unmarshal response: %w", err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("mcp: remote error: %s", rpcResp.Error.Error())
	}

	// Push response into inbox.
	respEnv := envelope.New(env.Destination, env.Source, protocol.ProtocolMCP, rpcResp.Result)
	respEnv.MessageType = envelope.MessageTypeResponse
	respEnv.TraceID = env.TraceID
	respEnv.Metadata["mcp.method"] = method
	if session.SessionID != "" {
		respEnv.Metadata["mcp.session_id"] = session.SessionID
	}

	select {
	case a.inbox <- respEnv:
	default:
		a.logger.Warn("mcp inbox full, dropping response")
	}

	a.logger.Info("mcp send complete", "dest", env.Destination, "method", method)
	return nil
}

func (a *Adapter) Receive(ctx context.Context) (<-chan *envelope.Envelope, error) {
	return a.inbox, nil
}

// getOrInitSession ensures there's an initialized session for the endpoint.
func (a *Adapter) getOrInitSession(ctx context.Context, endpoint string) (*MCPSession, error) {
	if v, ok := a.sessions.Load(endpoint); ok {
		s := v.(*MCPSession)
		if s.Initialized {
			return s, nil
		}
	}

	// Send initialize request.
	initParams := InitializeParams{
		ProtocolVersion: a.version,
		Capabilities: ClientCaps{
			Roots: &RootsCap{ListChanged: false},
		},
		ClientInfo: ImplementInfo{
			Name:    "PeerClaw",
			Version: "1.0",
		},
	}

	req, err := jsonrpc.NewRequest("initialize", initParams)
	if err != nil {
		return nil, err
	}

	respBody, err := a.doPost(ctx, endpoint, req, "")
	if err != nil {
		return nil, err
	}

	var rpcResp jsonrpc.Response
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("unmarshal init response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("init error: %s", rpcResp.Error.Error())
	}

	var initResult InitializeResult
	if err := json.Unmarshal(rpcResp.Result, &initResult); err != nil {
		return nil, fmt.Errorf("unmarshal init result: %w", err)
	}

	session := &MCPSession{
		ServerCaps:  initResult.Capabilities,
		Endpoint:    endpoint,
		Initialized: true,
		CreatedAt:   time.Now(),
	}

	// Send notifications/initialized.
	notif, _ := jsonrpc.NewNotification("notifications/initialized", nil)
	a.doPost(ctx, endpoint, notif, session.SessionID) //nolint:errcheck

	if _, loaded := a.sessions.LoadOrStore(endpoint, session); loaded {
		a.sessions.Store(endpoint, session) // update existing
	} else {
		a.sessionCount.Add(1)
	}

	a.logger.Info("mcp session initialized",
		"endpoint", endpoint,
		"server", initResult.ServerInfo.Name,
	)
	return session, nil
}

// Translate converts an envelope from MCP format to another protocol.
func (a *Adapter) Translate(ctx context.Context, env *envelope.Envelope, targetProtocol string) (*envelope.Envelope, error) {
	if targetProtocol == string(protocol.ProtocolMCP) {
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
		// MCP tool result → A2A message with text parts.
		var result ToolCallResult
		if err := json.Unmarshal(env.Payload, &result); err == nil && len(result.Content) > 0 {
			parts := make([]map[string]string, len(result.Content))
			for i, c := range result.Content {
				parts[i] = map[string]string{"text": c.Text}
			}
			msg := map[string]any{
				"role":  "agent",
				"parts": parts,
			}
			translated.Payload, marshalErr = json.Marshal(msg)
		} else {
			translated.Payload = env.Payload
		}

	case string(protocol.ProtocolACP):
		// MCP → ACP: wrap as ACP message part.
		acpMsg := map[string]any{
			"role": "agent",
			"parts": []map[string]any{
				{
					"content_type": "application/json",
					"content":      string(env.Payload),
				},
			},
		}
		translated.Payload, marshalErr = json.Marshal(acpMsg)

	default:
		wrapper := map[string]any{
			"original_protocol": protocol.ProtocolMCP,
			"payload":           json.RawMessage(env.Payload),
		}
		translated.Payload, marshalErr = json.Marshal(wrapper)
	}

	if marshalErr != nil {
		return nil, fmt.Errorf("mcp translate: marshal payload: %w", marshalErr)
	}

	return translated, nil
}

// GetSession retrieves a stored session by endpoint.
func (a *Adapter) GetSession(endpoint string) (*MCPSession, bool) {
	v, ok := a.sessions.Load(endpoint)
	if !ok {
		return nil, false
	}
	return v.(*MCPSession), true
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
		return fmt.Errorf("mcp inbox full")
	}
}

func (a *Adapter) doPost(ctx context.Context, endpoint string, msg any, sessionID string) ([]byte, error) {
	url := strings.TrimRight(endpoint, "/")
	if err := security.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("mcp: SSRF blocked: %w", err)
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("mcp: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("mcp: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", sessionID)
	}

	httpResp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mcp: http post: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("mcp: read response: %w", err)
	}

	// Store session ID from response header.
	if sid := httpResp.Header.Get("Mcp-Session-Id"); sid != "" {
		if v, ok := a.sessions.Load(endpoint); ok {
			s := v.(*MCPSession)
			s.SessionID = sid
		}
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mcp: status %d: %s", httpResp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// StartCleanup launches a background goroutine that periodically removes expired sessions.
func (a *Adapter) StartCleanup(ctx context.Context, maxAge time.Duration) {
	if maxAge == 0 {
		maxAge = DefaultSessionMaxAge
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
	a.sessions.Range(func(key, value any) bool {
		session := value.(*MCPSession)
		if !session.CreatedAt.IsZero() && session.CreatedAt.Before(cutoff) {
			a.sessions.Delete(key)
			a.sessionCount.Add(-1)
		}
		return true
	})
}

// SessionCount returns the current number of tracked sessions.
func (a *Adapter) SessionCount() int64 {
	return a.sessionCount.Load()
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
