package server

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/peerclaw/peerclaw-core/envelope"
	"github.com/peerclaw/peerclaw-core/protocol"
	"github.com/peerclaw/peerclaw-server/internal/identity"
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
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
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

	// Verify authenticated agent matches the source field.
	ctxAgentID, ok := identity.AgentIDFromContext(r.Context())
	if ok && ctxAgentID != req.Source {
		s.jsonError(w, "source does not match authenticated agent", http.StatusForbidden)
		return
	}

	// Check contacts whitelist: destination agent must have source in its contact list.
	if s.contacts != nil {
		allowed, err := s.contacts.IsAllowed(r.Context(), req.Source, req.Destination)
		if err != nil {
			s.jsonError(w, "failed to check contacts", http.StatusInternalServerError)
			return
		}
		if !allowed {
			s.jsonError(w, "not in destination agent's contact list", http.StatusForbidden)
			return
		}
	}

	// Per-source-agent rate limiting for bridge sends.
	if s.bridgeRateLimiter != nil {
		if !s.bridgeRateLimiter.GetLimiter("bridge:"+req.Source).Allow() {
			s.jsonError(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
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
	if s.bridges == nil {
		s.jsonError(w, "bridge not available", http.StatusServiceUnavailable)
		return
	}
	bridgeStart := time.Now()
	if err := s.bridges.Send(r.Context(), env); err != nil {
		s.logger.Error("bridge send failed", "error", err, "proto", proto, "dest", req.Destination)
		// Record bridge error reputation event.
		if s.reputation != nil {
			_ = s.reputation.RecordEvent(r.Context(), req.Source, "bridge_error", err.Error())
		}
		s.jsonError(w, "bridge send failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Record bridge success reputation event.
	if s.reputation != nil {
		_ = s.reputation.RecordEvent(r.Context(), req.Source, "bridge_success", "")
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
		"status":      "sent",
		"protocol":    proto,
		"envelope_id": env.ID,
	})
}
