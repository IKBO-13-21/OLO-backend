package grpcapp

import (
	"auth/internal/grpc/authgrpc"
	authbuff "auth/pkg/note_v1"
	"context"
	"errors"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const (
	grpcAddress = "localhost:1999"
	httpAddress = "localhost:8080"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	httpServer *http.Server
	port       int
	httpPort   int
}

//goland:noinspection ALL
func New(log *slog.Logger, port int, httpPort int, authService authgrpc.Auth) *App {
	gRPCServer := grpc.NewServer()
	authgrpc.Register(gRPCServer, authService)

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
	}
	if err := authbuff.RegisterAuthHandlerFromEndpoint(context.Background(), mux, fmt.Sprintf("localhost:%d", port), opts); err != nil {
		panic(err)
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: mux,
	}

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		httpServer: httpServer,
		port:       port,
		httpPort:   httpPort,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "grpcapp.Run"
	log := a.log.With(
		slog.String("op", op),
		slog.Int("port", a.port),
		slog.Int("httpPort", a.httpPort))

	log.Info("starting gRPC server")

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	go func() {
		if err := a.gRPCServer.Serve(l); err != nil {
			log.Error("failed to serve gRPC")
		}
	}()

	log.Info("gRPC server is running", slog.String("addr", l.Addr().String()))

	log.Info("starting HTTP server for gRPC-Gateway")

	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed to serve HTTP")
		}
	}()

	log.Info("HTTP server for gRPC-Gateway is running", slog.String("addr", a.httpServer.Addr))

	return nil
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).Info("stopping gRPC server", slog.Int("port", a.port))

	a.gRPCServer.GracefulStop()

	a.log.With(slog.String("op", op)).Info("stopping HTTP server for gRPC-Gateway", slog.Int("httpPort", a.httpPort))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.log.Error("failed to shutdown HTTP server")
	}
}
func (a *App) StartHttpServer(ctx context.Context) error {
	mux := runtime.NewServeMux()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	err := authbuff.RegisterAuthHandlerFromEndpoint(ctx, mux, grpcAddress, opts)
	if err != nil {
		return err
	}

	log.Printf("http server listening at %v\n", httpAddress)

	return http.ListenAndServe(httpAddress, mux)
}
