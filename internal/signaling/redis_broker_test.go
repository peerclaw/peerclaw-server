//go:build integration

package signaling

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/peerclaw/peerclaw-core/signaling"
	"github.com/redis/go-redis/v9"
)

func TestRedisBroker_PubSub(t *testing.T) {
	addr := os.Getenv("PEERCLAW_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set PEERCLAW_TEST_REDIS_ADDR to run Redis integration tests")
	}

	logger := slog.Default()

	// Create two hubs simulating two nodes.
	hub1 := NewHub(logger, nil, 0)
	hub2 := NewHub(logger, nil, 0)

	client1 := redis.NewClient(&redis.Options{Addr: addr})
	client2 := redis.NewClient(&redis.Options{Addr: addr})

	broker1 := NewRedisBroker(client1, hub1, logger)
	broker2 := NewRedisBroker(client2, hub2, logger)

	ctx := context.Background()

	// Subscribe broker2.
	_, err := broker2.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Small delay for subscription to be ready.
	time.Sleep(100 * time.Millisecond)

	// Publish from broker1.
	msg := signaling.SignalMessage{
		Type: signaling.MessageTypeOffer,
		From: "agent-a",
		To:   "agent-b",
		SDP:  "test-sdp",
	}
	if err := broker1.Publish(ctx, msg); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Allow message to propagate.
	time.Sleep(200 * time.Millisecond)

	broker1.Close()
	broker2.Close()
}
