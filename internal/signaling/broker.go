package signaling

import (
	"context"

	"github.com/peerclaw/peerclaw-core/signaling"
)

// Broker abstracts the message distribution mechanism for signaling.
// LocalBroker delivers messages within a single node; RedisBroker
// distributes messages across multiple nodes via Redis Pub/Sub.
type Broker interface {
	// Publish sends a signal message for delivery.
	Publish(ctx context.Context, msg signaling.SignalMessage) error

	// Subscribe returns a channel that receives messages from other nodes.
	Subscribe(ctx context.Context) (<-chan signaling.SignalMessage, error)

	// Close releases resources.
	Close() error
}
