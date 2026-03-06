package config

import (
	"fmt"
	"os"
	"strings"

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
	Federation    FederationConfig    `yaml:"federation"`
	Auth          AuthConfig          `yaml:"auth"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	Required bool `yaml:"required"` // When true, reject unauthenticated requests. Default false for transition.
}

// ServerConfig holds HTTP and gRPC server settings.
type ServerConfig struct {
	HTTPAddr    string   `yaml:"http_addr"`
	GRPCAddr    string   `yaml:"grpc_addr"`
	CORSOrigins []string `yaml:"cors_origins"`
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

// FederationConfig holds server federation settings.
type FederationConfig struct {
	Enabled    bool             `yaml:"enabled"`
	NodeName   string           `yaml:"node_name"`
	Peers      []FederationPeer `yaml:"peers"`
	DNSEnabled bool             `yaml:"dns_enabled"`
	DNSDomain  string           `yaml:"dns_domain"`
	AuthToken  string           `yaml:"auth_token"`
}

// FederationPeer holds connection details for a federated peer server.
type FederationPeer struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
	Token   string `yaml:"token"`
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
		Federation: FederationConfig{
			Enabled: false,
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

	cfg.resolveSecrets()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// resolveSecrets resolves ${ENV_VAR} references in sensitive config fields.
func (c *Config) resolveSecrets() {
	c.Redis.Password = resolveEnv(c.Redis.Password)
	c.Database.DSN = resolveEnv(c.Database.DSN)
	c.Signaling.TURN.Credential = resolveEnv(c.Signaling.TURN.Credential)
	c.Federation.AuthToken = resolveEnv(c.Federation.AuthToken)
	for i := range c.Federation.Peers {
		c.Federation.Peers[i].Token = resolveEnv(c.Federation.Peers[i].Token)
	}
}

// resolveEnv replaces ${ENV_VAR} with the value of the environment variable.
// If the value doesn't match the pattern, it's returned as-is (backwards compatible).
func resolveEnv(val string) string {
	if !strings.HasPrefix(val, "${") || !strings.HasSuffix(val, "}") {
		return val
	}
	envName := val[2 : len(val)-1]
	if envVal, ok := os.LookupEnv(envName); ok {
		return envVal
	}
	return val
}

// validate checks configuration for invalid or dangerous settings.
func (c *Config) validate() error {
	// Federation requires an auth token when enabled.
	if c.Federation.Enabled && c.Federation.AuthToken == "" {
		return fmt.Errorf("federation.auth_token is required when federation is enabled")
	}
	return nil
}
