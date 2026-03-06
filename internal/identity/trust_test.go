package identity

import (
	"testing"

	pcidentity "github.com/peerclaw/peerclaw-core/identity"
)

func TestNewVerifier(t *testing.T) {
	v := NewVerifier()
	if v == nil {
		t.Fatal("NewVerifier returned nil")
	}
	if v.apiKeys == nil {
		t.Fatal("NewVerifier should initialize apiKeys map")
	}
	if len(v.apiKeys) != 0 {
		t.Errorf("apiKeys length = %d, want 0", len(v.apiKeys))
	}
}

func TestVerifyAPIKey_Valid(t *testing.T) {
	v := NewVerifier()

	agentID := "agent-001"
	apiKey := "test-api-key-abc123"

	v.RegisterKey(agentID, apiKey)

	if err := v.VerifyAPIKey(agentID, apiKey); err != nil {
		t.Errorf("VerifyAPIKey with valid key returned error: %v", err)
	}
}

func TestVerifyAPIKey_Invalid(t *testing.T) {
	v := NewVerifier()

	agentID := "agent-001"
	apiKey := "correct-key"

	v.RegisterKey(agentID, apiKey)

	// Wrong key
	if err := v.VerifyAPIKey(agentID, "wrong-key"); err == nil {
		t.Error("VerifyAPIKey with wrong key should return error")
	}

	// Unregistered agent
	if err := v.VerifyAPIKey("unknown-agent", "any-key"); err == nil {
		t.Error("VerifyAPIKey with unknown agent should return error")
	}
}

func TestVerifyAPIKey_RegisterAndRemove(t *testing.T) {
	v := NewVerifier()

	agentID := "agent-002"
	apiKey := "removable-key"

	v.RegisterKey(agentID, apiKey)
	if err := v.VerifyAPIKey(agentID, apiKey); err != nil {
		t.Fatalf("VerifyAPIKey after RegisterKey failed: %v", err)
	}

	v.RemoveKey(agentID)
	if err := v.VerifyAPIKey(agentID, apiKey); err == nil {
		t.Error("VerifyAPIKey after RemoveKey should return error")
	}
}

func TestVerifySignature_Valid(t *testing.T) {
	// Generate a real Ed25519 keypair using the core identity package.
	kp, err := pcidentity.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair failed: %v", err)
	}

	data := []byte("hello, peerclaw!")
	sig := pcidentity.Sign(kp.PrivateKey, data)

	v := NewVerifier()
	pubKeyStr := kp.PublicKeyString()

	if err := v.VerifySignature(pubKeyStr, data, sig); err != nil {
		t.Errorf("VerifySignature with valid signature returned error: %v", err)
	}
}

func TestVerifySignature_InvalidSignature(t *testing.T) {
	kp, err := pcidentity.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair failed: %v", err)
	}

	data := []byte("hello, peerclaw!")
	// Sign with the correct key but verify against different data.
	sig := pcidentity.Sign(kp.PrivateKey, data)

	v := NewVerifier()
	pubKeyStr := kp.PublicKeyString()

	// Tampered data should fail verification.
	tamperedData := []byte("tampered data!")
	if err := v.VerifySignature(pubKeyStr, tamperedData, sig); err == nil {
		t.Error("VerifySignature with tampered data should return error")
	}
}

func TestVerifySignature_WrongKey(t *testing.T) {
	kp1, err := pcidentity.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair failed: %v", err)
	}
	kp2, err := pcidentity.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair failed: %v", err)
	}

	data := []byte("hello, peerclaw!")
	sig := pcidentity.Sign(kp1.PrivateKey, data)

	v := NewVerifier()
	// Verify with a different public key.
	if err := v.VerifySignature(kp2.PublicKeyString(), data, sig); err == nil {
		t.Error("VerifySignature with wrong public key should return error")
	}
}

func TestVerifySignature_InvalidPublicKey(t *testing.T) {
	v := NewVerifier()
	if err := v.VerifySignature("not-valid-base64!!!", []byte("data"), "c2ln"); err == nil {
		t.Error("VerifySignature with invalid public key should return error")
	}
}

func TestVerifySignature_InvalidSignatureEncoding(t *testing.T) {
	kp, err := pcidentity.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair failed: %v", err)
	}

	v := NewVerifier()
	if err := v.VerifySignature(kp.PublicKeyString(), []byte("data"), "not-valid-base64!!!"); err == nil {
		t.Error("VerifySignature with invalid signature encoding should return error")
	}
}

func TestExtractBearerToken_Valid(t *testing.T) {
	token, err := ExtractBearerToken("Bearer my-secret-token")
	if err != nil {
		t.Fatalf("ExtractBearerToken returned error: %v", err)
	}
	if token != "my-secret-token" {
		t.Errorf("token = %q, want %q", token, "my-secret-token")
	}
}

func TestExtractBearerToken_CaseInsensitive(t *testing.T) {
	token, err := ExtractBearerToken("bearer my-token")
	if err != nil {
		t.Fatalf("ExtractBearerToken returned error: %v", err)
	}
	if token != "my-token" {
		t.Errorf("token = %q, want %q", token, "my-token")
	}

	token, err = ExtractBearerToken("BEARER MY-TOKEN")
	if err != nil {
		t.Fatalf("ExtractBearerToken returned error: %v", err)
	}
	if token != "MY-TOKEN" {
		t.Errorf("token = %q, want %q", token, "MY-TOKEN")
	}
}

func TestExtractBearerToken_EmptyHeader(t *testing.T) {
	_, err := ExtractBearerToken("")
	if err == nil {
		t.Error("ExtractBearerToken with empty header should return error")
	}
}

func TestExtractBearerToken_MissingScheme(t *testing.T) {
	_, err := ExtractBearerToken("my-token-only")
	if err == nil {
		t.Error("ExtractBearerToken without scheme should return error")
	}
}

func TestExtractBearerToken_WrongScheme(t *testing.T) {
	_, err := ExtractBearerToken("Basic dXNlcjpwYXNz")
	if err == nil {
		t.Error("ExtractBearerToken with Basic scheme should return error")
	}
}

func TestExtractBearerToken_TokenWithSpaces(t *testing.T) {
	// The token part can contain spaces (SplitN with limit 2).
	token, err := ExtractBearerToken("Bearer token with spaces")
	if err != nil {
		t.Fatalf("ExtractBearerToken returned error: %v", err)
	}
	if token != "token with spaces" {
		t.Errorf("token = %q, want %q", token, "token with spaces")
	}
}
