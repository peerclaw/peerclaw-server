package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/peerclaw/peerclaw-go/agentcard"
	"github.com/peerclaw/peerclaw-go/protocol"
	"github.com/peerclaw/peerclaw-server/internal/bridge"
	"github.com/peerclaw/peerclaw-server/internal/registry"
	"github.com/peerclaw/peerclaw-server/internal/router"
	"github.com/peerclaw/peerclaw-server/internal/signaling"
)

// HTTPServer provides the REST API gateway for PeerClaw.
type HTTPServer struct {
	mux      *http.ServeMux
	server   *http.Server
	registry *registry.Service
	engine   *router.Engine
	bridges  *bridge.Manager
	sigHub   *signaling.Hub
	logger   *slog.Logger
}

// NewHTTPServer creates a new HTTP server with all routes registered.
func NewHTTPServer(addr string, reg *registry.Service, eng *router.Engine, brg *bridge.Manager, sigHub *signaling.Hub, logger *slog.Logger) *HTTPServer {
	if logger == nil {
		logger = slog.Default()
	}
	s := &HTTPServer{
		mux:      http.NewServeMux(),
		registry: reg,
		engine:   eng,
		bridges:  brg,
		sigHub:   sigHub,
		logger:   logger,
	}
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	s.routes()
	return s
}

func (s *HTTPServer) routes() {
	s.mux.HandleFunc("POST /api/v1/agents", s.handleRegister)
	s.mux.HandleFunc("GET /api/v1/agents", s.handleListAgents)
	s.mux.HandleFunc("GET /api/v1/agents/{id}", s.handleGetAgent)
	s.mux.HandleFunc("DELETE /api/v1/agents/{id}", s.handleDeregister)
	s.mux.HandleFunc("POST /api/v1/agents/{id}/heartbeat", s.handleHeartbeat)
	s.mux.HandleFunc("POST /api/v1/discover", s.handleDiscover)
	s.mux.HandleFunc("GET /api/v1/routes", s.handleGetRoutes)
	s.mux.HandleFunc("GET /api/v1/routes/resolve", s.handleResolveRoute)
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	if s.sigHub != nil {
		s.mux.HandleFunc("GET /api/v1/signaling", s.sigHub.HandleConnect)
	}
}

// Start begins listening and serving HTTP requests.
func (s *HTTPServer) Start() error {
	s.logger.Info("HTTP server listening", "addr", s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server.
func (s *HTTPServer) Stop() error {
	return s.server.Close()
}

// --- Handlers ---

type registerRequest struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Version      string            `json:"version"`
	PublicKey    string            `json:"public_key"`
	Capabilities []string          `json:"capabilities"`
	Endpoint     endpointReq       `json:"endpoint"`
	Protocols    []string          `json:"protocols"`
	Auth         authReq           `json:"auth"`
	Metadata     map[string]string `json:"metadata"`
	PeerClaw     peerclawReq       `json:"peerclaw"`
}

type endpointReq struct {
	URL       string `json:"url"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Transport string `json:"transport"`
}

type authReq struct {
	Type   string            `json:"type"`
	Params map[string]string `json:"params"`
}

type peerclawReq struct {
	NATType         string   `json:"nat_type"`
	RelayPreference string   `json:"relay_preference"`
	Priority        int      `json:"priority"`
	Tags            []string `json:"tags"`
}

func (s *HTTPServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	protocols := make([]protocol.Protocol, len(req.Protocols))
	for i, p := range req.Protocols {
		protocols[i] = protocol.Protocol(p)
	}

	card, err := s.registry.Register(r.Context(), registry.RegisterRequest{
		Name:         req.Name,
		Description:  req.Description,
		Version:      req.Version,
		PublicKey:    req.PublicKey,
		Capabilities: req.Capabilities,
		Endpoint: agentcard.Endpoint{
			URL:       req.Endpoint.URL,
			Host:      req.Endpoint.Host,
			Port:      req.Endpoint.Port,
			Transport: protocol.Transport(req.Endpoint.Transport),
		},
		Protocols: protocols,
		Auth: agentcard.AuthInfo{
			Type:   req.Auth.Type,
			Params: req.Auth.Params,
		},
		Metadata: req.Metadata,
		PeerClaw: agentcard.PeerClawExtension{
			NATType:         req.PeerClaw.NATType,
			RelayPreference: req.PeerClaw.RelayPreference,
			Priority:        req.PeerClaw.Priority,
			Tags:            req.PeerClaw.Tags,
		},
	})
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update routing table.
	s.engine.UpdateFromCard(card)

	s.jsonResponse(w, http.StatusCreated, card)
}

func (s *HTTPServer) handleListAgents(w http.ResponseWriter, r *http.Request) {
	filter := registry.ListFilter{
		Protocol:   r.URL.Query().Get("protocol"),
		Capability: r.URL.Query().Get("capability"),
		Status:     agentcard.AgentStatus(r.URL.Query().Get("status")),
		PageToken:  r.URL.Query().Get("page_token"),
	}
	result, err := s.registry.ListAgents(r.Context(), filter)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, http.StatusOK, result)
}

func (s *HTTPServer) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	card, err := s.registry.GetAgent(r.Context(), id)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	s.jsonResponse(w, http.StatusOK, card)
}

func (s *HTTPServer) handleDeregister(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.registry.Deregister(r.Context(), id); err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	s.engine.RemoveAgent(id)
	w.WriteHeader(http.StatusNoContent)
}

type heartbeatRequest struct {
	Status   string            `json:"status"`
	Metadata map[string]string `json:"metadata"`
}

func (s *HTTPServer) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req heartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	status := agentcard.AgentStatus(req.Status)
	if status == "" {
		status = agentcard.StatusOnline
	}

	deadline, err := s.registry.Heartbeat(r.Context(), id, status)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]any{"next_deadline": deadline})
}

type discoverRequest struct {
	Capabilities []string `json:"capabilities"`
	Protocol     string   `json:"protocol"`
	MaxResults   int      `json:"max_results"`
}

func (s *HTTPServer) handleDiscover(w http.ResponseWriter, r *http.Request) {
	var req discoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	agents, err := s.registry.Discover(r.Context(), req.Capabilities, req.Protocol, req.MaxResults)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]any{"agents": agents})
}

func (s *HTTPServer) handleGetRoutes(w http.ResponseWriter, r *http.Request) {
	routes := s.engine.Table().AllRoutes()
	s.jsonResponse(w, http.StatusOK, map[string]any{
		"routes":     routes,
		"updated_at": s.engine.Table().UpdatedAt(),
	})
}

func (s *HTTPServer) handleResolveRoute(w http.ResponseWriter, r *http.Request) {
	opts := router.ResolveOptions{
		TargetID: r.URL.Query().Get("target_id"),
		Protocol: r.URL.Query().Get("protocol"),
	}
	route, err := s.engine.Resolve(opts)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	s.jsonResponse(w, http.StatusOK, route)
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Helpers ---

func (s *HTTPServer) jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *HTTPServer) jsonError(w http.ResponseWriter, message string, status int) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}
