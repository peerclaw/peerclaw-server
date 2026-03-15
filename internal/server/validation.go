package server

import (
	"fmt"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/protocol"
)

// validStatuses are the valid heartbeat status values.
var validStatuses = map[string]bool{
	"online":   true,
	"busy":     true,
	"degraded": true,
	"offline":  true,
}

// validateRegisterRequest validates a registration request by building a Card
// and delegating to Card.Validate().
func validateRegisterRequest(req *registerRequest) error {
	protocols := make([]protocol.Protocol, len(req.Protocols))
	for i, p := range req.Protocols {
		protocols[i] = protocol.Protocol(p)
	}

	card := &agentcard.Card{
		Name:         req.Name,
		PublicKey:    req.PublicKey,
		Capabilities: req.Capabilities,
		Endpoint: agentcard.Endpoint{
			URL:  req.Endpoint.URL,
			Port: req.Endpoint.Port,
		},
		Protocols: protocols,
		Metadata:  req.Metadata,
	}
	return card.Validate()
}

func validateHeartbeatStatus(status string) error {
	if status == "" {
		return nil // Empty means default to "online"
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status %q: must be one of online, busy, degraded, offline", status)
	}
	return nil
}

