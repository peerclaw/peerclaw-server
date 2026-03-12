package reputation

import (
	"context"
	"log/slog"
	"math"
	"time"
)

// DefaultAlpha is the default EWMA decay factor. Higher values make the score
// more responsive to recent events.
const DefaultAlpha = 0.1

// Alpha is kept for backward compatibility with tests.
const Alpha = DefaultAlpha

// DefaultScore is the initial reputation score for new agents.
const DefaultScore = 0.5

// EventConfig defines the weight and normalized value for each event type.
type EventConfig struct {
	Weight          float64
	NormalizedValue float64
}

// DefaultEventConfigs returns the default event weight/value configuration.
func DefaultEventConfigs() map[EventType]EventConfig {
	return map[EventType]EventConfig{
		EventRegistration:     {Weight: 0.2, NormalizedValue: 0.6},
		EventHeartbeatSuccess: {Weight: 0.1, NormalizedValue: 0.55},
		EventHeartbeatMiss:    {Weight: -0.3, NormalizedValue: 0.35},
		EventVerificationPass: {Weight: 1.0, NormalizedValue: 1.0},
		EventVerificationFail: {Weight: -0.5, NormalizedValue: 0.25},
		EventBridgeSuccess:    {Weight: 0.3, NormalizedValue: 0.65},
		EventBridgeError:      {Weight: -0.2, NormalizedValue: 0.4},
		EventBridgeTimeout:    {Weight: -0.3, NormalizedValue: 0.35},
		EventReviewPositive:   {Weight: 0.4, NormalizedValue: 0.7},
		EventReviewNegative:   {Weight: -0.4, NormalizedValue: 0.3},
	}
}

// EngineConfig holds tunable parameters for the reputation engine.
type EngineConfig struct {
	// Alpha is the EWMA decay factor (0..1). Default: 0.1.
	Alpha float64
	// EventWeights overrides the default event configurations.
	// If nil, DefaultEventConfigs() is used.
	EventWeights map[EventType]EventConfig
}

// Engine computes and manages agent reputation scores using EWMA.
type Engine struct {
	store        Store
	logger       *slog.Logger
	alpha        float64
	eventConfigs map[EventType]EventConfig
}

// NewEngine creates a new reputation engine with default configuration.
func NewEngine(store Store, logger *slog.Logger) *Engine {
	return NewEngineWithConfig(store, logger, EngineConfig{})
}

// NewEngineWithConfig creates a new reputation engine with the given configuration.
func NewEngineWithConfig(store Store, logger *slog.Logger, cfg EngineConfig) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	alpha := cfg.Alpha
	if alpha <= 0 || alpha > 1 {
		alpha = DefaultAlpha
	}
	ec := cfg.EventWeights
	if ec == nil {
		ec = DefaultEventConfigs()
	}
	return &Engine{
		store:        store,
		logger:       logger,
		alpha:        alpha,
		eventConfigs: ec,
	}
}

// RecordEvent records a reputation event and updates the agent's score.
func (e *Engine) RecordEvent(ctx context.Context, agentID string, eventType EventType, metadata string) error {
	cfg, ok := e.eventConfigs[eventType]
	if !ok {
		e.logger.Warn("unknown reputation event type", "event_type", eventType)
		return nil
	}

	// Get current score.
	currentScore, eventCount, err := e.store.GetScore(ctx, agentID)
	if err != nil {
		// Agent has no events yet, use default score.
		currentScore = DefaultScore
		eventCount = 0
	}

	// EWMA update: score = alpha * normalizedValue + (1 - alpha) * oldScore
	newScore := e.alpha*cfg.NormalizedValue + (1-e.alpha)*currentScore
	newScore = math.Max(0, math.Min(1, newScore)) // clamp to [0, 1]
	eventCount++

	event := &Event{
		AgentID:    agentID,
		EventType:  eventType,
		Weight:     cfg.Weight,
		ScoreAfter: newScore,
		Metadata:   metadata,
		CreatedAt:  time.Now().UTC(),
	}

	if err := e.store.InsertEvent(ctx, event); err != nil {
		e.logger.Error("failed to insert reputation event", "error", err, "agent_id", agentID)
		return err
	}

	if err := e.store.UpdateAgentReputation(ctx, agentID, newScore, eventCount); err != nil {
		e.logger.Error("failed to update agent reputation", "error", err, "agent_id", agentID)
		return err
	}

	e.logger.Debug("reputation event recorded",
		"agent_id", agentID,
		"event_type", eventType,
		"old_score", currentScore,
		"new_score", newScore,
		"event_count", eventCount,
	)

	return nil
}

// GetScore returns the current reputation score for an agent.
func (e *Engine) GetScore(ctx context.Context, agentID string) (float64, error) {
	score, _, err := e.store.GetScore(ctx, agentID)
	if err != nil {
		return DefaultScore, nil
	}
	return score, nil
}

// GetScoresBatch returns reputation scores for multiple agents in a single pass.
func (e *Engine) GetScoresBatch(ctx context.Context, agentIDs []string) map[string]float64 {
	result := make(map[string]float64, len(agentIDs))
	for _, id := range agentIDs {
		score, _, err := e.store.GetScore(ctx, id)
		if err != nil {
			score = DefaultScore
		}
		result[id] = score
	}
	return result
}

// GetHistory returns the reputation event history for an agent.
func (e *Engine) GetHistory(ctx context.Context, agentID string, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 50
	}
	return e.store.ListEvents(ctx, agentID, limit)
}

// SetVerified marks an agent as verified and records a verification_pass event.
func (e *Engine) SetVerified(ctx context.Context, agentID string) error {
	if err := e.store.SetAgentVerified(ctx, agentID); err != nil {
		return err
	}
	return e.RecordEvent(ctx, agentID, EventVerificationPass, "")
}

// UnsetVerified removes the verified status from an agent.
func (e *Engine) UnsetVerified(ctx context.Context, agentID string) error {
	return e.store.UnsetAgentVerified(ctx, agentID)
}
