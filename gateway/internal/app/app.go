package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Novip1906/tasks-grpc/gateway/internal/config"
	"github.com/Novip1906/tasks-grpc/gateway/internal/middleware"
	"github.com/Novip1906/tasks-grpc/gateway/pkg/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	httpSwagger "github.com/swaggo/http-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	authpb "github.com/Novip1906/tasks-grpc/gateway/internal/gen/auth"
	taskspb "github.com/Novip1906/tasks-grpc/gateway/internal/gen/tasks"
)

type Server struct {
	cfg    *config.Config
	log    *slog.Logger
	server *http.Server
}

func NewServer(cfg *config.Config, log *slog.Logger) (*Server, error) {
	ctx := context.Background()

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			switch strings.ToLower(key) {
			case "authorization", "content-type", "user-agent", "grpc-timeout":
				return key, true
			case "x-request-id", "x-correlation-id":
				return key, true
			}
			return runtime.DefaultHeaderMatcher(key)
		}),
		runtime.WithErrorHandler(func(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
			log.Error("grpc-gateway error", logging.Err(err))
			runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
		}),
	)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if err := authpb.RegisterAuthServiceHandlerFromEndpoint(
		ctx, mux, cfg.AuthAddress, opts,
	); err != nil {
		log.Error("failed to register auth", logging.Err(err))
		return nil, err
	}

	if err := taskspb.RegisterTasksServiceHandlerFromEndpoint(
		ctx, mux, cfg.TasksAddress, opts,
	); err != nil {
		log.Error("failed to register tasks", logging.Err(err))
		return nil, err
	}

	rateLimiter := middleware.NewRateLimiter(ctx, log, &cfg.Redis, &cfg.RateLimiter)

	rootMux := http.NewServeMux()

	rootMux.Handle("/", mux)

	rootMux.Handle("/swagger.yaml", http.FileServer(http.Dir("./swagger")))

	rootMux.Handle("/docs/", httpSwagger.Handler(
		httpSwagger.URL("/swagger.yaml"),
	))

	handler := rateLimiter.Middleware(log)(rootMux)
	handler = middleware.LoggingMiddleware(log)(handler)

	httpServer := &http.Server{
		Addr:         cfg.Address,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		cfg:    cfg,
		log:    log,
		server: httpServer,
	}, nil
}

func (s *Server) Run() error {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		s.log.Info("starting server", slog.String("address", s.cfg.Address))
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return err
	case <-stopChan:
		s.log.Info("received shutdown signal")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			s.log.Error("server shutdown failed", logging.Err(err))
			s.server.Close()
			return err
		}

		s.log.Info("server stopped gracefully")
		return nil
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
