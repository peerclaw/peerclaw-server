package server

import (
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCServer wraps a gRPC server with PeerClaw services.
type GRPCServer struct {
	server *grpc.Server
	logger *slog.Logger
	addr   string
}

// NewGRPCServer creates a new gRPC server.
// Set enableReflection to true only for debugging/development; disable in production.
func NewGRPCServer(addr string, logger *slog.Logger, enableReflection bool) *GRPCServer {
	if logger == nil {
		logger = slog.Default()
	}
	s := grpc.NewServer()
	if enableReflection {
		reflection.Register(s)
	}
	return &GRPCServer{
		server: s,
		logger: logger,
		addr:   addr,
	}
}

// Server returns the underlying grpc.Server for service registration.
func (s *GRPCServer) Server() *grpc.Server {
	return s.server
}

// Start begins listening and serving gRPC requests.
func (s *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.logger.Info("gRPC server listening", "addr", s.addr)
	return s.server.Serve(lis)
}

// Stop gracefully stops the gRPC server.
func (s *GRPCServer) Stop() {
	s.server.GracefulStop()
}
