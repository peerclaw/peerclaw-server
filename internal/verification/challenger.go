package verification

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/peerclaw/peerclaw-server/internal/security"
)

const (
	// ChallengeTTL is how long a challenge remains valid.
	ChallengeTTL = 5 * time.Minute

	// HTTPTimeout is the timeout for the verification HTTP request.
	HTTPTimeout = 5 * time.Second

	// NonceBytes is the number of random bytes for the nonce.
	NonceBytes = 32
)

// Challenger manages the endpoint verification challenge-response flow.
type Challenger struct {
	store  Store
	logger *slog.Logger
	client *http.Client
}

// NewChallenger creates a new verification challenger.
func NewChallenger(store Store, logger *slog.Logger) *Challenger {
	if logger == nil {
		logger = slog.Default()
	}
	// Use a safe HTTP client that blocks private IPs (SSRF protection).
	client := security.NewSafeHTTPClient()
	client.Timeout = HTTPTimeout
	// Disable redirects for security.
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Challenger{
		store:  store,
		logger: logger,
		client: client,
	}
}

// ChallengeResult holds the result of initiating a challenge.
type ChallengeResult struct {
	Challenge string    `json:"challenge"`
	ExpiresAt time.Time `json:"expires_at"`
}

// InitiateChallenge creates a new challenge for the given agent and sends it
// to the agent's endpoint for verification.
func (c *Challenger) InitiateChallenge(ctx context.Context, agentID, endpointURL, publicKey string) (*ChallengeResult, error) {
	// Validate the endpoint URL (SSRF protection).
	if err := security.ValidateURL(endpointURL); err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	// Generate random nonce.
	nonceBytes := make([]byte, NonceBytes)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	nonce := hex.EncodeToString(nonceBytes)

	now := time.Now().UTC()
	ch := &Challenge{
		AgentID:   agentID,
		Challenge: nonce,
		CreatedAt: now,
		ExpiresAt: now.Add(ChallengeTTL),
		Status:    StatusPending,
	}

	if err := c.store.InsertChallenge(ctx, ch); err != nil {
		return nil, fmt.Errorf("store challenge: %w", err)
	}

	// Send challenge to agent's endpoint.
	verifyURL := endpointURL + "/.well-known/peerclaw-verify?challenge=" + nonce
	resp, err := c.client.Get(verifyURL)
	if err != nil {
		_ = c.store.UpdateChallengeStatus(ctx, agentID, nonce, StatusFailed)
		return nil, fmt.Errorf("request to endpoint failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_ = c.store.UpdateChallengeStatus(ctx, agentID, nonce, StatusFailed)
		return nil, fmt.Errorf("endpoint returned status %d", resp.StatusCode)
	}

	// Read and verify the response.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		_ = c.store.UpdateChallengeStatus(ctx, agentID, nonce, StatusFailed)
		return nil, fmt.Errorf("read response: %w", err)
	}

	var verifyResp struct {
		Challenge string `json:"challenge"`
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(body, &verifyResp); err != nil {
		_ = c.store.UpdateChallengeStatus(ctx, agentID, nonce, StatusFailed)
		return nil, fmt.Errorf("invalid response format: %w", err)
	}

	// Verify nonce matches.
	if verifyResp.Challenge != nonce {
		_ = c.store.UpdateChallengeStatus(ctx, agentID, nonce, StatusFailed)
		return nil, fmt.Errorf("challenge mismatch")
	}

	// Verify Ed25519 signature.
	pubKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		_ = c.store.UpdateChallengeStatus(ctx, agentID, nonce, StatusFailed)
		return nil, fmt.Errorf("invalid public key: %w", err)
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		_ = c.store.UpdateChallengeStatus(ctx, agentID, nonce, StatusFailed)
		return nil, fmt.Errorf("invalid public key size")
	}

	sigBytes, err := hex.DecodeString(verifyResp.Signature)
	if err != nil {
		_ = c.store.UpdateChallengeStatus(ctx, agentID, nonce, StatusFailed)
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}

	if !ed25519.Verify(ed25519.PublicKey(pubKeyBytes), []byte(nonce), sigBytes) {
		_ = c.store.UpdateChallengeStatus(ctx, agentID, nonce, StatusFailed)
		return nil, fmt.Errorf("signature verification failed")
	}

	// Success: mark challenge as verified.
	_ = c.store.UpdateChallengeStatus(ctx, agentID, nonce, StatusVerified)

	c.logger.Info("endpoint verification passed", "agent_id", agentID, "endpoint", endpointURL)

	return &ChallengeResult{
		Challenge: nonce,
		ExpiresAt: ch.ExpiresAt,
	}, nil
}
