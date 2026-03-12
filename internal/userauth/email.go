package userauth

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"github.com/peerclaw/peerclaw-server/internal/config"
)

// EmailSender sends verification emails.
type EmailSender interface {
	SendVerificationCode(to, code, purpose string) error
}

// SMTPSender sends emails via SMTP.
type SMTPSender struct {
	host     string
	port     int
	username string
	password string
	from     string
	useTLS   bool
}

// LogSender logs OTP codes instead of sending emails (development mode).
type LogSender struct {
	logger *slog.Logger
}

// NewEmailSender returns an SMTPSender if SMTP is configured, otherwise a LogSender.
func NewEmailSender(cfg config.SMTPConfig, logger *slog.Logger) EmailSender {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Host != "" {
		port := cfg.Port
		if port == 0 {
			port = 587
		}
		from := cfg.From
		if from == "" {
			from = cfg.Username
		}
		return &SMTPSender{
			host:     cfg.Host,
			port:     port,
			username: cfg.Username,
			password: cfg.Password,
			from:     from,
			useTLS:   cfg.TLS || cfg.Port == 0, // default TLS on
		}
	}
	logger.Warn("SMTP not configured — email verification codes will be logged (dev mode)")
	return &LogSender{logger: logger}
}

func (s *SMTPSender) SendVerificationCode(to, code, purpose string) error {
	subject := "Your PeerClaw verification code"
	if purpose == "password_reset" {
		subject = "Your PeerClaw password reset code"
	}

	body := fmt.Sprintf("Your verification code is: %s\n\nThis code expires in 10 minutes.\nIf you did not request this, please ignore this email.", code)

	msg := strings.Join([]string{
		"From: " + s.from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	if s.useTLS {
		tlsConfig := &tls.Config{ServerName: s.host}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("smtp tls dial: %w", err)
		}
		client, err := smtp.NewClient(conn, s.host)
		if err != nil {
			return fmt.Errorf("smtp new client: %w", err)
		}
		defer func() { _ = client.Quit() }()

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
		if err := client.Mail(s.from); err != nil {
			return fmt.Errorf("smtp mail: %w", err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("smtp data: %w", err)
		}
		if _, err := w.Write([]byte(msg)); err != nil {
			return fmt.Errorf("smtp write: %w", err)
		}
		return w.Close()
	}

	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}

func (s *LogSender) SendVerificationCode(to, code, purpose string) error {
	s.logger.Warn("EMAIL VERIFICATION CODE (dev mode)",
		"to", to,
		"code", code,
		"purpose", purpose,
	)
	return nil
}
