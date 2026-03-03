package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/peerclaw/peerclaw-go/agentcard"
	"github.com/peerclaw/peerclaw-go/envelope"
	"github.com/peerclaw/peerclaw-go/protocol"
)

// Adapter implements the ProtocolBridge interface for IBM ACP protocol.
// ACP uses REST/HTTP-based communication.
type Adapter struct {
	logger *slog.Logger
	inbox  chan *envelope.Envelope
}

// New creates a new ACP adapter.
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
	return string(protocol.ProtocolACP)
}

func (a *Adapter) Send(ctx context.Context, env *envelope.Envelope) error {
	// TODO: Implement REST API call to ACP-compatible agent.
	a.logger.Info("acp send", "dest", env.Destination, "type", env.MessageType)
	return nil
}

func (a *Adapter) Receive(ctx context.Context) (<-chan *envelope.Envelope, error) {
	return a.inbox, nil
}

func (a *Adapter) Handshake(ctx context.Context, card *agentcard.Card) error {
	// TODO: Implement ACP agent discovery handshake.
	a.logger.Info("acp handshake", "agent", card.ID, "url", card.Endpoint.URL)
	return nil
}

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
		Metadata:    env.Metadata,
		Timestamp:   env.Timestamp,
		TTL:         env.TTL,
		TraceID:     env.TraceID,
	}
	wrapper := map[string]any{
		"original_protocol": protocol.ProtocolACP,
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
		return fmt.Errorf("acp inbox full")
	}
}
