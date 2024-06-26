// Package grpc provides functionality for gRPC server initialization.
package grpc

import (
	"OLO-backend/auth_service/internal/grpc/authgrpc"
	"fmt"
	"google.golang.org/grpc"
	"log/slog"
	"net"
)

// Grpc represents the gRPC server.
type Grpc struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

// New creates a new instance of the gRPC server.
func New(log *slog.Logger, port int, authService authgrpc.Auth) *Grpc {
	gRPCServer := grpc.NewServer()
	authgrpc.Register(gRPCServer, authService)

	return &Grpc{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
	}
}

// MustRun runs the gRPC server and panics if there's an error.
func (a *Grpc) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

// Run starts the gRPC server.
func (a *Grpc) Run() error {
	const op = "grpcapp.Run"
	log := a.log.With(
		slog.String("op", op),
		slog.Int("grpc_port", a.port),
	)

	log.Info("starting gRPC server")

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("failed to listen %s: %w", op, err)
	}

	log.Info("gRPC server is running", slog.String("addr", l.Addr().String()))
	if err := a.gRPCServer.Serve(l); err != nil {
		log.Error("failed to serve gRPC")
	}
	return nil
}

// Stop stops the gRPC server.
func (a *Grpc) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).Info("stopping gRPC server", slog.Int("port", a.port))
	a.gRPCServer.GracefulStop()
}
