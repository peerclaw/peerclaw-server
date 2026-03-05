package federation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/signaling"
)

// FederationService manages connections to federated peer servers.
type FederationService struct {
	mu        sync.RWMutex
	nodeName  string
	peers     map[string]*FederationPeer
	authToken string
	logger    *slog.Logger
	client    *http.Client
	onSignal  func(ctx context.Context, msg signaling.SignalMessage)
	stopCh    chan struct{}
}

// FederationPeer represents a connected federated server peer.
type FederationPeer struct {
	Name      string
	Address   string
	Token     string
	Connected bool
	LastSync  time.Time
}

// New creates a new FederationService.
func New(nodeName, authToken string, logger *slog.Logger) *FederationService {
	if logger == nil {
		logger = slog.Default()
	}
	return &FederationService{
		nodeName:  nodeName,
		peers:     make(map[string]*FederationPeer),
		authToken: authToken,
		logger:    logger,
		client:    &http.Client{Timeout: 10 * time.Second},
		stopCh:    make(chan struct{}),
	}
}

// NodeName returns the name of this federation node.
func (fs *FederationService) NodeName() string {
	return fs.nodeName
}

// OnSignal registers a callback for incoming federated signal messages.
func (fs *FederationService) OnSignal(fn func(ctx context.Context, msg signaling.SignalMessage)) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.onSignal = fn
}

// AddPeer adds a federated peer server.
func (fs *FederationService) AddPeer(name, address, token string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.peers[name] = &FederationPeer{
		Name:    name,
		Address: address,
		Token:   token,
	}
	fs.logger.Info("federation peer added", "name", name, "address", address)
}

// ListPeers returns all configured federation peers.
func (fs *FederationService) ListPeers() []*FederationPeer {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	result := make([]*FederationPeer, 0, len(fs.peers))
	for _, p := range fs.peers {
		result = append(result, p)
	}
	return result
}

// ForwardSignal forwards a signaling message to all federated peers.
func (fs *FederationService) ForwardSignal(ctx context.Context, msg signaling.SignalMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal signal message: %w", err)
	}

	fs.mu.RLock()
	peers := make([]*FederationPeer, 0, len(fs.peers))
	for _, p := range fs.peers {
		peers = append(peers, p)
	}
	fs.mu.RUnlock()

	var lastErr error
	for _, p := range peers {
		req, err := http.NewRequestWithContext(ctx, "POST",
			p.Address+"/api/v1/federation/signal",
			bytes.NewReader(data))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if p.Token != "" {
			req.Header.Set("Authorization", "Bearer "+p.Token)
		}

		resp, err := fs.client.Do(req)
		if err != nil {
			fs.logger.Warn("federation forward failed", "peer", p.Name, "error", err)
			lastErr = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fs.logger.Warn("federation forward rejected", "peer", p.Name, "status", resp.StatusCode)
		}
	}
	return lastErr
}

// QueryAgents queries all federated peers for agents with the given capabilities.
func (fs *FederationService) QueryAgents(ctx context.Context, capabilities []string) ([]*agentcard.Card, error) {
	fs.mu.RLock()
	peers := make([]*FederationPeer, 0, len(fs.peers))
	for _, p := range fs.peers {
		peers = append(peers, p)
	}
	fs.mu.RUnlock()

	var allCards []*agentcard.Card
	for _, p := range peers {
		body, _ := json.Marshal(map[string]any{
			"capabilities": capabilities,
		})
		req, err := http.NewRequestWithContext(ctx, "POST",
			p.Address+"/api/v1/discover",
			bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if p.Token != "" {
			req.Header.Set("Authorization", "Bearer "+p.Token)
		}

		resp, err := fs.client.Do(req)
		if err != nil {
			fs.logger.Warn("federation query failed", "peer", p.Name, "error", err)
			continue
		}

		var result struct {
			Agents []*agentcard.Card `json:"agents"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
		allCards = append(allCards, result.Agents...)
	}
	return allCards, nil
}

// HandleIncomingSignal processes a signal message received from a federated peer.
func (fs *FederationService) HandleIncomingSignal(ctx context.Context, msg signaling.SignalMessage) {
	fs.mu.RLock()
	handler := fs.onSignal
	fs.mu.RUnlock()
	if handler != nil {
		handler(ctx, msg)
	}
}

// AuthToken returns the authentication token for this federation node.
func (fs *FederationService) AuthToken() string {
	return fs.authToken
}

// Close stops the federation service.
func (fs *FederationService) Close() error {
	close(fs.stopCh)
	return nil
}
