package userauth

import (
	"log/slog"
	"testing"

	"github.com/peerclaw/peerclaw-server/internal/config"
)

func TestNewEmailSender_NoConfig(t *testing.T) {
	sender := NewEmailSender(config.SMTPConfig{}, slog.Default())
	if _, ok := sender.(*LogSender); !ok {
		t.Fatalf("expected LogSender, got %T", sender)
	}
}

func TestNewEmailSender_WithConfig(t *testing.T) {
	sender := NewEmailSender(config.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "noreply@example.com",
	}, slog.Default())
	if _, ok := sender.(*SMTPSender); !ok {
		t.Fatalf("expected SMTPSender, got %T", sender)
	}
}

func TestLogSender_SendVerificationCode(t *testing.T) {
	sender := &LogSender{logger: slog.Default()}
	err := sender.SendVerificationCode("test@example.com", "123456", "register")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
