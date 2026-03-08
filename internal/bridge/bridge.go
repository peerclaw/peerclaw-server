package bridge

import (
	"context"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/envelope"
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

// StreamChunk represents a single chunk of a streamed response.
type StreamChunk struct {
	Data  string // chunk content
	Error error  // non-nil signals end with error
	Done  bool   // true when stream is complete
}

// StreamSender is an optional interface for bridges that support streaming responses.
// Bridges implement this to enable real-time SSE passthrough for invoke calls.
type StreamSender interface {
	// SendStream delivers an envelope and returns a channel of response chunks.
	// The channel is closed when the stream is complete.
	SendStream(ctx context.Context, env *envelope.Envelope) (<-chan StreamChunk, error)
}
