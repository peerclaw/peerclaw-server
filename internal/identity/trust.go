package identity

import (
	"context"
	"crypto/ed25519"
	"crypto/subtle"
	"fmt"
	"strings"

	pcidentity "github.com/peerclaw/peerclaw-core/identity"
)

// Verifier validates agent identity and authorization.
type Verifier struct {
	apiKeys map[string]string // agentID -> API key
}

// NewVerifier creates a new identity verifier.
func NewVerifier() *Verifier {
	return &Verifier{
		apiKeys: make(map[string]string),
	}
}

// RegisterKey associates an API key with an agent ID.
func (v *Verifier) RegisterKey(agentID, apiKey string) {
	v.apiKeys[agentID] = apiKey
}

// RemoveKey removes the API key for an agent.
func (v *Verifier) RemoveKey(agentID string) {
	delete(v.apiKeys, agentID)
}

// VerifyAPIKey checks whether the provided key matches the registered key for the agent.
func (v *Verifier) VerifyAPIKey(agentID, providedKey string) error {
	expected, ok := v.apiKeys[agentID]
	if !ok {
		return fmt.Errorf("no API key registered for agent %s", agentID)
	}
	if subtle.ConstantTimeCompare([]byte(expected), []byte(providedKey)) != 1 {
		return fmt.Errorf("invalid API key for agent %s", agentID)
	}
	return nil
}

// VerifySignature verifies a message signature using the agent's Ed25519 public key.
func (v *Verifier) VerifySignature(pubKeyStr string, data []byte, signature string) error {
	pubKey, err := pcidentity.ParsePublicKey(pubKeyStr)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}
	return pcidentity.Verify(ed25519.PublicKey(pubKey), data, signature)
}

// ExtractBearerToken extracts a bearer token from an Authorization header value.
func ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", fmt.Errorf("invalid authorization header format")
	}
	return parts[1], nil
}

type contextKey string

const agentIDKey contextKey = "agent_id"

// WithAgentID stores the agent ID in the context.
func WithAgentID(ctx context.Context, agentID string) context.Context {
	return context.WithValue(ctx, agentIDKey, agentID)
}

// AgentIDFromContext retrieves the agent ID from the context.
func AgentIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(agentIDKey).(string)
	return id, ok
}
