package server

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/protocol"
	coresignaling "github.com/peerclaw/peerclaw-core/signaling"
	"github.com/peerclaw/peerclaw-server/internal/audit"
	"github.com/peerclaw/peerclaw-server/internal/bridge"
	"github.com/peerclaw/peerclaw-server/internal/federation"
	"github.com/peerclaw/peerclaw-server/internal/identity"
	"github.com/peerclaw/peerclaw-server/internal/observability"
	"github.com/peerclaw/peerclaw-server/internal/registry"
	"github.com/peerclaw/peerclaw-server/internal/reputation"
	"github.com/peerclaw/peerclaw-server/internal/router"
	"github.com/peerclaw/peerclaw-server/internal/signaling"
	"github.com/peerclaw/peerclaw-server/internal/invocation"
	"github.com/peerclaw/peerclaw-server/internal/review"
	"github.com/peerclaw/peerclaw-server/internal/userauth"
	"github.com/peerclaw/peerclaw-server/internal/verification"
	"go.opentelemetry.io/otel/trace"
)

// HTTPServerConfig holds optional settings for the HTTP server.
type HTTPServerConfig struct {
	RateLimiter  *IPRateLimiter
	MaxBodyBytes int64 // 0 means no limit
	Metrics      *observability.Metrics
	Tracer       trace.Tracer
	Auth         AuthConfig
	CORSOrigins  []string
}

// HTTPServer provides the REST API gateway for PeerClaw.
type HTTPServer struct {
	mux                    *http.ServeMux
	server                 *http.Server
	registry               *registry.Service
	engine                 *router.Engine
	bridges                *bridge.Manager
	sigHub                 *signaling.Hub
	logger                 *slog.Logger
	store                  registry.Store
	audit                  *audit.Logger
	metrics                *observability.Metrics
	federation             *federation.FederationService
	verifier               *identity.Verifier
	authCfg                AuthConfig
	reputation             *reputation.Engine
	verificationChallenger *verification.Challenger
	userAuth               *userauth.Service
	invocation             *invocation.Service
	invokeRateLimiter      *IPRateLimiter
	reviewService          *review.Service
}

// NewHTTPServer creates a new HTTP server with all routes registered.
func NewHTTPServer(addr string, reg *registry.Service, eng *router.Engine, brg *bridge.Manager, sigHub *signaling.Hub, logger *slog.Logger, opts *HTTPServerConfig) *HTTPServer {
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
		verifier: identity.NewVerifier(),
	}

	if opts != nil {
		s.authCfg = opts.Auth
	}
	if s.authCfg.Verifier == nil {
		s.authCfg.Verifier = s.verifier
	}

	s.routes()

	// Build middleware chain.
	middlewares := []Middleware{
		RecoveryMiddleware(logger),
		RequestIDMiddleware(),
		LoggingMiddleware(logger),
	}

	if opts != nil {
		if len(opts.CORSOrigins) > 0 {
			middlewares = append(middlewares, CORSMiddleware(opts.CORSOrigins))
		}
		if opts.Tracer != nil {
			middlewares = append(middlewares, TracingMiddleware(opts.Tracer))
		}
		if opts.Metrics != nil {
			middlewares = append(middlewares, MetricsMiddleware(opts.Metrics))
		}
		if opts.RateLimiter != nil {
			middlewares = append(middlewares, RateLimitMiddleware(opts.RateLimiter, logger))
		}
		if opts.MaxBodyBytes > 0 {
			middlewares = append(middlewares, MaxBodyMiddleware(opts.MaxBodyBytes))
		}
	}

	s.server = &http.Server{
		Addr:         addr,
		Handler:      Chain(s.mux, middlewares...),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s
}

// SetStore sets the Store for health checks to report database status.
func (s *HTTPServer) SetStore(store registry.Store) {
	s.store = store
}

// SetAudit sets the audit logger for recording audit events.
func (s *HTTPServer) SetAudit(a *audit.Logger) {
	s.audit = a
}

// SetMetrics sets the metrics instruments for observability.
func (s *HTTPServer) SetMetrics(m *observability.Metrics) {
	s.metrics = m
}

// SetFederation sets the federation service for cross-server communication.
func (s *HTTPServer) SetFederation(f *federation.FederationService) {
	s.federation = f
}

// SetReputation sets the reputation engine for recording and querying reputation.
func (s *HTTPServer) SetReputation(r *reputation.Engine) {
	s.reputation = r
}

// SetVerificationChallenger sets the endpoint verification challenger.
func (s *HTTPServer) SetVerificationChallenger(v *verification.Challenger) {
	s.verificationChallenger = v
}

// SetUserAuth sets the user authentication service.
func (s *HTTPServer) SetUserAuth(ua *userauth.Service) {
	s.userAuth = ua
}

// SetInvocation sets the invocation service.
func (s *HTTPServer) SetInvocation(inv *invocation.Service) {
	s.invocation = inv
}

// SetInvokeRateLimiter sets the rate limiter for agent invocations.
func (s *HTTPServer) SetInvokeRateLimiter(rl *IPRateLimiter) {
	s.invokeRateLimiter = rl
}

// SetReviewService sets the review service for ratings, reviews, and reports.
func (s *HTTPServer) SetReviewService(rs *review.Service) {
	s.reviewService = rs
}

func (s *HTTPServer) routes() {
	authMW := AuthMiddleware(s.authCfg, s.logger)
	ownerMW := OwnerOnlyMiddleware(s.logger)

	// wrapAuth applies authentication middleware to a handler.
	wrapAuth := func(h http.HandlerFunc) http.Handler {
		return authMW(h)
	}
	// wrapOwner applies authentication + owner-only middleware.
	wrapOwner := func(h http.HandlerFunc) http.Handler {
		return authMW(ownerMW(h))
	}

	// Public routes — no authentication required.
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/v1/directory", s.handleDirectory)
	s.mux.HandleFunc("GET /api/v1/directory/{id}", s.handlePublicProfile)
	s.mux.HandleFunc("GET /api/v1/directory/{id}/reputation", s.handleReputationHistory)

	// Authenticated routes.
	s.mux.Handle("POST /api/v1/agents", wrapAuth(s.handleRegister))
	s.mux.Handle("GET /api/v1/agents", wrapAuth(s.handleListAgents))
	s.mux.Handle("GET /api/v1/agents/{id}", wrapAuth(s.handleGetAgent))
	s.mux.Handle("POST /api/v1/discover", wrapAuth(s.handleDiscover))
	s.mux.Handle("GET /api/v1/routes", wrapAuth(s.handleGetRoutes))
	s.mux.Handle("GET /api/v1/routes/resolve", wrapAuth(s.handleResolveRoute))
	s.mux.Handle("POST /api/v1/bridge/send", wrapAuth(s.handleBridgeSend))
	if s.sigHub != nil {
		s.mux.Handle("GET /api/v1/signaling", wrapAuth(s.sigHub.HandleConnect))
	}

	// Owner-only routes — authenticated agent must match {id} in path.
	s.mux.Handle("DELETE /api/v1/agents/{id}", wrapOwner(s.handleDeregister))
	s.mux.Handle("POST /api/v1/agents/{id}/heartbeat", wrapOwner(s.handleHeartbeat))
	s.mux.Handle("POST /api/v1/agents/{id}/verify", wrapOwner(s.handleVerifyEndpoint))

	// Federation endpoints (use their own token-based auth).
	s.mux.HandleFunc("POST /api/v1/federation/signal", s.handleFederationSignal)
	s.mux.HandleFunc("POST /api/v1/federation/discover", s.handleFederationDiscover)

	// Protocol-specific inbound endpoints.
	s.registerProtocolRoutes()

	// Dashboard stats API (public).
	s.mux.HandleFunc("GET /api/v1/dashboard/stats", s.handleDashboardStats)

	// User auth routes.
	s.registerUserAuthRoutes()

	// Invoke routes.
	s.registerInvokeRoutes()

	// Provider console routes.
	s.registerProviderRoutes()

	// Review and community routes.
	s.registerReviewRoutes()

	// Admin routes.
	s.registerAdminRoutes()

	// Dashboard SPA (catch-all, registered last).
	s.mux.Handle("GET /", DashboardHandler())
}

func (s *HTTPServer) registerProtocolRoutes() {
	if s.bridges == nil {
		return
	}

	// A2A protocol endpoints.
	if s.bridges.HasBridge("a2a") {
		b, _ := s.bridges.GetBridge("a2a")
		if a2aAdapter, ok := b.(a2aAdapter); ok {
			s.mux.HandleFunc("POST /a2a", a2aAdapter.HandleMessages)
			s.mux.HandleFunc("GET /.well-known/agent.json", a2aAdapter.HandleAgentCard)
			s.mux.HandleFunc("GET /a2a/tasks/{id}", a2aAdapter.HandleGetTask)
		}
	}

	// MCP protocol endpoints.
	if s.bridges.HasBridge("mcp") {
		b, _ := s.bridges.GetBridge("mcp")
		if mcpAdapter, ok := b.(mcpAdapter); ok {
			s.mux.HandleFunc("POST /mcp", mcpAdapter.HandleMCP)
			s.mux.HandleFunc("GET /mcp", mcpAdapter.HandleMCPStream)
		}
	}

	// ACP protocol endpoints.
	if s.bridges.HasBridge("acp") {
		b, _ := s.bridges.GetBridge("acp")
		if acpAdapter, ok := b.(acpAdapter); ok {
			s.mux.HandleFunc("GET /acp/agents", acpAdapter.HandleListAgents)
			s.mux.HandleFunc("GET /acp/agents/{name}", acpAdapter.HandleGetAgent)
			s.mux.HandleFunc("POST /acp/runs", acpAdapter.HandleCreateRun)
			s.mux.HandleFunc("GET /acp/runs/{run_id}", acpAdapter.HandleGetRun)
			s.mux.HandleFunc("POST /acp/runs/{run_id}/cancel", acpAdapter.HandleCancelRun)
			s.mux.HandleFunc("GET /acp/ping", acpAdapter.HandlePing)
		}
	}
}

// Adapter interfaces for type assertions.
type a2aAdapter interface {
	HandleMessages(w http.ResponseWriter, r *http.Request)
	HandleAgentCard(w http.ResponseWriter, r *http.Request)
	HandleGetTask(w http.ResponseWriter, r *http.Request)
}

type mcpAdapter interface {
	HandleMCP(w http.ResponseWriter, r *http.Request)
	HandleMCPStream(w http.ResponseWriter, r *http.Request)
}

type acpAdapter interface {
	HandleListAgents(w http.ResponseWriter, r *http.Request)
	HandleGetAgent(w http.ResponseWriter, r *http.Request)
	HandleCreateRun(w http.ResponseWriter, r *http.Request)
	HandleGetRun(w http.ResponseWriter, r *http.Request)
	HandleCancelRun(w http.ResponseWriter, r *http.Request)
	HandlePing(w http.ResponseWriter, r *http.Request)
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
	PublicEndpoint  bool     `json:"public_endpoint"`
}

func (s *HTTPServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateRegisterRequest(&req); err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
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
			PublicEndpoint:  req.PeerClaw.PublicEndpoint,
		},
	})
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update routing table.
	s.engine.UpdateFromCard(card)

	// Record reputation event.
	if s.reputation != nil {
		_ = s.reputation.RecordEvent(r.Context(), card.ID, "registration", "")
	}

	// Audit log.
	if s.audit != nil {
		s.audit.LogRegistration(r.Context(), card.ID, card.Name, r.RemoteAddr)
	}
	if s.metrics != nil {
		s.metrics.RegisteredAgents.Add(r.Context(), 1)
	}

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

	// Audit log.
	if s.audit != nil {
		s.audit.LogDeregistration(r.Context(), id, r.RemoteAddr)
	}
	if s.metrics != nil {
		s.metrics.RegisteredAgents.Add(r.Context(), -1)
	}

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

	if err := validateHeartbeatStatus(req.Status); err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
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

	// Record reputation event.
	if s.reputation != nil {
		_ = s.reputation.RecordEvent(r.Context(), id, "heartbeat_success", "")
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
	resp := map[string]any{
		"status": "ok",
	}

	components := map[string]string{}

	// Database health.
	if s.store != nil {
		if _, err := s.store.List(r.Context(), registry.ListFilter{PageSize: 1}); err != nil {
			components["database"] = "error"
			resp["status"] = "degraded"
		} else {
			components["database"] = "ok"
		}
	}

	// Signaling health.
	if s.sigHub != nil {
		components["signaling"] = "ok"
		resp["connected_agents"] = s.sigHub.ConnectedAgents()
	}

	// Registered agents count.
	if s.store != nil {
		if result, err := s.store.List(r.Context(), registry.ListFilter{PageSize: 1}); err == nil {
			resp["registered_agents"] = result.TotalCount
		}
	}

	resp["components"] = components
	s.jsonResponse(w, http.StatusOK, resp)
}

func (s *HTTPServer) handleFederationSignal(w http.ResponseWriter, r *http.Request) {
	if s.federation == nil {
		s.jsonError(w, "federation not enabled", http.StatusNotFound)
		return
	}

	// Reject all federation requests when no auth token is configured.
	expectedToken := s.federation.AuthToken()
	if expectedToken == "" {
		s.jsonError(w, "federation auth token not configured", http.StatusForbidden)
		return
	}

	// Extract bearer token and use constant-time comparison.
	providedToken, err := identity.ExtractBearerToken(r.Header.Get("Authorization"))
	if err != nil {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) != 1 {
		s.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var msg coresignaling.SignalMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		s.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate that the signal message has a legitimate source.
	if msg.From == "" {
		s.jsonError(w, "missing 'from' field in signal message", http.StatusBadRequest)
		return
	}

	s.federation.HandleIncomingSignal(r.Context(), msg)
	w.WriteHeader(http.StatusOK)
}

func (s *HTTPServer) handleFederationDiscover(w http.ResponseWriter, r *http.Request) {
	if s.federation == nil {
		s.jsonError(w, "federation not enabled", http.StatusNotFound)
		return
	}

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

func (s *HTTPServer) registerReviewRoutes() {
	wrapReviewAuth := func(h http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.userAuth == nil {
				s.jsonError(w, "user authentication not enabled", http.StatusNotImplemented)
				return
			}
			UserAuthMiddleware(s.userAuth.JWTManager(), s.logger)(http.HandlerFunc(h)).ServeHTTP(w, r)
		})
	}

	// Public review endpoints.
	s.mux.HandleFunc("GET /api/v1/directory/{id}/reviews", s.handleListReviews)
	s.mux.HandleFunc("GET /api/v1/directory/{id}/reviews/summary", s.handleGetReviewSummary)

	// JWT-protected review endpoints.
	s.mux.Handle("POST /api/v1/directory/{id}/reviews", wrapReviewAuth(s.handleSubmitReview))
	s.mux.Handle("DELETE /api/v1/directory/{id}/reviews", wrapReviewAuth(s.handleDeleteReview))

	// Category endpoint (public).
	s.mux.HandleFunc("GET /api/v1/categories", s.handleListCategories)

	// Report endpoint (JWT-protected).
	s.mux.Handle("POST /api/v1/reports", wrapReviewAuth(s.handleSubmitReport))
}

func (s *HTTPServer) registerProviderRoutes() {
	wrapProviderAuth := func(h http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.userAuth == nil {
				s.jsonError(w, "user authentication not enabled", http.StatusNotImplemented)
				return
			}
			UserAuthMiddleware(s.userAuth.JWTManager(), s.logger)(http.HandlerFunc(h)).ServeHTTP(w, r)
		})
	}

	s.mux.Handle("POST /api/v1/provider/agents", wrapProviderAuth(s.handleProviderPublishAgent))
	s.mux.Handle("GET /api/v1/provider/agents", wrapProviderAuth(s.handleProviderListAgents))
	s.mux.Handle("GET /api/v1/provider/agents/{id}", wrapProviderAuth(s.handleProviderGetAgent))
	s.mux.Handle("PUT /api/v1/provider/agents/{id}", wrapProviderAuth(s.handleProviderUpdateAgent))
	s.mux.Handle("DELETE /api/v1/provider/agents/{id}", wrapProviderAuth(s.handleProviderDeleteAgent))
	s.mux.Handle("GET /api/v1/provider/agents/{id}/analytics", wrapProviderAuth(s.handleProviderAgentAnalytics))
	s.mux.Handle("GET /api/v1/provider/dashboard", wrapProviderAuth(s.handleProviderDashboard))
}

func (s *HTTPServer) registerInvokeRoutes() {
	// Invoke with optional auth (anonymous allowed with rate limiting).
	wrapOptionalAuth := func(h http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.userAuth != nil {
				OptionalUserAuthMiddleware(s.userAuth.JWTManager(), s.logger)(http.HandlerFunc(h)).ServeHTTP(w, r)
				return
			}
			h(w, r)
		})
	}

	wrapInvokeAuth := func(h http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.userAuth != nil {
				UserAuthMiddleware(s.userAuth.JWTManager(), s.logger)(http.HandlerFunc(h)).ServeHTTP(w, r)
				return
			}
			h(w, r)
		})
	}

	s.mux.Handle("POST /api/v1/invoke/{agent_id}", wrapOptionalAuth(s.handleInvoke))
	s.mux.Handle("GET /api/v1/invocations", wrapInvokeAuth(s.handleListInvocations))
	s.mux.Handle("GET /api/v1/invocations/{id}", wrapInvokeAuth(s.handleGetInvocation))
}

func (s *HTTPServer) registerUserAuthRoutes() {
	// Guard: if userAuth is not available, return 501 for auth endpoints.
	guardUserAuth := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if s.userAuth == nil {
				s.jsonError(w, "user authentication not enabled", http.StatusNotImplemented)
				return
			}
			h(w, r)
		}
	}

	// wrapUserAuth applies JWT auth middleware.
	wrapUserAuth := func(h http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.userAuth == nil {
				s.jsonError(w, "user authentication not enabled", http.StatusNotImplemented)
				return
			}
			UserAuthMiddleware(s.userAuth.JWTManager(), s.logger)(http.HandlerFunc(h)).ServeHTTP(w, r)
		})
	}

	// Public auth routes.
	s.mux.HandleFunc("POST /api/v1/auth/register", guardUserAuth(s.handleAuthRegister))
	s.mux.HandleFunc("POST /api/v1/auth/login", guardUserAuth(s.handleAuthLogin))
	s.mux.HandleFunc("POST /api/v1/auth/refresh", guardUserAuth(s.handleAuthRefresh))
	s.mux.HandleFunc("POST /api/v1/auth/logout", guardUserAuth(s.handleAuthLogout))

	// JWT-protected auth routes.
	s.mux.Handle("GET /api/v1/auth/me", wrapUserAuth(s.handleAuthMe))
	s.mux.Handle("PUT /api/v1/auth/me", wrapUserAuth(s.handleAuthUpdateMe))
	s.mux.Handle("POST /api/v1/auth/api-keys", wrapUserAuth(s.handleAuthCreateAPIKey))
	s.mux.Handle("GET /api/v1/auth/api-keys", wrapUserAuth(s.handleAuthListAPIKeys))
	s.mux.Handle("DELETE /api/v1/auth/api-keys/{key_id}", wrapUserAuth(s.handleAuthRevokeAPIKey))
}

// --- Helpers ---

func (s *HTTPServer) jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *HTTPServer) jsonError(w http.ResponseWriter, message string, status int) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}
