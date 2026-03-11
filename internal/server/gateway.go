package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (s *HTTPServer) registerGatewayRoutes() {
	s.mux.HandleFunc("POST /agent/{agent_id}", s.handleGatewayInvoke)
	s.mux.HandleFunc("GET /agent/{agent_id}", s.handleGatewayDiscover)
}

// handleGatewayInvoke handles POST /agent/{agent_id} — protocol auto-detection + dispatch.
func (s *HTTPServer) handleGatewayInvoke(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		s.jsonError(w, "failed to read body", http.StatusBadRequest)
		return
	}

	protocol := detectProtocol(body)

	// Record gateway metric.
	if s.metrics != nil && s.metrics.GatewayRequestsTotal != nil {
		s.metrics.GatewayRequestsTotal.Add(r.Context(), 1,
			metric.WithAttributes(attribute.String("protocol", protocol)),
		)
	}

	// Restore body for downstream handler.
	r.Body = io.NopCloser(bytes.NewReader(body))

	switch protocol {
	case "a2a":
		s.handleA2ABridgeMessages(w, r)
	case "mcp":
		s.handleMCPBridgeMessages(w, r)
	case "acp":
		s.handleACPBridgeCreateRun(w, r)
	default:
		s.jsonError(w, "unable to detect protocol from request body", http.StatusBadRequest)
	}
}

// handleGatewayDiscover handles GET /agent/{agent_id} — multi-format discovery.
func (s *HTTPServer) handleGatewayDiscover(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agent_id")
	if agentID == "" {
		s.jsonError(w, "missing agent_id", http.StatusBadRequest)
		return
	}

	card, err := s.registry.GetAgent(r.Context(), agentID)
	if err != nil {
		s.jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	format := r.URL.Query().Get("format")
	baseURL := requestBaseURL(r)

	w.Header().Set("Content-Type", "application/json")

	switch format {
	case "a2a":
		a2aCard := cardToA2AAgentCard(card, baseURL)
		_ = json.NewEncoder(w).Encode(a2aCard)
	case "mcp":
		mcpInfo := cardToMCPInfo(card.Name, card.Version, card.Description, baseURL+"/mcp/"+card.ID)
		_ = json.NewEncoder(w).Encode(mcpInfo)
	case "acp":
		manifest := cardToACPManifest(card)
		_ = json.NewEncoder(w).Encode(manifest)
	default:
		_ = json.NewEncoder(w).Encode(card)
	}
}

// detectProtocol inspects a JSON body and determines which protocol it belongs to.
func detectProtocol(body []byte) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return ""
	}

	// JSON-RPC: has "method" field → A2A or MCP.
	if methodRaw, ok := raw["method"]; ok {
		var method string
		if err := json.Unmarshal(methodRaw, &method); err != nil {
			return ""
		}

		// A2A methods.
		if hasPrefix(method, "message/", "tasks/") {
			return "a2a"
		}

		// MCP methods.
		if hasPrefix(method, "tools/", "resources/", "prompts/") || method == "initialize" {
			return "mcp"
		}

		// Fallback: inspect params shape.
		if paramsRaw, ok := raw["params"]; ok {
			var params map[string]json.RawMessage
			if json.Unmarshal(paramsRaw, &params) == nil {
				if _, ok := params["message"]; ok {
					return "a2a"
				}
				if _, ok := params["name"]; ok {
					return "mcp"
				}
			}
		}

		return ""
	}

	// ACP: has "input" field.
	if _, ok := raw["input"]; ok {
		return "acp"
	}

	// ACP: has "agent_name" field.
	if _, ok := raw["agent_name"]; ok {
		return "acp"
	}

	return ""
}

// hasPrefix checks if s starts with any of the given prefixes.
func hasPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if len(s) >= len(p) && s[:len(p)] == p {
			return true
		}
	}
	return false
}
