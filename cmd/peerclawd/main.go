package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"database/sql"

	"github.com/peerclaw/peerclaw-server/internal/audit"
	"github.com/peerclaw/peerclaw-server/internal/bridge"
	"github.com/peerclaw/peerclaw-server/internal/bridge/a2a"
	"github.com/peerclaw/peerclaw-server/internal/bridge/acp"
	"github.com/peerclaw/peerclaw-server/internal/bridge/mcp"
	"github.com/peerclaw/peerclaw-server/internal/claimtoken"
	"github.com/peerclaw/peerclaw-server/internal/config"
	"github.com/peerclaw/peerclaw-server/internal/contactreq"
	"github.com/peerclaw/peerclaw-server/internal/contacts"
	"github.com/peerclaw/peerclaw-server/internal/federation"
	"github.com/peerclaw/peerclaw-server/internal/notification"
	"github.com/peerclaw/peerclaw-server/internal/observability"
	"github.com/peerclaw/peerclaw-server/internal/registry"
	"github.com/peerclaw/peerclaw-server/internal/reputation"
	"github.com/peerclaw/peerclaw-server/internal/retention"
	"github.com/peerclaw/peerclaw-server/internal/router"
	"github.com/peerclaw/peerclaw-server/internal/server"
	"github.com/peerclaw/peerclaw-server/internal/signaling"
	"github.com/peerclaw/peerclaw-server/internal/invocation"
	"github.com/peerclaw/peerclaw-server/internal/review"
	"github.com/peerclaw/peerclaw-server/internal/useracl"
	"github.com/peerclaw/peerclaw-server/internal/userauth"
	"github.com/peerclaw/peerclaw-server/internal/versioncheck"
	"github.com/peerclaw/peerclaw-server/internal/verification"
	goredis "github.com/redis/go-redis/v9"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Set up logger.
	var handler slog.Handler
	logLevel := slog.LevelInfo
	switch cfg.Logging.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: logLevel}
	if cfg.Logging.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	logger := slog.New(handler)

	// Initialize OpenTelemetry.
	otelProvider, err := observability.Init(context.Background(), cfg.Observability, logger)
	if err != nil {
		logger.Error("failed to initialize OpenTelemetry", "error", err)
		os.Exit(1)
	}
	defer func() { _ = otelProvider.Shutdown(context.Background()) }()

	// Initialize metrics.
	otelMetrics, err := observability.NewMetrics(observability.Meter("peerclaw-gateway"))
	if err != nil {
		logger.Error("failed to create metrics", "error", err)
		os.Exit(1)
	}

	// Initialize store.
	store, err := registry.NewStore(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		logger.Error("failed to open database", "error", err, "driver", cfg.Database.Driver)
		os.Exit(1)
	}
	defer store.Close()

	// Extract the underlying *sql.DB for shared use by reputation and verification modules.
	sqlDB, _ := store.GetDB().(*sql.DB)

	// Initialize reputation engine.
	var repEngine *reputation.Engine
	var repStore reputation.Store
	if sqlDB != nil {
		repStore = reputation.NewStore(cfg.Database.Driver, sqlDB)
		if err := repStore.Migrate(context.Background()); err != nil {
			logger.Error("failed to migrate reputation tables", "error", err)
			os.Exit(1)
		}
		repEngine = reputation.NewEngine(repStore, logger)
		logger.Info("reputation engine initialized")
	}

	// Initialize verification challenger.
	var verifyChallenger *verification.Challenger
	if sqlDB != nil {
		verifyStore := verification.NewStore(cfg.Database.Driver, sqlDB)
		if err := verifyStore.Migrate(context.Background()); err != nil {
			logger.Error("failed to migrate verification tables", "error", err)
			os.Exit(1)
		}
		verifyChallenger = verification.NewChallenger(verifyStore, logger)
		logger.Info("endpoint verification initialized")
	}

	// Initialize user authentication.
	var userAuthService *userauth.Service
	if cfg.UserAuth.Enabled && sqlDB != nil {
		uaStore := userauth.NewStore(cfg.Database.Driver, sqlDB)
		if err := uaStore.Migrate(context.Background()); err != nil {
			logger.Error("failed to migrate userauth tables", "error", err)
			os.Exit(1)
		}

		jwtSecret := cfg.UserAuth.JWTSecret
		if jwtSecret == "" {
			jwtSecret = "peerclaw-dev-secret-change-me"
			logger.Warn("using default JWT secret — set user_auth.jwt_secret in config for production")
		}

		accessTTL, err := time.ParseDuration(cfg.UserAuth.AccessTTL)
		if err != nil {
			accessTTL = 15 * time.Minute
		}
		refreshTTL, err := time.ParseDuration(cfg.UserAuth.RefreshTTL)
		if err != nil {
			refreshTTL = 168 * time.Hour
		}

		jwtMgr := userauth.NewJWTManager(jwtSecret, accessTTL, refreshTTL)
		emailSender := userauth.NewEmailSender(cfg.SMTP, logger)
		userAuthService = userauth.NewService(uaStore, jwtMgr, cfg.UserAuth.BcryptCost, cfg.UserAuth.AdminEmails, emailSender, logger)
		logger.Info("user authentication initialized",
			"access_ttl", accessTTL,
			"refresh_ttl", refreshTTL,
			"smtp_configured", cfg.SMTP.Host != "",
		)
	}

	// Initialize invocation store.
	var invocationService *invocation.Service
	var invStore invocation.Store
	if sqlDB != nil {
		invStore = invocation.NewStore(cfg.Database.Driver, sqlDB)
		if err := invStore.Migrate(context.Background()); err != nil {
			logger.Error("failed to migrate invocation tables", "error", err)
			os.Exit(1)
		}
		invocationService = invocation.NewService(invStore, logger)
		logger.Info("invocation tracking initialized")
	}

	// Initialize review service.
	var reviewService *review.Service
	var revStore review.Store
	if sqlDB != nil {
		revStore = review.NewStore(cfg.Database.Driver, sqlDB)
		if err := revStore.Migrate(context.Background()); err != nil {
			logger.Error("failed to migrate review tables", "error", err)
			os.Exit(1)
		}
		reviewService = review.NewService(revStore, repEngine, logger)
		logger.Info("review service initialized")
	}

	// Initialize claim token service.
	var claimTokenService *claimtoken.Service
	if sqlDB != nil {
		ctStore := claimtoken.NewStore(cfg.Database.Driver, sqlDB)
		if err := ctStore.Migrate(context.Background()); err != nil {
			logger.Error("failed to migrate claim token tables", "error", err)
			os.Exit(1)
		}
		claimTokenService = claimtoken.NewService(ctStore, logger)
		logger.Info("claim token service initialized")
	}

	// Initialize contacts service.
	var contactsService *contacts.Service
	if sqlDB != nil {
		ctcStore := contacts.NewStore(cfg.Database.Driver, sqlDB)
		if err := ctcStore.Migrate(context.Background()); err != nil {
			logger.Error("failed to migrate contacts tables", "error", err)
			os.Exit(1)
		}
		contactsService = contacts.NewService(ctcStore, logger)
		logger.Info("contacts service initialized")
	}

	// Initialize contact request service.
	var contactReqService *contactreq.Service
	if sqlDB != nil {
		crStore := contactreq.NewStore(cfg.Database.Driver, sqlDB)
		if err := crStore.Migrate(context.Background()); err != nil {
			logger.Error("failed to migrate contact request tables", "error", err)
			os.Exit(1)
		}
		contactReqService = contactreq.NewService(crStore, contactsService, logger)
		logger.Info("contact request service initialized")
	}

	// Initialize user ACL service.
	var userACLService *useracl.Service
	if sqlDB != nil {
		uaclStore := useracl.NewStore(cfg.Database.Driver, sqlDB)
		if err := uaclStore.Migrate(context.Background()); err != nil {
			logger.Error("failed to migrate user ACL tables", "error", err)
			os.Exit(1)
		}
		userACLService = useracl.NewService(uaclStore, logger)
		logger.Info("user ACL service initialized")
	}

	// Initialize notification service.
	var notificationSvc *notification.Service
	var notifHub *notification.DashboardHub
	if cfg.Notification.Enabled && sqlDB != nil {
		notifStore := notification.NewStore(cfg.Database.Driver, sqlDB)
		if err := notifStore.Migrate(context.Background()); err != nil {
			logger.Error("failed to migrate notification tables", "error", err)
			os.Exit(1)
		}
		notificationSvc = notification.NewService(notifStore, logger)
		notifHub = notification.NewDashboardHub(logger)

		// Set emitter: WebSocket push + optional email for critical events.
		emailSender := userauth.NewEmailSender(cfg.SMTP, logger)
		notificationSvc.SetEmitter(func(n *notification.Notification) {
			// Push via WebSocket.
			notifHub.Push(n)

			// Send email for critical events if enabled.
			if cfg.Notification.EmailEnabled && n.Severity == notification.SeverityCritical {
				if cfg.Notification.EmailOnOffline || n.Type != notification.TypeAgentOffline {
					// Look up user email for notification.
					if userAuthService != nil {
						if user, err := userAuthService.GetUser(context.Background(), n.UserID); err == nil && user != nil {
							_ = emailSender.SendNotification(user.Email, n.Title, n.Body)
						}
					}
				}
			}
		})
		logger.Info("notification service initialized",
			"retention_days", cfg.Notification.RetentionDays,
			"email_enabled", cfg.Notification.EmailEnabled,
		)
	}

	// Initialize services.
	regService := registry.NewService(store, logger)
	routeTable := router.NewTable()
	routeEngine := router.NewEngine(routeTable, logger)
	bridgeManager := bridge.NewManager(logger)

	// Register protocol bridges.
	if cfg.Bridge.A2A.Enabled {
		_ = bridgeManager.RegisterBridge(a2a.New(logger, nil))
	}
	if cfg.Bridge.ACP.Enabled {
		_ = bridgeManager.RegisterBridge(acp.New(logger, nil))
	}
	if cfg.Bridge.MCP.Enabled {
		_ = bridgeManager.RegisterBridge(mcp.New(logger, nil))
	}

	// Initialize signaling hub.
	var sigHub *signaling.Hub
	if cfg.Signaling.Enabled {
		var turnCfg *signaling.TURNConfig
		if len(cfg.Signaling.TURN.URLs) > 0 {
			turnCfg = &signaling.TURNConfig{
				URLs:       cfg.Signaling.TURN.URLs,
				Username:   cfg.Signaling.TURN.Username,
				Credential: cfg.Signaling.TURN.Credential,
			}
			logger.Info("TURN servers configured", "urls", cfg.Signaling.TURN.URLs)
		}
		sigHub = signaling.NewHub(logger, turnCfg, cfg.RateLimit.MaxConnections)
		if len(cfg.Signaling.AllowedOrigins) > 0 {
			sigHub.SetAllowedOrigins(cfg.Signaling.AllowedOrigins)
		}
		logger.Info("WebSocket signaling enabled",
			"max_connections", cfg.RateLimit.MaxConnections,
		)
	}

	// Initialize federation.
	var fedService *federation.FederationService
	if cfg.Federation.Enabled {
		fedService = federation.New(cfg.Federation.NodeName, cfg.Federation.AuthToken, logger)
		for _, p := range cfg.Federation.Peers {
			fedService.AddPeer(p.Name, p.Address, p.Token)
		}

		// DNS-based peer discovery.
		if cfg.Federation.DNSEnabled && cfg.Federation.DNSDomain != "" {
			dnsPeers, err := federation.DiscoverPeers(cfg.Federation.DNSDomain)
			if err != nil {
				logger.Warn("federation DNS discovery failed", "error", err)
			} else {
				for _, p := range dnsPeers {
					fedService.AddPeer(p.Name, p.Address, p.Token)
				}
				logger.Info("federation DNS peers discovered", "count", len(dnsPeers))
			}
		}

		logger.Info("federation enabled", "node_name", cfg.Federation.NodeName, "peers", len(cfg.Federation.Peers))
	}

	// Initialize audit logger.
	auditLogger, err := audit.NewFromConfig(cfg.AuditLog)
	if err != nil {
		logger.Error("failed to create audit logger", "error", err)
		os.Exit(1)
	}

	// Configure HTTP server options.
	httpOpts := &server.HTTPServerConfig{
		Metrics:     otelMetrics,
		Tracer:      observability.Tracer("peerclaw-http"),
		CORSOrigins: cfg.Server.CORSOrigins,
	}
	if cfg.RateLimit.Enabled {
		rl := server.NewIPRateLimiter(cfg.RateLimit.RequestsPerSec, cfg.RateLimit.BurstSize)
		stopCleanup := rl.StartCleanup(time.Minute)
		defer stopCleanup()
		httpOpts.RateLimiter = rl
		httpOpts.MaxBodyBytes = int64(cfg.RateLimit.MaxMessageBytes)
		logger.Info("rate limiting enabled",
			"requests_per_sec", cfg.RateLimit.RequestsPerSec,
			"burst", cfg.RateLimit.BurstSize,
		)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start servers.
	httpServer := server.NewHTTPServer(cfg.Server.HTTPAddr, regService, routeEngine, bridgeManager, sigHub, logger, httpOpts)
	httpServer.SetStore(store)
	httpServer.SetAudit(auditLogger)
	httpServer.SetMetrics(otelMetrics)
	if repEngine != nil {
		httpServer.SetReputation(repEngine)
	}
	if verifyChallenger != nil {
		httpServer.SetVerificationChallenger(verifyChallenger)
	}
	if fedService != nil {
		httpServer.SetFederation(fedService)
	}
	if userAuthService != nil {
		httpServer.SetUserAuth(userAuthService)
	}
	if reviewService != nil {
		httpServer.SetReviewService(reviewService)
	}
	if invocationService != nil {
		httpServer.SetInvocation(invocationService)
		// Anonymous: 10 calls/hour/IP, Authenticated: 100 calls/hour.
		invokeRL := server.NewIPRateLimiter(10.0/3600.0, 3)
		stopInvokeCleanup := invokeRL.StartCleanup(time.Minute)
		defer stopInvokeCleanup()
		httpServer.SetInvokeRateLimiter(invokeRL)
	}
	if claimTokenService != nil {
		httpServer.SetClaimToken(claimTokenService)
	}
	if contactsService != nil {
		httpServer.SetContacts(contactsService)
		// Per-source-agent rate limiter for bridge sends: 1 msg/sec, burst 10.
		bridgeRL := server.NewIPRateLimiter(1.0, 10)
		stopBridgeCleanup := bridgeRL.StartCleanup(time.Minute)
		defer stopBridgeCleanup()
		httpServer.SetBridgeRateLimiter(bridgeRL)
		logger.Info("bridge per-agent rate limiter enabled", "rate", 1.0, "burst", 10)
	}
	if contactReqService != nil {
		httpServer.SetContactRequests(contactReqService)
	}
	if userACLService != nil {
		httpServer.SetUserACL(userACLService)
	}
	if notificationSvc != nil {
		httpServer.SetNotification(notificationSvc)
		httpServer.SetNotificationHub(notifHub)
	}
	if sigHub != nil {
		sigHub.SetAudit(auditLogger)
		sigHub.SetMetrics(otelMetrics)
		if contactsService != nil {
			sigHub.SetContacts(contactsService)
			logger.Info("signaling hub contacts whitelist enabled")
		}

		// Configure signaling broker.
		if cfg.Redis.Addr != "" {
			redisClient := goredis.NewClient(&goredis.Options{
				Addr:     cfg.Redis.Addr,
				Password: cfg.Redis.Password,
				DB:       cfg.Redis.DB,
			})
			// Test connection; fall back to local if Redis is unavailable.
			if err := redisClient.Ping(ctx).Err(); err != nil {
				logger.Warn("Redis unavailable, using local signaling broker", "error", err)
				sigHub.SetBroker(signaling.NewLocalBroker(sigHub))
			} else {
				broker := signaling.NewRedisBroker(redisClient, sigHub, logger)
				if _, err := broker.Subscribe(ctx); err != nil {
					logger.Error("Redis subscribe failed", "error", err)
					_ = redisClient.Close()
					sigHub.SetBroker(signaling.NewLocalBroker(sigHub))
				} else {
					sigHub.SetBroker(broker)
					logger.Info("Redis signaling broker enabled", "addr", cfg.Redis.Addr)
				}
			}
		} else {
			sigHub.SetBroker(signaling.NewLocalBroker(sigHub))
		}
	}
	// Initialize version check service.
	if cfg.VersionCheck.Enabled {
		vcInterval, err := time.ParseDuration(cfg.VersionCheck.Interval)
		if err != nil {
			vcInterval = time.Hour
		}
		vcService := versioncheck.New(cfg.VersionCheck.Repo, vcInterval, logger)
		httpServer.SetVersionCheck(vcService)
		go vcService.Start(ctx)
		logger.Info("version check service started", "repo", cfg.VersionCheck.Repo, "interval", vcInterval)
	}

	// Register routes after all services are configured so that
	// optional-service routes (claim tokens, contacts, etc.) are included.
	httpServer.RegisterRoutes()

	// Start heartbeat timeout checker goroutine.
	if repEngine != nil && repStore != nil {
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					staleAgents, err := repStore.ListStaleOnlineAgents(ctx, 5*time.Minute)
					if err != nil {
						logger.Warn("heartbeat check query failed", "error", err)
						continue
					}
					for _, agentID := range staleAgents {
						_ = repEngine.RecordEvent(ctx, agentID, "heartbeat_miss", "")
						logger.Debug("heartbeat miss recorded", "agent_id", agentID)

						// Emit agent offline notification to owner.
						if notificationSvc != nil {
							card, err := regService.GetAgent(ctx, agentID)
							if err == nil && card.Metadata != nil {
								if ownerUserID, ok := card.Metadata["owner_user_id"]; ok && ownerUserID != "" {
									_, _ = notificationSvc.Notify(ctx, ownerUserID, agentID,
										notification.TypeAgentOffline, notification.SeverityCritical,
										"Agent offline",
										"Your agent has stopped sending heartbeats",
										map[string]string{"agent_id": agentID},
									)
								}
							}
						}
					}
				}
			}
		}()
		logger.Info("heartbeat timeout checker started", "interval", "60s", "timeout", "5m")
	}

	// Start expired email verification cleanup goroutine.
	if userAuthService != nil {
		go func() {
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := userAuthService.DeleteExpiredVerifications(ctx); err != nil {
						logger.Warn("email verification cleanup failed", "error", err)
					}
				}
			}
		}()
		logger.Info("email verification cleanup goroutine started", "interval", "1h")
	}

	// Start data retention cleanup goroutine.
	if cfg.Retention.Enabled {
		retentionInterval, err := time.ParseDuration(cfg.Retention.CleanupInterval)
		if err != nil {
			retentionInterval = 1 * time.Hour
		}
		retentionSvc := retention.NewService(repStore, invStore, revStore, retention.Config{
			ReputationEventsDays: cfg.Retention.ReputationEventsDays,
			InvocationsDays:      cfg.Retention.InvocationsDays,
			AbuseReportsDays:     cfg.Retention.AbuseReportsDays,
		}, logger)
		go func() {
			ticker := time.NewTicker(retentionInterval)
			defer ticker.Stop()
			// Run once on startup.
			retentionSvc.RunOnce(ctx)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					retentionSvc.RunOnce(ctx)
				}
			}
		}()
		logger.Info("data retention cleanup started",
			"interval", retentionInterval,
			"reputation_events_days", cfg.Retention.ReputationEventsDays,
			"invocations_days", cfg.Retention.InvocationsDays,
			"abuse_reports_days", cfg.Retention.AbuseReportsDays,
		)
	}

	// Start notification retention cleanup goroutine.
	if notificationSvc != nil && cfg.Notification.RetentionDays > 0 {
		go func() {
			ticker := time.NewTicker(6 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					cutoff := time.Now().AddDate(0, 0, -cfg.Notification.RetentionDays)
					deleted, err := notificationSvc.Prune(ctx, cutoff)
					if err != nil {
						logger.Warn("notification retention cleanup failed", "error", err)
					} else if deleted > 0 {
						logger.Info("notification retention cleanup", "deleted", deleted)
					}
				}
			}
		}()
		logger.Info("notification retention cleanup started",
			"interval", "6h",
			"retention_days", cfg.Notification.RetentionDays,
		)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- httpServer.Start()
	}()

	logger.Info("PeerClaw gateway started",
		"http", cfg.Server.HTTPAddr,
	)

	// Wait for shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Info("shutting down", "signal", sig)
	case err := <-errCh:
		logger.Error("server error", "error", err)
	case <-ctx.Done():
	}

	// Graceful shutdown: cancel context to signal all goroutines, then
	// stop the servers and wait for goroutines to finish.
	cancel()
	_ = httpServer.Stop()

	// Wait for server goroutines to complete before tearing down
	// remaining services, ensuring a clean shutdown.
	wg.Wait()

	if fedService != nil {
		_ = fedService.Close()
	}
	_ = bridgeManager.Close()
	logger.Info("PeerClaw gateway stopped")
}
