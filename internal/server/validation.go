package server

import (
	"encoding/base64"
	"crypto/ed25519"
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

// knownProtocols are the valid protocol values for agent registration.
var knownProtocols = map[string]bool{
	"a2a":      true,
	"mcp":      true,
	"acp":      true,
	"custom":   true,
	"peerclaw": true,
}

// validStatuses are the valid heartbeat status values.
var validStatuses = map[string]bool{
	"online":  true,
	"busy":    true,
	"offline": true,
}

func validateRegisterRequest(req *registerRequest) error {
	// Name: 1-256 chars, no control characters.
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(req.Name) > 256 {
		return fmt.Errorf("name must be at most 256 characters")
	}
	if containsControlChars(req.Name) {
		return fmt.Errorf("name must not contain control characters")
	}

	// PublicKey: if provided, must be valid base64-encoded Ed25519 key (32 bytes).
	if req.PublicKey != "" {
		keyBytes, err := base64.StdEncoding.DecodeString(req.PublicKey)
		if err != nil {
			return fmt.Errorf("public_key must be valid base64: %w", err)
		}
		if len(keyBytes) != ed25519.PublicKeySize {
			return fmt.Errorf("public_key must be %d bytes (Ed25519), got %d", ed25519.PublicKeySize, len(keyBytes))
		}
	}

	// Capabilities: max 50 items, each ≤128 chars.
	if len(req.Capabilities) > 50 {
		return fmt.Errorf("capabilities must have at most 50 items")
	}
	for _, cap := range req.Capabilities {
		if len(cap) > 128 {
			return fmt.Errorf("each capability must be at most 128 characters")
		}
	}

	// Endpoint URL: if provided, must be valid URL.
	if req.Endpoint.URL != "" {
		u, err := url.Parse(req.Endpoint.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("endpoint URL must be a valid http/https URL")
		}
	}

	// Endpoint Port: 1-65535.
	if req.Endpoint.Port < 0 || req.Endpoint.Port > 65535 {
		return fmt.Errorf("endpoint port must be between 0 and 65535")
	}

	// Protocols: max 10, must be known.
	if len(req.Protocols) > 10 {
		return fmt.Errorf("protocols must have at most 10 items")
	}
	for _, p := range req.Protocols {
		if !knownProtocols[strings.ToLower(p)] {
			return fmt.Errorf("unknown protocol: %s", p)
		}
	}

	// Metadata: max 50 keys, key ≤128, value ≤1024.
	if len(req.Metadata) > 50 {
		return fmt.Errorf("metadata must have at most 50 keys")
	}
	for k, v := range req.Metadata {
		if len(k) > 128 {
			return fmt.Errorf("metadata key must be at most 128 characters")
		}
		if len(v) > 1024 {
			return fmt.Errorf("metadata value must be at most 1024 characters")
		}
	}

	return nil
}

func validateHeartbeatStatus(status string) error {
	if status == "" {
		return nil // Empty means default to "online"
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status %q: must be one of online, busy, offline", status)
	}
	return nil
}

func containsControlChars(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
