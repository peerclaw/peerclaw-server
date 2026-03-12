package federation

import (
	"testing"
)

func TestFederationServiceNew(t *testing.T) {
	fs := New("node-a", "secret-token", nil)
	if fs.AuthToken() != "secret-token" {
		t.Errorf("expected auth token 'secret-token', got %q", fs.AuthToken())
	}
}

func TestFederationAddPeer(t *testing.T) {
	fs := New("node-a", "", nil)
	fs.AddPeer("node-b", "http://localhost:8081", "token-b")
	fs.AddPeer("node-c", "http://localhost:8082", "token-c")

	if len(fs.peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(fs.peers))
	}
}
