package bridge

import (
	"context"

	"github.com/peerclaw/peerclaw-go/agentcard"
	"github.com/peerclaw/peerclaw-go/envelope"
)

// ProtocolBridge defines the interface for protocol-specific adapters.
// Each supported protocol (A2A, ACP, MCP) implements this interface.
type ProtocolBridge interface {
	// Protocol returns the protocol identifier this bridge handles.
	Protocol() string

	// Send delivers an envelope through this protocol.
	Send(ctx context.Context, env *envelope.Envelope) error

	// Receive returns a channel of incoming envelopes.
	Receive(ctx context.Context) (<-chan *envelope.Envelope, error)

	// Handshake initiates a connection handshake with a remote agent.
	Handshake(ctx context.Context, card *agentcard.Card) error

	// Translate converts an envelope from this protocol to the target protocol format.
	Translate(ctx context.Context, env *envelope.Envelope, targetProtocol string) (*envelope.Envelope, error)

	// Close releases resources held by this bridge.
	Close() error
}
