package federation

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

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
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		},
		stopCh: make(chan struct{}),
	}
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
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fs.logger.Warn("federation forward rejected", "peer", p.Name, "status", resp.StatusCode)
		}
	}
	return lastErr
}

// HandleIncomingSignal processes a signal message received from a federated peer.
// sourcePeer identifies which peer sent this message (empty means unverified).
func (fs *FederationService) HandleIncomingSignal(ctx context.Context, msg signaling.SignalMessage) {
	// Validate the From field is non-empty.
	if msg.From == "" {
		fs.logger.Warn("rejected federated signal with empty From field")
		return
	}

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
