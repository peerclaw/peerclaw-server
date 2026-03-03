package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
)

// Adapter implements the ProtocolBridge interface for Anthropic MCP protocol.
// MCP uses JSON-RPC over stdio or HTTP with SSE.
type Adapter struct {
	logger *slog.Logger
	inbox  chan *envelope.Envelope
}

// New creates a new MCP adapter.
func New(logger *slog.Logger) *Adapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &Adapter{
		logger: logger,
		inbox:  make(chan *envelope.Envelope, 100),
	}
}

func (a *Adapter) Protocol() string {
	return string(protocol.ProtocolMCP)
}

func (a *Adapter) Send(ctx context.Context, env *envelope.Envelope) error {
	// TODO: Implement JSON-RPC message delivery over stdio/HTTP.
	a.logger.Info("mcp send", "dest", env.Destination, "type", env.MessageType)
	return nil
}

func (a *Adapter) Receive(ctx context.Context) (<-chan *envelope.Envelope, error) {
	return a.inbox, nil
}

func (a *Adapter) Handshake(ctx context.Context, card *agentcard.Card) error {
	// TODO: Implement MCP initialization handshake (initialize request/response).
	a.logger.Info("mcp handshake", "agent", card.ID, "url", card.Endpoint.URL)
	return nil
}

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
		Metadata:    env.Metadata,
		Timestamp:   env.Timestamp,
		TTL:         env.TTL,
		TraceID:     env.TraceID,
	}
	wrapper := map[string]any{
		"original_protocol": protocol.ProtocolMCP,
		"payload":           json.RawMessage(env.Payload),
	}
	translated.Payload, _ = json.Marshal(wrapper)
	return translated, nil
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
