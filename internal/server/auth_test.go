package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/peerclaw/peerclaw-core/identity"
	srvidentity "github.com/peerclaw/peerclaw-server/internal/identity"
)

func TestAuthMiddleware_NoCredentials_RequiredTrue(t *testing.T) {
	verifier := srvidentity.NewVerifier()
	cfg := AuthConfig{Required: true, Verifier: verifier}
	handler := AuthMiddleware(cfg, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_NoCredentials_RequiredFalse(t *testing.T) {
	verifier := srvidentity.NewVerifier()
	cfg := AuthConfig{Required: false, Verifier: verifier}
	handler := AuthMiddleware(cfg, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_ValidBearerToken(t *testing.T) {
	verifier := srvidentity.NewVerifier()
	verifier.RegisterKey("agent-1", "secret-key-123")
	cfg := AuthConfig{Required: true, Verifier: verifier}

	var gotAgentID string
	handler := AuthMiddleware(cfg, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := srvidentity.AgentIDFromContext(r.Context())
		gotAgentID = id
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer secret-key-123")
	req.Header.Set("X-PeerClaw-Agent-ID", "agent-1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotAgentID != "agent-1" {
		t.Errorf("agentID = %q, want %q", gotAgentID, "agent-1")
	}
}

func TestAuthMiddleware_InvalidBearerToken(t *testing.T) {
	verifier := srvidentity.NewVerifier()
	verifier.RegisterKey("agent-1", "secret-key-123")
	cfg := AuthConfig{Required: true, Verifier: verifier}

	handler := AuthMiddleware(cfg, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	req.Header.Set("X-PeerClaw-Agent-ID", "agent-1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ValidSignature(t *testing.T) {
	kp, err := identity.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	verifier := srvidentity.NewVerifier()
	cfg := AuthConfig{Required: true, Verifier: verifier}

	var gotAgentID string
	handler := AuthMiddleware(cfg, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := srvidentity.AgentIDFromContext(r.Context())
		gotAgentID = id
		w.WriteHeader(http.StatusOK)
	}))

	body := `{"test":"data"}`
	sig := identity.Sign(kp.PrivateKey, []byte(body))
	pubKeyStr := kp.PublicKeyString()

	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("X-PeerClaw-Signature", sig)
	req.Header.Set("X-PeerClaw-PublicKey", pubKeyStr)
	req.Header.Set("X-PeerClaw-Agent-ID", "agent-sig")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotAgentID != "agent-sig" {
		t.Errorf("agentID = %q, want %q", gotAgentID, "agent-sig")
	}
}

func TestAuthMiddleware_InvalidSignature(t *testing.T) {
	kp, err := identity.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	verifier := srvidentity.NewVerifier()
	cfg := AuthConfig{Required: true, Verifier: verifier}

	handler := AuthMiddleware(cfg, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := `{"test":"data"}`
	sig := identity.Sign(kp.PrivateKey, []byte("different data"))
	pubKeyStr := kp.PublicKeyString()

	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("X-PeerClaw-Signature", sig)
	req.Header.Set("X-PeerClaw-PublicKey", pubKeyStr)
	req.Header.Set("X-PeerClaw-Agent-ID", "agent-sig")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestOwnerOnlyMiddleware_OwnerMatch(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := OwnerOnlyMiddleware(nil)(inner)

	req := httptest.NewRequest("DELETE", "/api/v1/agents/agent-1", nil)
	req.SetPathValue("id", "agent-1")
	ctx := srvidentity.WithAgentID(req.Context(), "agent-1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOwnerOnlyMiddleware_OwnerMismatch(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := OwnerOnlyMiddleware(nil)(inner)

	req := httptest.NewRequest("DELETE", "/api/v1/agents/agent-1", nil)
	req.SetPathValue("id", "agent-1")
	ctx := srvidentity.WithAgentID(req.Context(), "agent-2")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}
