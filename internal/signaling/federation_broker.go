package signaling

import (
	"context"
	"log/slog"

	"github.com/peerclaw/peerclaw-core/signaling"
)

// FederationForwarder is the interface for forwarding signals to federated peers.
type FederationForwarder interface {
	ForwardSignal(ctx context.Context, msg signaling.SignalMessage) error
}

// FederationBroker wraps an existing Broker and forwards messages to
// federated peers when the target agent is not locally connected.
type FederationBroker struct {
	local      Broker
	federation FederationForwarder
	hub        *Hub
	logger     *slog.Logger
}

// NewFederationBroker creates a broker that tries local delivery first,
// then falls back to federation forwarding.
func NewFederationBroker(local Broker, federation FederationForwarder, hub *Hub, logger *slog.Logger) *FederationBroker {
	if logger == nil {
		logger = slog.Default()
	}
	return &FederationBroker{
		local:      local,
		federation: federation,
		hub:        hub,
		logger:     logger,
	}
}

// Publish sends a message locally first; if the target is not connected locally,
// it forwards to federated peers.
func (fb *FederationBroker) Publish(ctx context.Context, msg signaling.SignalMessage) error {
	// Check if target agent is connected locally.
	if fb.hub.HasAgent(msg.To) {
		return fb.local.Publish(ctx, msg)
	}

	// Forward to federation peers.
	fb.logger.Debug("forwarding signal to federation", "from", msg.From, "to", msg.To)
	if err := fb.federation.ForwardSignal(ctx, msg); err != nil {
		fb.logger.Warn("federation forward failed, trying local", "error", err)
		return fb.local.Publish(ctx, msg)
	}
	return nil
}

// Subscribe delegates to the local broker.
func (fb *FederationBroker) Subscribe(ctx context.Context) (<-chan signaling.SignalMessage, error) {
	return fb.local.Subscribe(ctx)
}

// Close closes the local broker.
func (fb *FederationBroker) Close() error {
	return fb.local.Close()
}
