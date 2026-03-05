package audit

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/peerclaw/peerclaw-server/internal/config"
)

// EventType represents a type of audit event.
type EventType string

const (
	EventAgentRegistered     EventType = "agent.registered"
	EventAgentDeregistered   EventType = "agent.deregistered"
	EventMessageRouted       EventType = "message.routed"
	EventBridgeSend          EventType = "bridge.send"
	EventSignalingConnect    EventType = "signaling.connect"
	EventSignalingDisconnect EventType = "signaling.disconnect"
	EventRateLimited         EventType = "security.rate_limited"
)

// Event represents a single audit log entry.
type Event struct {
	Timestamp time.Time         `json:"timestamp"`
	Type      EventType         `json:"type"`
	AgentID   string            `json:"agent_id,omitempty"`
	SourceIP  string            `json:"source_ip,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

// Logger writes audit events to a dedicated slog.Logger instance.
type Logger struct {
	slogger *slog.Logger
}

// New creates an audit Logger from an existing slog.Logger.
// If logger is nil, a no-op logger is returned (Log calls are silently ignored).
func New(logger *slog.Logger) *Logger {
	return &Logger{slogger: logger}
}

// NewFromConfig creates an audit Logger based on the configuration.
// When cfg.Enabled is false, returns a no-op logger.
func NewFromConfig(cfg config.AuditLogConfig) (*Logger, error) {
	if !cfg.Enabled {
		return &Logger{}, nil
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}

	if strings.HasPrefix(cfg.Output, "file:") {
		path := strings.TrimPrefix(cfg.Output, "file:")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		handler = slog.NewJSONHandler(f, opts)
	} else {
		// Default to stdout.
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return &Logger{slogger: slog.New(handler)}, nil
}

// Log writes a raw audit event.
func (l *Logger) Log(ctx context.Context, event Event) {
	if l == nil || l.slogger == nil {
		return
	}
	attrs := []slog.Attr{
		slog.String("audit_type", string(event.Type)),
		slog.Time("event_time", event.Timestamp),
	}
	if event.AgentID != "" {
		attrs = append(attrs, slog.String("agent_id", event.AgentID))
	}
	if event.SourceIP != "" {
		attrs = append(attrs, slog.String("source_ip", event.SourceIP))
	}
	if event.RequestID != "" {
		attrs = append(attrs, slog.String("request_id", event.RequestID))
	}
	for k, v := range event.Details {
		attrs = append(attrs, slog.String("detail."+k, v))
	}

	args := make([]any, len(attrs))
	for i, a := range attrs {
		args[i] = a
	}
	l.slogger.LogAttrs(ctx, slog.LevelInfo, "audit", attrs...)
}

// LogRegistration records an agent registration event.
func (l *Logger) LogRegistration(ctx context.Context, agentID, agentName, sourceIP string) {
	l.Log(ctx, Event{
		Timestamp: time.Now().UTC(),
		Type:      EventAgentRegistered,
		AgentID:   agentID,
		SourceIP:  sourceIP,
		Details:   map[string]string{"agent_name": agentName},
	})
}

// LogDeregistration records an agent deregistration event.
func (l *Logger) LogDeregistration(ctx context.Context, agentID, sourceIP string) {
	l.Log(ctx, Event{
		Timestamp: time.Now().UTC(),
		Type:      EventAgentDeregistered,
		AgentID:   agentID,
		SourceIP:  sourceIP,
	})
}

// LogMessageRouted records a message routing event.
func (l *Logger) LogMessageRouted(ctx context.Context, source, dest, protocol string) {
	l.Log(ctx, Event{
		Timestamp: time.Now().UTC(),
		Type:      EventMessageRouted,
		Details: map[string]string{
			"source":      source,
			"destination": dest,
			"protocol":    protocol,
		},
	})
}

// LogBridgeSend records a bridge send event.
func (l *Logger) LogBridgeSend(ctx context.Context, source, dest, protocol string) {
	l.Log(ctx, Event{
		Timestamp: time.Now().UTC(),
		Type:      EventBridgeSend,
		Details: map[string]string{
			"source":      source,
			"destination": dest,
			"protocol":    protocol,
		},
	})
}

// LogSignalingConnect records a signaling connection event.
func (l *Logger) LogSignalingConnect(ctx context.Context, agentID, sourceIP string) {
	l.Log(ctx, Event{
		Timestamp: time.Now().UTC(),
		Type:      EventSignalingConnect,
		AgentID:   agentID,
		SourceIP:  sourceIP,
	})
}

// LogSignalingDisconnect records a signaling disconnection event.
func (l *Logger) LogSignalingDisconnect(ctx context.Context, agentID string) {
	l.Log(ctx, Event{
		Timestamp: time.Now().UTC(),
		Type:      EventSignalingDisconnect,
		AgentID:   agentID,
	})
}

// LogSecurityEvent records a security-related event.
func (l *Logger) LogSecurityEvent(ctx context.Context, eventType EventType, sourceIP string, details map[string]string) {
	l.Log(ctx, Event{
		Timestamp: time.Now().UTC(),
		Type:      eventType,
		SourceIP:  sourceIP,
		Details:   details,
	})
}
