package signaling

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-core/signaling"
	"github.com/redis/go-redis/v9"
)

const redisChannel = "peerclaw:signaling"

// redisEnvelope wraps a signaling message with a node ID to avoid echo.
type redisEnvelope struct {
	NodeID  string                  `json:"node_id"`
	Message signaling.SignalMessage `json:"message"`
}

// RedisBroker distributes signaling messages across multiple nodes via Redis Pub/Sub.
type RedisBroker struct {
	client *redis.Client
	hub    *Hub
	nodeID string
	logger *slog.Logger
	cancel context.CancelFunc
}

// NewRedisBroker creates a broker backed by Redis Pub/Sub.
func NewRedisBroker(client *redis.Client, hub *Hub, logger *slog.Logger) *RedisBroker {
	if logger == nil {
		logger = slog.Default()
	}
	return &RedisBroker{
		client: client,
		hub:    hub,
		nodeID: uuid.New().String(),
		logger: logger,
	}
}

// Publish serializes the message and publishes it to the Redis channel.
func (b *RedisBroker) Publish(ctx context.Context, msg signaling.SignalMessage) error {
	env := redisEnvelope{
		NodeID:  b.nodeID,
		Message: msg,
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal redis envelope: %w", err)
	}
	return b.client.Publish(ctx, redisChannel, data).Err()
}

// Subscribe starts a Redis subscription and returns a channel of messages
// from other nodes. Messages from this node are filtered out.
func (b *RedisBroker) Subscribe(ctx context.Context) (<-chan signaling.SignalMessage, error) {
	subCtx, cancel := context.WithCancel(ctx)
	b.cancel = cancel

	pubsub := b.client.Subscribe(subCtx, redisChannel)
	// Wait for confirmation.
	if _, err := pubsub.Receive(subCtx); err != nil {
		cancel()
		return nil, fmt.Errorf("redis subscribe: %w", err)
	}

	ch := make(chan signaling.SignalMessage, 256)
	go func() {
		defer close(ch)
		defer func() { _ = pubsub.Close() }()
		msgCh := pubsub.Channel()
		for {
			select {
			case <-subCtx.Done():
				return
			case redisMsg, ok := <-msgCh:
				if !ok {
					return
				}
				var env redisEnvelope
				if err := json.Unmarshal([]byte(redisMsg.Payload), &env); err != nil {
					b.logger.Warn("invalid redis signaling message", "error", err)
					continue
				}
				// Skip messages from this node.
				if env.NodeID == b.nodeID {
					continue
				}
				// Deliver locally.
				b.hub.DeliverLocal(subCtx, env.Message)
			}
		}
	}()

	b.logger.Info("Redis signaling broker subscribed",
		"channel", redisChannel,
		"node_id", b.nodeID,
	)

	return ch, nil
}

// Close stops the subscription.
func (b *RedisBroker) Close() error {
	if b.cancel != nil {
		b.cancel()
	}
	return b.client.Close()
}
