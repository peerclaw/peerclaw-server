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
	"github.com/peerclaw/peerclaw-server/internal/config"
	"github.com/peerclaw/peerclaw-server/internal/federation"
	"github.com/peerclaw/peerclaw-server/internal/observability"
	"github.com/peerclaw/peerclaw-server/internal/registry"
	"github.com/peerclaw/peerclaw-server/internal/reputation"
	"github.com/peerclaw/peerclaw-server/internal/router"
	"github.com/peerclaw/peerclaw-server/internal/server"
	"github.com/peerclaw/peerclaw-server/internal/signaling"
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
	defer func() { _ = store.Close() }()

	// Extract the underlying *sql.DB for shared use by reputation and verification modules.
	sqlDB, _ := store.GetDB().(*sql.DB)

	// Initialize reputation engine.
	var repEngine *reputation.Engine
	if sqlDB != nil {
		repStore := reputation.NewSQLiteStore(sqlDB)
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
		verifyStore := verification.NewSQLiteStore(sqlDB)
		if err := verifyStore.Migrate(context.Background()); err != nil {
			logger.Error("failed to migrate verification tables", "error", err)
			os.Exit(1)
		}
		verifyChallenger = verification.NewChallenger(verifyStore, logger)
		logger.Info("endpoint verification initialized")
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
	if sigHub != nil {
		sigHub.SetAudit(auditLogger)
		sigHub.SetMetrics(otelMetrics)

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
	// Start heartbeat timeout checker goroutine.
	if repEngine != nil && sqlDB != nil {
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Find agents whose last heartbeat is older than 5 minutes and status is online.
					rows, err := sqlDB.QueryContext(ctx,
						`SELECT id FROM agents WHERE status = 'online' AND last_heartbeat < ?`,
						time.Now().UTC().Add(-5*time.Minute).Format(time.RFC3339),
					)
					if err != nil {
						logger.Warn("heartbeat check query failed", "error", err)
						continue
					}
					for rows.Next() {
						var agentID string
						if err := rows.Scan(&agentID); err != nil {
							continue
						}
						_ = repEngine.RecordEvent(ctx, agentID, "heartbeat_miss", "")
						logger.Debug("heartbeat miss recorded", "agent_id", agentID)
					}
					_ = rows.Close()
				}
			}
		}()
		logger.Info("heartbeat timeout checker started", "interval", "60s", "timeout", "5m")
	}

	grpcServer := server.NewGRPCServer(cfg.Server.GRPCAddr, logger)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- httpServer.Start()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- grpcServer.Start()
	}()

	logger.Info("PeerClaw gateway started",
		"http", cfg.Server.HTTPAddr,
		"grpc", cfg.Server.GRPCAddr,
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
	grpcServer.Stop()
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
