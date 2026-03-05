package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/peerclaw/peerclaw-server/internal/bridge"
	"github.com/peerclaw/peerclaw-server/internal/bridge/a2a"
	"github.com/peerclaw/peerclaw-server/internal/bridge/acp"
	"github.com/peerclaw/peerclaw-server/internal/bridge/mcp"
	"github.com/peerclaw/peerclaw-server/internal/config"
	"github.com/peerclaw/peerclaw-server/internal/registry"
	"github.com/peerclaw/peerclaw-server/internal/router"
	"github.com/peerclaw/peerclaw-server/internal/server"
	"github.com/peerclaw/peerclaw-server/internal/signaling"
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

	// Initialize store.
	store, err := registry.NewSQLiteStore(cfg.Database.DSN)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Initialize services.
	regService := registry.NewService(store, logger)
	routeTable := router.NewTable()
	routeEngine := router.NewEngine(routeTable, logger)
	bridgeManager := bridge.NewManager(logger)

	// Register protocol bridges.
	if cfg.Bridge.A2A.Enabled {
		bridgeManager.RegisterBridge(a2a.New(logger, nil))
	}
	if cfg.Bridge.ACP.Enabled {
		bridgeManager.RegisterBridge(acp.New(logger, nil))
	}
	if cfg.Bridge.MCP.Enabled {
		bridgeManager.RegisterBridge(mcp.New(logger, nil))
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
		sigHub = signaling.NewHub(logger, turnCfg)
		logger.Info("WebSocket signaling enabled")
	}

	// Start servers.
	httpServer := server.NewHTTPServer(cfg.Server.HTTPAddr, regService, routeEngine, bridgeManager, sigHub, logger)
	grpcServer := server.NewGRPCServer(cfg.Server.GRPCAddr, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 2)
	go func() { errCh <- httpServer.Start() }()
	go func() { errCh <- grpcServer.Start() }()

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

	// Graceful shutdown.
	grpcServer.Stop()
	httpServer.Stop()
	bridgeManager.Close()
	logger.Info("PeerClaw gateway stopped")
}
