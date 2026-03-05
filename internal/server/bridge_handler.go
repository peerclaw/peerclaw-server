package server

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/router"
)

func routerResolveOptions(targetID, proto string) router.ResolveOptions {
	return router.ResolveOptions{
		TargetID: targetID,
		Protocol: proto,
	}
}

// bridgeSendRequest is the request body for POST /api/v1/bridge/send.
type bridgeSendRequest struct {
	Source      string            `json:"source"`
	Destination string            `json:"destination"`
	Protocol    string            `json:"protocol"`
	Payload     string            `json:"payload"`
	Metadata    map[string]string `json:"metadata"`
}

// handleBridgeSend handles POST /api/v1/bridge/send.
// This allows PeerClaw agents to send messages to external agents via the bridge.
func (s *HTTPServer) handleBridgeSend(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.jsonError(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var req bridgeSendRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.jsonError(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Source == "" || req.Destination == "" {
		s.jsonError(w, "source and destination are required", http.StatusBadRequest)
		return
	}

	proto := req.Protocol
	if proto == "" {
		// Try to resolve protocol from routing table.
		route, err := s.engine.Resolve(routerResolveOptions(req.Destination, ""))
		if err == nil {
			proto = route.Protocol
		} else {
			s.jsonError(w, "protocol required: no route found for destination", http.StatusBadRequest)
			return
		}
	}

	if !protocol.Protocol(proto).Valid() {
		s.jsonError(w, "invalid protocol: "+proto, http.StatusBadRequest)
		return
	}

	// Build envelope.
	env := envelope.New(req.Source, req.Destination, protocol.Protocol(proto), []byte(req.Payload))
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			env.Metadata[k] = v
		}
	}

	// If endpoint not in metadata, try to resolve from routing.
	endpointKey := proto + ".endpoint"
	if env.Metadata[endpointKey] == "" {
		route, err := s.engine.Resolve(routerResolveOptions(req.Destination, proto))
		if err == nil && route.Endpoint != "" {
			env.Metadata[endpointKey] = route.Endpoint
		}
	}

	// Send via bridge manager.
	bridgeStart := time.Now()
	if err := s.bridges.Send(r.Context(), env); err != nil {
		s.logger.Error("bridge send failed", "error", err, "proto", proto, "dest", req.Destination)
		s.jsonError(w, "bridge send failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Audit log and metrics.
	if s.audit != nil {
		s.audit.LogBridgeSend(r.Context(), req.Source, req.Destination, proto)
	}
	if s.metrics != nil {
		s.metrics.BridgeMessagesTotal.Add(r.Context(), 1)
		s.metrics.BridgeMessageDuration.Record(r.Context(), time.Since(bridgeStart).Seconds())
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"status":   "sent",
		"protocol": proto,
		"envelope_id": env.ID,
	})
}
