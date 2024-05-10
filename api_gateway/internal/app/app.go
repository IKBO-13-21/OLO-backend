// Package app provides the entry point for the OLO-backend API Gateway application.
//
// The API gateway is responsible for routing incoming requests to the appropriate services
// and handling cross-cutting concerns such as authentication, rate limiting, and logging.
package app

import (
	"OLO-backend/api_gateway/internal/config"
	"OLO-backend/api_gateway/internal/entity"
	pauth "OLO-backend/auth_service/generated"
	polo "OLO-backend/olo_service/generated"
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/negroni"
	"log/slog"
	"strconv"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
)

type App struct {
	config *config.Config // It holds various configuration parameters required by the application.
	log    *slog.Logger   // It provides logging capabilities for the application.
}

var (
	// метрика для отслеживания затраченного времени и сделанных запросов
	requestMetrics = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "ads",
			Subsystem:  "http",
			Name:       "request",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status"},
	)

	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ads",
			Subsystem: "http",
			Name:      "client_request_count",
			Help:      "Total number of requests from client",
		},
		[]string{"client", "server", "method", "route", "status"},
	)

	durationHistorgram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ads",
			Subsystem: "http",
			Name:      "client_request_duration_secs",
			Help:      "Duration of requests from Client",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5},
		},
		[]string{"client", "server", "method", "route", "status"},
	)
)

// New is the entry point for the API Gateway application.
// It initializes the application
func New(log *slog.Logger) (app *App, err error) {
	app = &App{
		config: config.MustLoad(),
		log:    log,
	}
	return
}

// The initService function initializes a service using the provided socket information.
// It executes a callback function with the formatted address of the service.
func (app *App) initService(s entity.Socket, fn func(formattedAddr string)) {
	fn(fmt.Sprintf("%s:%d", s.Host, s.Port))
}

func wrapHandlerForStatPrometheus(wrappedHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		//log.Printf("--> %s %s", req.Method, req.URL.Path)
		lrw := negroni.NewResponseWriter(w)
		start := time.Now()
		defer func() {
			statusCode := lrw.Status()

			labels := prometheus.Labels{
				"client": req.RemoteAddr,           // defines the client server
				"server": req.Host,                 // defines the outbound request server
				"method": req.Method,               // HTTP method
				"route":  req.URL.Path,             // Request route
				"status": strconv.Itoa(statusCode), // Response status
			}
			duration := time.Since(start).Seconds()

			requestMetrics.WithLabelValues(strconv.Itoa(statusCode)).Observe(duration)

			// the duration
			durationHistorgram.With(labels).Observe(duration)

			// request api count
			requestCounter.With(labels).Inc()

			//log.Printf("<-- %d %s", statusCode, http.StatusText(statusCode))
		}()
		wrappedHandler.ServeHTTP(lrw, req)
	})
}

// The Start function initializes and starts the API gateway service.
// It sets up HTTP server configurations, registers gRPC services, and starts listening for incoming requests.
func (app *App) Start() {
	const op = "httpSrv.Start"

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	prometheus.MustRegister(requestCounter)
	prometheus.MustRegister(durationHistorgram)

	logger := app.log.With(
		slog.String("op", op),
		slog.Int("http_port", app.config.HTTP.Port),
	)

	gatewayMux := runtime.NewServeMux()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", wrapHandlerForStatPrometheus(gatewayMux))

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	app.initService(entity.Socket{
		Host: app.config.AuthService.Host,
		Port: app.config.AuthService.Port,
	}, func(formattedAddr string) {
		err := pauth.RegisterAuthHandlerFromEndpoint(ctx, gatewayMux, formattedAddr, opts)
		if err != nil {
			logger.Error("can`t register service: %v", err)
		}
	})

	app.initService(entity.Socket{
		Host: app.config.OloService.Host,
		Port: app.config.OloService.Port,
	}, func(formattedAddr string) {
		err := polo.RegisterOLOHandlerFromEndpoint(ctx, gatewayMux, formattedAddr, opts)
		if err != nil {
			logger.Error("can`t to register service: %v", err)
		}
	})

	withCors := cors.New(cors.Options{
		AllowCredentials: true,
		MaxAge:           300,
	}).Handler(mux)

	httpAddr := app.config.HTTP.ToStr()
	logger.Info("API Gateway is listening", slog.String("addr", httpAddr))
	if err := http.ListenAndServe(httpAddr, withCors); err != nil {
		logger.Error("failed to serve: %v", err)
	}
}
