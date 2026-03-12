package bridge

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/peerclaw/peerclaw-core/envelope"
)

// Manager manages the lifecycle of protocol bridges and routes messages between them.
type Manager struct {
	mu      sync.RWMutex
	bridges map[string]ProtocolBridge
	logger  *slog.Logger
}

// NewManager creates a new bridge manager.
func NewManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		bridges: make(map[string]ProtocolBridge),
		logger:  logger,
	}
}

// RegisterBridge adds a protocol bridge to the manager.
func (m *Manager) RegisterBridge(b ProtocolBridge) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proto := b.Protocol()
	if _, exists := m.bridges[proto]; exists {
		return fmt.Errorf("bridge already registered for protocol: %s", proto)
	}
	m.bridges[proto] = b
	m.logger.Info("bridge registered", "protocol", proto)
	return nil
}

// HasBridge returns true if a bridge is registered for the given protocol.
func (m *Manager) HasBridge(proto string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.bridges[proto]
	return ok
}

// GetBridge returns the bridge for a given protocol.
func (m *Manager) GetBridge(protocol string) (ProtocolBridge, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	b, ok := m.bridges[protocol]
	if !ok {
		return nil, fmt.Errorf("no bridge registered for protocol: %s", protocol)
	}
	return b, nil
}

// Send delivers an envelope using the appropriate protocol bridge.
func (m *Manager) Send(ctx context.Context, env *envelope.Envelope) error {
	b, err := m.GetBridge(string(env.Protocol))
	if err != nil {
		return err
	}
	return b.Send(ctx, env)
}

// SendStream delivers an envelope using streaming if the bridge supports it.
// Falls back to non-streaming Send and returns a single-chunk channel.
func (m *Manager) SendStream(ctx context.Context, env *envelope.Envelope) (<-chan StreamChunk, error) {
	b, err := m.GetBridge(string(env.Protocol))
	if err != nil {
		return nil, err
	}

	// Check if bridge supports streaming.
	if ss, ok := b.(StreamSender); ok {
		return ss.SendStream(ctx, env)
	}

	// Fallback: non-streaming Send wrapped in a channel.
	ch := make(chan StreamChunk, 2)
	go func() {
		defer close(ch)
		err := b.Send(ctx, env)
		if err != nil {
			ch <- StreamChunk{Error: err}
		} else {
			ch <- StreamChunk{Data: "Message delivered to agent " + env.Destination, Done: true}
		}
	}()
	return ch, nil
}

// Translate converts an envelope between protocols.
func (m *Manager) Translate(ctx context.Context, env *envelope.Envelope, targetProtocol string) (*envelope.Envelope, error) {
	b, err := m.GetBridge(string(env.Protocol))
	if err != nil {
		return nil, fmt.Errorf("source bridge: %w", err)
	}
	return b.Translate(ctx, env, targetProtocol)
}

// ListBridges returns info about all registered bridges.
func (m *Manager) ListBridges() []BridgeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var infos []BridgeInfo
	for proto := range m.bridges {
		infos = append(infos, BridgeInfo{
			Protocol:  proto,
			Available: true,
		})
	}
	return infos
}

// BridgeInfo holds metadata about a registered bridge.
type BridgeInfo struct {
	Protocol  string `json:"protocol"`
	Available bool   `json:"available"`
}

// Close shuts down all bridges.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for proto, b := range m.bridges {
		if err := b.Close(); err != nil && firstErr == nil {
			firstErr = err
			m.logger.Error("error closing bridge", "protocol", proto, "error", err)
		}
	}
	m.bridges = make(map[string]ProtocolBridge)
	return firstErr
}
