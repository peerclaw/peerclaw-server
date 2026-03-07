package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/peerclaw/peerclaw-server/internal/config"
)

func TestNew_NilLogger(t *testing.T) {
	l := New(nil)
	// Should not panic.
	l.Log(context.Background(), Event{Type: EventAgentRegistered})
}

func TestNew_WithLogger(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	l := New(slog.New(handler))

	l.LogRegistration(context.Background(), "agent-1", "TestAgent", "1.2.3.4")

	if buf.Len() == 0 {
		t.Error("expected audit log output")
	}

	// Parse the JSON to verify fields.
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if entry["audit_type"] != string(EventAgentRegistered) {
		t.Errorf("audit_type = %v, want %v", entry["audit_type"], EventAgentRegistered)
	}
	if entry["agent_id"] != "agent-1" {
		t.Errorf("agent_id = %v, want agent-1", entry["agent_id"])
	}
}

func TestNewFromConfig_Disabled(t *testing.T) {
	l, err := NewFromConfig(config.AuditLogConfig{Enabled: false})
	if err != nil {
		t.Fatalf("NewFromConfig() error = %v", err)
	}
	// Should not panic.
	l.LogRegistration(context.Background(), "agent-1", "TestAgent", "1.2.3.4")
}

func TestNewFromConfig_Stdout(t *testing.T) {
	l, err := NewFromConfig(config.AuditLogConfig{Enabled: true, Output: "stdout"})
	if err != nil {
		t.Fatalf("NewFromConfig() error = %v", err)
	}
	if l.slogger == nil {
		t.Error("expected non-nil slogger for stdout output")
	}
}

func TestNewFromConfig_File(t *testing.T) {
	tmpFile := t.TempDir() + "/audit.log"
	l, err := NewFromConfig(config.AuditLogConfig{Enabled: true, Output: "file:" + tmpFile})
	if err != nil {
		t.Fatalf("NewFromConfig() error = %v", err)
	}
	if l.slogger == nil {
		t.Error("expected non-nil slogger for file output")
	}

	l.LogRegistration(context.Background(), "agent-1", "TestAgent", "1.2.3.4")
}

func TestLogDeregistration(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	l := New(slog.New(handler))

	l.LogDeregistration(context.Background(), "agent-1", "1.2.3.4")

	var entry map[string]any
	_ = json.Unmarshal(buf.Bytes(), &entry)
	if entry["audit_type"] != string(EventAgentDeregistered) {
		t.Errorf("audit_type = %v, want %v", entry["audit_type"], EventAgentDeregistered)
	}
}

func TestLogMessageRouted(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	l := New(slog.New(handler))

	l.LogMessageRouted(context.Background(), "agent-a", "agent-b", "a2a")

	var entry map[string]any
	_ = json.Unmarshal(buf.Bytes(), &entry)
	if entry["audit_type"] != string(EventMessageRouted) {
		t.Errorf("audit_type = %v, want %v", entry["audit_type"], EventMessageRouted)
	}
}

func TestLogSecurityEvent(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	l := New(slog.New(handler))

	l.LogSecurityEvent(context.Background(), EventRateLimited, "1.2.3.4", map[string]string{"reason": "exceeded"})

	var entry map[string]any
	_ = json.Unmarshal(buf.Bytes(), &entry)
	if entry["audit_type"] != string(EventRateLimited) {
		t.Errorf("audit_type = %v, want %v", entry["audit_type"], EventRateLimited)
	}
}

func TestNilLogger_AllMethods(t *testing.T) {
	var l *Logger
	ctx := context.Background()
	// All should be no-ops without panic.
	l.Log(ctx, Event{})
	l.LogRegistration(ctx, "", "", "")
	l.LogDeregistration(ctx, "", "")
	l.LogMessageRouted(ctx, "", "", "")
	l.LogBridgeSend(ctx, "", "", "")
	l.LogSignalingConnect(ctx, "", "")
	l.LogSignalingDisconnect(ctx, "")
	l.LogSecurityEvent(ctx, "", "", nil)
}
