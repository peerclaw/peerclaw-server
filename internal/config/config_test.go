package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Server defaults
	if cfg.Server.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.Server.HTTPAddr, ":8080")
	}
	// Database defaults
	if cfg.Database.Driver != "sqlite" {
		t.Errorf("Driver = %q, want %q", cfg.Database.Driver, "sqlite")
	}
	if cfg.Database.DSN != "peerclaw.db" {
		t.Errorf("DSN = %q, want %q", cfg.Database.DSN, "peerclaw.db")
	}

	// Redis defaults
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("Redis.Addr = %q, want %q", cfg.Redis.Addr, "localhost:6379")
	}
	if cfg.Redis.DB != 0 {
		t.Errorf("Redis.DB = %d, want 0", cfg.Redis.DB)
	}

	// Logging defaults
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "info")
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("Logging.Format = %q, want %q", cfg.Logging.Format, "text")
	}

	// Bridge defaults: all enabled
	if !cfg.Bridge.A2A.Enabled {
		t.Error("Bridge.A2A.Enabled = false, want true")
	}
	if !cfg.Bridge.ACP.Enabled {
		t.Error("Bridge.ACP.Enabled = false, want true")
	}
	if !cfg.Bridge.MCP.Enabled {
		t.Error("Bridge.MCP.Enabled = false, want true")
	}

	// Signaling default
	if !cfg.Signaling.Enabled {
		t.Error("Signaling.Enabled = false, want true")
	}

	// Observability defaults
	if cfg.Observability.Enabled {
		t.Error("Observability.Enabled = true, want false")
	}
	if cfg.Observability.ServiceName != "peerclaw-gateway" {
		t.Errorf("Observability.ServiceName = %q, want %q", cfg.Observability.ServiceName, "peerclaw-gateway")
	}
	if cfg.Observability.TracesSampling != 0.1 {
		t.Errorf("Observability.TracesSampling = %v, want 0.1", cfg.Observability.TracesSampling)
	}

	// RateLimit defaults
	if !cfg.RateLimit.Enabled {
		t.Error("RateLimit.Enabled = false, want true")
	}
	if cfg.RateLimit.RequestsPerSec != 100 {
		t.Errorf("RateLimit.RequestsPerSec = %v, want 100", cfg.RateLimit.RequestsPerSec)
	}
	if cfg.RateLimit.BurstSize != 200 {
		t.Errorf("RateLimit.BurstSize = %d, want 200", cfg.RateLimit.BurstSize)
	}
	if cfg.RateLimit.MaxConnections != 1000 {
		t.Errorf("RateLimit.MaxConnections = %d, want 1000", cfg.RateLimit.MaxConnections)
	}
	if cfg.RateLimit.MaxMessageBytes != 1<<20 {
		t.Errorf("RateLimit.MaxMessageBytes = %d, want %d", cfg.RateLimit.MaxMessageBytes, 1<<20)
	}

	// AuditLog defaults
	if !cfg.AuditLog.Enabled {
		t.Error("AuditLog.Enabled = false, want true")
	}
	if cfg.AuditLog.Output != "stdout" {
		t.Errorf("AuditLog.Output = %q, want %q", cfg.AuditLog.Output, "stdout")
	}

	// Federation defaults
	if cfg.Federation.Enabled {
		t.Error("Federation.Enabled = true, want false")
	}
}

func TestLoad_EmptyPath(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load(\"\") returned nil config")
	}

	// Should match defaults
	def := DefaultConfig()
	if cfg.Server.HTTPAddr != def.Server.HTTPAddr {
		t.Errorf("HTTPAddr = %q, want %q", cfg.Server.HTTPAddr, def.Server.HTTPAddr)
	}
	if cfg.Database.Driver != def.Database.Driver {
		t.Errorf("Driver = %q, want %q", cfg.Database.Driver, def.Database.Driver)
	}
}

func TestLoad_YAML(t *testing.T) {
	yamlContent := `
server:
  http_addr: ":3000"
database:
  driver: postgres
  dsn: "postgres://user:pass@localhost/peerclaw"
redis:
  addr: "redis.example.com:6379"
  db: 2
logging:
  level: debug
  format: json
bridge:
  a2a:
    enabled: false
  acp:
    enabled: true
  mcp:
    enabled: false
signaling:
  enabled: false
  turn:
    urls:
      - "turn:turn.example.com:3478"
    username: "turnuser"
    credential: "turncred"
observability:
  enabled: true
  otlp_endpoint: "otel.example.com:4317"
  service_name: "my-gateway"
  traces_sampling: 0.5
rate_limit:
  enabled: false
  requests_per_sec: 50
  burst_size: 100
  max_connections: 500
  max_message_bytes: 2097152
audit_log:
  enabled: false
  output: "file:/var/log/audit.log"
federation:
  enabled: true
  node_name: "node-1"
  auth_token: "secret-token"
  dns_enabled: true
  dns_domain: "peerclaw.example.com"
  peers:
    - name: "node-2"
      address: "node2.example.com:9090"
      token: "peer-token"
`

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load(%q) returned error: %v", cfgPath, err)
	}

	// Server
	if cfg.Server.HTTPAddr != ":3000" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.Server.HTTPAddr, ":3000")
	}
	// Database
	if cfg.Database.Driver != "postgres" {
		t.Errorf("Driver = %q, want %q", cfg.Database.Driver, "postgres")
	}
	if cfg.Database.DSN != "postgres://user:pass@localhost/peerclaw" {
		t.Errorf("DSN = %q, want %q", cfg.Database.DSN, "postgres://user:pass@localhost/peerclaw")
	}

	// Redis
	if cfg.Redis.Addr != "redis.example.com:6379" {
		t.Errorf("Redis.Addr = %q, want %q", cfg.Redis.Addr, "redis.example.com:6379")
	}
	if cfg.Redis.DB != 2 {
		t.Errorf("Redis.DB = %d, want 2", cfg.Redis.DB)
	}

	// Logging
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("Logging.Format = %q, want %q", cfg.Logging.Format, "json")
	}

	// Bridge
	if cfg.Bridge.A2A.Enabled {
		t.Error("Bridge.A2A.Enabled = true, want false")
	}
	if !cfg.Bridge.ACP.Enabled {
		t.Error("Bridge.ACP.Enabled = false, want true")
	}
	if cfg.Bridge.MCP.Enabled {
		t.Error("Bridge.MCP.Enabled = true, want false")
	}

	// Signaling
	if cfg.Signaling.Enabled {
		t.Error("Signaling.Enabled = true, want false")
	}
	if len(cfg.Signaling.TURN.URLs) != 1 || cfg.Signaling.TURN.URLs[0] != "turn:turn.example.com:3478" {
		t.Errorf("Signaling.TURN.URLs = %v, want [turn:turn.example.com:3478]", cfg.Signaling.TURN.URLs)
	}
	if cfg.Signaling.TURN.Username != "turnuser" {
		t.Errorf("Signaling.TURN.Username = %q, want %q", cfg.Signaling.TURN.Username, "turnuser")
	}

	// Observability
	if !cfg.Observability.Enabled {
		t.Error("Observability.Enabled = false, want true")
	}
	if cfg.Observability.OTLPEndpoint != "otel.example.com:4317" {
		t.Errorf("Observability.OTLPEndpoint = %q, want %q", cfg.Observability.OTLPEndpoint, "otel.example.com:4317")
	}
	if cfg.Observability.ServiceName != "my-gateway" {
		t.Errorf("Observability.ServiceName = %q, want %q", cfg.Observability.ServiceName, "my-gateway")
	}
	if cfg.Observability.TracesSampling != 0.5 {
		t.Errorf("Observability.TracesSampling = %v, want 0.5", cfg.Observability.TracesSampling)
	}

	// RateLimit
	if cfg.RateLimit.Enabled {
		t.Error("RateLimit.Enabled = true, want false")
	}
	if cfg.RateLimit.RequestsPerSec != 50 {
		t.Errorf("RateLimit.RequestsPerSec = %v, want 50", cfg.RateLimit.RequestsPerSec)
	}
	if cfg.RateLimit.BurstSize != 100 {
		t.Errorf("RateLimit.BurstSize = %d, want 100", cfg.RateLimit.BurstSize)
	}
	if cfg.RateLimit.MaxConnections != 500 {
		t.Errorf("RateLimit.MaxConnections = %d, want 500", cfg.RateLimit.MaxConnections)
	}
	if cfg.RateLimit.MaxMessageBytes != 2097152 {
		t.Errorf("RateLimit.MaxMessageBytes = %d, want 2097152", cfg.RateLimit.MaxMessageBytes)
	}

	// AuditLog
	if cfg.AuditLog.Enabled {
		t.Error("AuditLog.Enabled = true, want false")
	}
	if cfg.AuditLog.Output != "file:/var/log/audit.log" {
		t.Errorf("AuditLog.Output = %q, want %q", cfg.AuditLog.Output, "file:/var/log/audit.log")
	}

	// Federation
	if !cfg.Federation.Enabled {
		t.Error("Federation.Enabled = false, want true")
	}
	if cfg.Federation.NodeName != "node-1" {
		t.Errorf("Federation.NodeName = %q, want %q", cfg.Federation.NodeName, "node-1")
	}
	if cfg.Federation.AuthToken != "secret-token" {
		t.Errorf("Federation.AuthToken = %q, want %q", cfg.Federation.AuthToken, "secret-token")
	}
	if !cfg.Federation.DNSEnabled {
		t.Error("Federation.DNSEnabled = false, want true")
	}
	if cfg.Federation.DNSDomain != "peerclaw.example.com" {
		t.Errorf("Federation.DNSDomain = %q, want %q", cfg.Federation.DNSDomain, "peerclaw.example.com")
	}
	if len(cfg.Federation.Peers) != 1 {
		t.Fatalf("Federation.Peers length = %d, want 1", len(cfg.Federation.Peers))
	}
	peer := cfg.Federation.Peers[0]
	if peer.Name != "node-2" {
		t.Errorf("Peer.Name = %q, want %q", peer.Name, "node-2")
	}
	if peer.Address != "node2.example.com:9090" {
		t.Errorf("Peer.Address = %q, want %q", peer.Address, "node2.example.com:9090")
	}
	if peer.Token != "peer-token" {
		t.Errorf("Peer.Token = %q, want %q", peer.Token, "peer-token")
	}
}

func TestLoad_FederationRequiresToken(t *testing.T) {
	yamlContent := `
federation:
  enabled: true
  node_name: "node-1"
  auth_token: ""
`
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("Load should fail when federation is enabled without auth_token")
	}

	expected := "federation.auth_token is required when federation is enabled"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestResolveEnv(t *testing.T) {
	const envKey = "PEERCLAW_TEST_REDIS_PASS"
	const envVal = "super-secret-password"

	t.Setenv(envKey, envVal)

	yamlContent := `
redis:
  password: "${PEERCLAW_TEST_REDIS_PASS}"
`
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Redis.Password != envVal {
		t.Errorf("Redis.Password = %q, want %q", cfg.Redis.Password, envVal)
	}
}

func TestResolveEnv_UnsetVariable(t *testing.T) {
	// When the env var is not set, the placeholder should be preserved as-is.
	const envKey = "PEERCLAW_TEST_UNSET_VAR_12345"
	os.Unsetenv(envKey)

	yamlContent := `
redis:
  password: "${PEERCLAW_TEST_UNSET_VAR_12345}"
`
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	expected := "${PEERCLAW_TEST_UNSET_VAR_12345}"
	if cfg.Redis.Password != expected {
		t.Errorf("Redis.Password = %q, want %q (unset env var should be preserved)", cfg.Redis.Password, expected)
	}
}

func TestResolveEnv_PlainValue(t *testing.T) {
	// A plain value (not wrapped in ${...}) should be returned as-is.
	yamlContent := `
redis:
  password: "plain-password"
`
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Redis.Password != "plain-password" {
		t.Errorf("Redis.Password = %q, want %q", cfg.Redis.Password, "plain-password")
	}
}

func TestLoad_NonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Load should fail for nonexistent file")
	}
}
