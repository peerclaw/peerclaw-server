package signaling

import (
	"context"

	"github.com/peerclaw/peerclaw-core/signaling"
)

// LocalBroker delivers messages directly within a single node.
type LocalBroker struct {
	hub *Hub
}

// NewLocalBroker creates a broker that delivers messages locally via the Hub.
func NewLocalBroker(hub *Hub) *LocalBroker {
	return &LocalBroker{hub: hub}
}

// Publish delivers the message locally via the Hub.
func (b *LocalBroker) Publish(ctx context.Context, msg signaling.SignalMessage) error {
	b.hub.DeliverLocal(ctx, msg)
	return nil
}

// Subscribe returns a nil channel since local delivery is synchronous.
func (b *LocalBroker) Subscribe(ctx context.Context) (<-chan signaling.SignalMessage, error) {
	return nil, nil
}

// Close is a no-op for the local broker.
func (b *LocalBroker) Close() error {
	return nil
}
