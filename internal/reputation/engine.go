package reputation

import (
	"context"
	"log/slog"
	"math"
	"time"
)

// Alpha is the EWMA decay factor. Higher values make the score more responsive
// to recent events.
const Alpha = 0.1

// DefaultScore is the initial reputation score for new agents.
const DefaultScore = 0.5

// eventConfig defines the weight and normalized value for each event type.
type eventConfig struct {
	Weight          float64
	NormalizedValue float64
}

var eventConfigs = map[EventType]eventConfig{
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

// Engine computes and manages agent reputation scores using EWMA.
type Engine struct {
	store  Store
	logger *slog.Logger
}

// NewEngine creates a new reputation engine.
func NewEngine(store Store, logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	return &Engine{
		store:  store,
		logger: logger,
	}
}

// RecordEvent records a reputation event and updates the agent's score.
func (e *Engine) RecordEvent(ctx context.Context, agentID string, eventType EventType, metadata string) error {
	cfg, ok := eventConfigs[eventType]
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
	newScore := Alpha*cfg.NormalizedValue + (1-Alpha)*currentScore
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
