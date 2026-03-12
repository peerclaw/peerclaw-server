package registry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/protocol"
)

// Service implements agent registration, discovery, and lifecycle management.
type Service struct {
	store  Store
	logger *slog.Logger
}

// NewService creates a new registry service.
func NewService(store Store, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: store, logger: logger}
}

// RegisterRequest holds the parameters for registering an agent.
type RegisterRequest struct {
	Name         string
	Description  string
	Version      string
	PublicKey    string
	Capabilities []string
	Skills       []agentcard.Skill
	Tools        []agentcard.Tool
	Endpoint     agentcard.Endpoint
	Protocols    []protocol.Protocol
	Auth         agentcard.AuthInfo
	Metadata     map[string]string
	PeerClaw     agentcard.PeerClawExtension
	OwnerUserID  string
}

// Register creates a new agent registration.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*agentcard.Card, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if req.Endpoint.URL == "" {
		return nil, fmt.Errorf("agent endpoint URL is required")
	}
	if len(req.Protocols) == 0 {
		return nil, fmt.Errorf("at least one protocol is required")
	}
	for _, p := range req.Protocols {
		if !p.Valid() {
			return nil, fmt.Errorf("invalid protocol: %s", p)
		}
	}

	now := time.Now().UTC()
	card := &agentcard.Card{
		ID:            uuid.New().String(),
		Name:          req.Name,
		Description:   req.Description,
		Version:       req.Version,
		PublicKey:     req.PublicKey,
		Capabilities:  req.Capabilities,
		Skills:        req.Skills,
		Tools:         req.Tools,
		Endpoint:      req.Endpoint,
		Protocols:     req.Protocols,
		Auth:          req.Auth,
		Metadata:      req.Metadata,
		PeerClaw:      req.PeerClaw,
		Status:        agentcard.StatusOnline,
		RegisteredAt:  now,
		LastHeartbeat: now,
	}

	// Store owner in metadata for ownership checks.
	if req.OwnerUserID != "" {
		if card.Metadata == nil {
			card.Metadata = make(map[string]string)
		}
		card.Metadata["owner_user_id"] = req.OwnerUserID
	}

	if err := s.store.Put(ctx, card); err != nil {
		return nil, fmt.Errorf("store agent: %w", err)
	}

	s.logger.Info("agent registered", "id", card.ID, "name", card.Name)
	return card, nil
}

// Deregister removes an agent registration.
func (s *Service) Deregister(ctx context.Context, agentID string) error {
	if err := s.store.Delete(ctx, agentID); err != nil {
		return fmt.Errorf("deregister agent: %w", err)
	}
	s.logger.Info("agent deregistered", "id", agentID)
	return nil
}

// Heartbeat updates the agent's heartbeat timestamp.
func (s *Service) Heartbeat(ctx context.Context, agentID string, status agentcard.AgentStatus) (time.Time, error) {
	if status == "" {
		status = agentcard.StatusOnline
	}
	if err := s.store.UpdateHeartbeat(ctx, agentID, status); err != nil {
		return time.Time{}, fmt.Errorf("heartbeat: %w", err)
	}
	// Next heartbeat expected within 30 seconds.
	deadline := time.Now().Add(30 * time.Second)
	return deadline, nil
}

// GetAgent retrieves a single agent by ID.
func (s *Service) GetAgent(ctx context.Context, agentID string) (*agentcard.Card, error) {
	card, err := s.store.Get(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}
	return card, nil
}

// ListAgents returns agents matching the filter.
func (s *Service) ListAgents(ctx context.Context, filter ListFilter) (*ListResult, error) {
	return s.store.List(ctx, filter)
}

// Discover finds agents by capabilities and protocol.
func (s *Service) Discover(ctx context.Context, capabilities []string, proto string, maxResults int) ([]*agentcard.Card, error) {
	if len(capabilities) == 0 {
		return nil, fmt.Errorf("at least one capability is required for discovery")
	}
	return s.store.FindByCapabilities(ctx, capabilities, proto, maxResults)
}

// GetAccessFlags returns access control flags for an agent.
func (s *Service) GetAccessFlags(ctx context.Context, agentID string) (*AccessFlags, error) {
	return s.store.GetAccessFlags(ctx, agentID)
}

// GetAccessFlagsBatch returns access control flags for multiple agents.
func (s *Service) GetAccessFlagsBatch(ctx context.Context, agentIDs []string) (map[string]*AccessFlags, error) {
	return s.store.GetAccessFlagsBatch(ctx, agentIDs)
}

// SetAccessFlags updates access control flags for an agent.
func (s *Service) SetAccessFlags(ctx context.Context, agentID string, flags *AccessFlags) error {
	return s.store.SetAccessFlags(ctx, agentID, flags)
}

