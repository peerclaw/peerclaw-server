package server

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-server/internal/bridge"
	"github.com/peerclaw/peerclaw-server/internal/signaling"
)

// BridgeForwarder reads envelopes from bridge inboxes and delivers
// them to connected agents via the signaling hub.
type BridgeForwarder struct {
	bridges *bridge.Manager
	sigHub  *signaling.Hub
	logger  *slog.Logger
	cancel  context.CancelFunc
}

// NewBridgeForwarder creates a new bridge forwarder.
func NewBridgeForwarder(bridges *bridge.Manager, sigHub *signaling.Hub, logger *slog.Logger) *BridgeForwarder {
	if logger == nil {
		logger = slog.Default()
	}
	return &BridgeForwarder{
		bridges: bridges,
		sigHub:  sigHub,
		logger:  logger,
	}
}

// Start begins forwarding messages from all bridge inboxes to the signaling hub.
func (f *BridgeForwarder) Start(ctx context.Context) {
	ctx, f.cancel = context.WithCancel(ctx)

	for _, info := range f.bridges.ListBridges() {
		b, err := f.bridges.GetBridge(info.Protocol)
		if err != nil {
			continue
		}
		ch, err := b.Receive(ctx)
		if err != nil {
			f.logger.Error("failed to get receive channel", "protocol", info.Protocol, "error", err)
			continue
		}
		go f.forwardLoop(ctx, info.Protocol, ch)
	}

	f.logger.Info("bridge forwarder started")
}

// Stop stops the forwarder.
func (f *BridgeForwarder) Stop() {
	if f.cancel != nil {
		f.cancel()
	}
}

func (f *BridgeForwarder) forwardLoop(ctx context.Context, proto string, ch <-chan *envelope.Envelope) {
	for {
		select {
		case <-ctx.Done():
			return
		case env, ok := <-ch:
			if !ok {
				return
			}
			f.deliverToAgent(ctx, env)
		}
	}
}

func (f *BridgeForwarder) deliverToAgent(ctx context.Context, env *envelope.Envelope) {
	if env.Destination == "" || env.Destination == "external" {
		return // No specific destination agent.
	}

	if f.sigHub == nil {
		f.logger.Warn("no signaling hub, cannot deliver bridge message", "dest", env.Destination)
		return
	}

	payload, err := json.Marshal(env)
	if err != nil {
		f.logger.Error("marshal envelope for delivery", "error", err)
		return
	}

	if err := f.sigHub.DeliverEnvelope(ctx, env.Destination, payload); err != nil {
		f.logger.Debug("bridge message delivery failed (agent may not be connected via signaling)",
			"dest", env.Destination, "error", err)
	}
}
