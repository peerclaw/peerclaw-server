package server

import (
	"strings"
	"testing"

	"github.com/peerclaw/peerclaw-core/identity"
)

func TestValidateRegisterRequest_Valid(t *testing.T) {
	kp, _ := identity.GenerateKeypair()
	req := &registerRequest{
		Name:         "TestAgent",
		PublicKey:    kp.PublicKeyString(),
		Capabilities: []string{"chat", "search"},
		Endpoint:     endpointReq{URL: "https://agent.example.com", Port: 8080},
		Protocols:    []string{"a2a", "mcp"},
	}
	if err := validateRegisterRequest(req); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateRegisterRequest_EmptyName(t *testing.T) {
	req := &registerRequest{}
	err := validateRegisterRequest(req)
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Errorf("expected name error, got: %v", err)
	}
}

func TestValidateRegisterRequest_LongName(t *testing.T) {
	req := &registerRequest{Name: strings.Repeat("a", 257)}
	err := validateRegisterRequest(req)
	if err == nil || !strings.Contains(err.Error(), "at most 256") {
		t.Errorf("expected name length error, got: %v", err)
	}
}

func TestValidateRegisterRequest_ControlCharsInName(t *testing.T) {
	req := &registerRequest{Name: "test\x00agent"}
	err := validateRegisterRequest(req)
	if err == nil || !strings.Contains(err.Error(), "control characters") {
		t.Errorf("expected control char error, got: %v", err)
	}
}

func TestValidateRegisterRequest_InvalidPublicKey(t *testing.T) {
	req := &registerRequest{Name: "test", PublicKey: "not-valid-base64!!!"}
	err := validateRegisterRequest(req)
	if err == nil || !strings.Contains(err.Error(), "base64") {
		t.Errorf("expected base64 error, got: %v", err)
	}
}

func TestValidateRegisterRequest_WrongSizePublicKey(t *testing.T) {
	req := &registerRequest{Name: "test", PublicKey: "AQIDBA=="}
	err := validateRegisterRequest(req)
	if err == nil || !strings.Contains(err.Error(), "32 bytes") {
		t.Errorf("expected key size error, got: %v", err)
	}
}

func TestValidateRegisterRequest_TooManyCapabilities(t *testing.T) {
	caps := make([]string, 51)
	for i := range caps {
		caps[i] = "cap"
	}
	req := &registerRequest{Name: "test", Capabilities: caps}
	err := validateRegisterRequest(req)
	if err == nil || !strings.Contains(err.Error(), "50") {
		t.Errorf("expected capabilities limit error, got: %v", err)
	}
}

func TestValidateRegisterRequest_UnknownProtocol(t *testing.T) {
	req := &registerRequest{Name: "test", Protocols: []string{"a2a", "fake"}}
	err := validateRegisterRequest(req)
	if err == nil || !strings.Contains(err.Error(), "unknown protocol") {
		t.Errorf("expected protocol error, got: %v", err)
	}
}

func TestValidateRegisterRequest_InvalidURL(t *testing.T) {
	req := &registerRequest{Name: "test", Endpoint: endpointReq{URL: "ftp://bad"}}
	err := validateRegisterRequest(req)
	if err == nil || !strings.Contains(err.Error(), "http/https") {
		t.Errorf("expected URL error, got: %v", err)
	}
}

func TestValidateHeartbeatStatus(t *testing.T) {
	tests := []struct {
		status  string
		wantErr bool
	}{
		{"", false},
		{"online", false},
		{"busy", false},
		{"offline", false},
		{"invalid", true},
	}
	for _, tt := range tests {
		err := validateHeartbeatStatus(tt.status)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateHeartbeatStatus(%q) err=%v, wantErr=%v", tt.status, err, tt.wantErr)
		}
	}
}
