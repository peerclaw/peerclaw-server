package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
)

// Adapter implements the ProtocolBridge interface for Google A2A protocol.
// A2A uses HTTP/SSE with JSON-RPC message format.
type Adapter struct {
	logger *slog.Logger
	inbox  chan *envelope.Envelope
}

// New creates a new A2A adapter.
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
	return string(protocol.ProtocolA2A)
}

func (a *Adapter) Send(ctx context.Context, env *envelope.Envelope) error {
	// TODO: Implement HTTP POST with JSON-RPC format to target agent.
	a.logger.Info("a2a send", "dest", env.Destination, "type", env.MessageType)
	return nil
}

func (a *Adapter) Receive(ctx context.Context) (<-chan *envelope.Envelope, error) {
	return a.inbox, nil
}

func (a *Adapter) Handshake(ctx context.Context, card *agentcard.Card) error {
	// TODO: Fetch /.well-known/agent.json from the agent's endpoint.
	a.logger.Info("a2a handshake", "agent", card.ID, "url", card.Endpoint.URL)
	return nil
}

func (a *Adapter) Translate(ctx context.Context, env *envelope.Envelope, targetProtocol string) (*envelope.Envelope, error) {
	if targetProtocol == string(protocol.ProtocolA2A) {
		return env, nil
	}
	// Convert payload to a generic JSON format for cross-protocol translation.
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
	// Wrap original payload with protocol metadata.
	wrapper := map[string]any{
		"original_protocol": protocol.ProtocolA2A,
		"payload":           json.RawMessage(env.Payload),
	}
	translated.Payload, _ = json.Marshal(wrapper)
	return translated, nil
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
