package federation

import (
	"testing"
)

func TestFederationServiceNew(t *testing.T) {
	fs := New("node-a", "secret-token", nil)
	if fs.NodeName() != "node-a" {
		t.Errorf("expected node name 'node-a', got %q", fs.NodeName())
	}
	if fs.AuthToken() != "secret-token" {
		t.Errorf("expected auth token 'secret-token', got %q", fs.AuthToken())
	}
}

func TestFederationAddListPeers(t *testing.T) {
	fs := New("node-a", "", nil)
	fs.AddPeer("node-b", "http://localhost:8081", "token-b")
	fs.AddPeer("node-c", "http://localhost:8082", "token-c")

	peers := fs.ListPeers()
	if len(peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(peers))
	}

	names := map[string]bool{}
	for _, p := range peers {
		names[p.Name] = true
	}
	if !names["node-b"] || !names["node-c"] {
		t.Errorf("expected peers node-b and node-c, got %v", names)
	}
}
