package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the PeerClaw gateway.
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Database      DatabaseConfig      `yaml:"database"`
	Redis         RedisConfig         `yaml:"redis"`
	Logging       LoggingConfig       `yaml:"logging"`
	Bridge        BridgeConfig        `yaml:"bridge"`
	Signaling     SignalingConfig     `yaml:"signaling"`
	Observability ObservabilityConfig `yaml:"observability"`
	RateLimit     RateLimitConfig     `yaml:"rate_limit"`
	AuditLog      AuditLogConfig      `yaml:"audit_log"`
}

// ServerConfig holds HTTP and gRPC server settings.
type ServerConfig struct {
	HTTPAddr string `yaml:"http_addr"`
	GRPCAddr string `yaml:"grpc_addr"`
}

// DatabaseConfig holds database settings.
type DatabaseConfig struct {
	Driver string `yaml:"driver"` // "sqlite" (default) or "postgres"
	DSN    string `yaml:"dsn"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"` // json or text
}

// BridgeConfig holds protocol bridge settings.
type BridgeConfig struct {
	A2A BridgeEndpoint `yaml:"a2a"`
	ACP BridgeEndpoint `yaml:"acp"`
	MCP BridgeEndpoint `yaml:"mcp"`
}

// BridgeEndpoint holds settings for a single protocol bridge.
type BridgeEndpoint struct {
	Enabled bool `yaml:"enabled"`
}

// SignalingConfig holds WebSocket signaling and TURN server settings.
type SignalingConfig struct {
	Enabled  bool       `yaml:"enabled"`
	TURN     TURNConfig `yaml:"turn"`
}

// TURNConfig holds TURN server settings for WebRTC NAT traversal.
type TURNConfig struct {
	URLs       []string `yaml:"urls"`
	Username   string   `yaml:"username"`
	Credential string   `yaml:"credential"`
}

// ObservabilityConfig holds OpenTelemetry settings.
type ObservabilityConfig struct {
	Enabled        bool    `yaml:"enabled"`          // default false
	OTLPEndpoint   string  `yaml:"otlp_endpoint"`    // e.g. "localhost:4317"
	ServiceName    string  `yaml:"service_name"`     // default "peerclaw-gateway"
	TracesSampling float64 `yaml:"traces_sampling"`  // default 0.1
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	Enabled         bool    `yaml:"enabled"`           // default true
	RequestsPerSec  float64 `yaml:"requests_per_sec"`  // default 100
	BurstSize       int     `yaml:"burst_size"`        // default 200
	MaxConnections  int     `yaml:"max_connections"`   // WebSocket, default 1000
	MaxMessageBytes int     `yaml:"max_message_bytes"` // default 1MB
}

// AuditLogConfig holds audit logging settings.
type AuditLogConfig struct {
	Enabled bool   `yaml:"enabled"` // default true
	Output  string `yaml:"output"`  // "stdout" or "file:/path"
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPAddr: ":8080",
			GRPCAddr: ":9090",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "peerclaw.db",
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
			DB:   0,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		Bridge: BridgeConfig{
			A2A: BridgeEndpoint{Enabled: true},
			ACP: BridgeEndpoint{Enabled: true},
			MCP: BridgeEndpoint{Enabled: true},
		},
		Signaling: SignalingConfig{
			Enabled: true,
		},
		Observability: ObservabilityConfig{
			Enabled:        false,
			OTLPEndpoint:   "localhost:4317",
			ServiceName:    "peerclaw-gateway",
			TracesSampling: 0.1,
		},
		RateLimit: RateLimitConfig{
			Enabled:         true,
			RequestsPerSec:  100,
			BurstSize:       200,
			MaxConnections:  1000,
			MaxMessageBytes: 1 << 20, // 1MB
		},
		AuditLog: AuditLogConfig{
			Enabled: true,
			Output:  "stdout",
		},
	}
}

// Load reads configuration from a YAML file, falling back to defaults.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}
