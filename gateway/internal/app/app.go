package app

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Novip1906/tasks-grpc/gateway/internal/config"
	"github.com/Novip1906/tasks-grpc/gateway/internal/middleware"
	"github.com/Novip1906/tasks-grpc/gateway/pkg/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	authpb "github.com/Novip1906/tasks-grpc/gateway/internal/gen/auth"
	taskspb "github.com/Novip1906/tasks-grpc/gateway/internal/gen/tasks"
)

type Server struct {
	cfg        *config.Config
	log        *slog.Logger
	mux        *runtime.ServeMux
	cancelFunc context.CancelFunc
}

func NewServer(cfg *config.Config, log *slog.Logger) *Server {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

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
		panic(err)
	}

	if err := taskspb.RegisterTasksServiceHandlerFromEndpoint(
		ctx, mux, cfg.TasksAddress, opts,
	); err != nil {
		log.Error("failed to register tasks", logging.Err(err))
		panic(err)
	}
	return &Server{log: log, cfg: cfg, mux: mux, cancelFunc: cancel}
}

func (s *Server) Run() error {
	defer s.cancelFunc()

	handler := http.Handler(s.mux)
	handler = middleware.LoggingMiddleware(s.log)(handler)

	s.log.Info("starting server")
	return http.ListenAndServe(s.cfg.Address, handler)
}
