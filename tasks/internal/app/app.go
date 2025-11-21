package app

import (
	"log/slog"
	"net"
	"time"

	"github.com/Novip1906/tasks-grpc/tasks/internal/config"
	"github.com/Novip1906/tasks-grpc/tasks/internal/interceptors"
	"github.com/Novip1906/tasks-grpc/tasks/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	tasksPb "github.com/Novip1906/tasks-grpc/tasks/api/proto/gen"
	authPb "github.com/Novip1906/tasks-grpc/tasks/internal/auth_gen"
	"github.com/Novip1906/tasks-grpc/tasks/internal/service"
)

type Server struct {
	cfg          *config.Config
	gs           *grpc.Server
	log          *slog.Logger
	tasksService *service.TasksService
}

var authTimeout = 3 * time.Second

func NewServer(cfg *config.Config, log *slog.Logger) *Server {
	authConn, err := grpc.NewClient(cfg.AuthAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	authClient := authPb.NewAuthServiceClient(authConn)
	authInterceptor := interceptors.AuthUnaryInterceptor(authClient, authTimeout, log)
	loggingInterceptor := interceptors.LoggingInterceptor(log)

	gs := grpc.NewServer(grpc.ChainUnaryInterceptor(loggingInterceptor, authInterceptor))

	p := cfg.DB
	db, err := storage.NewPostgresStorage(p.Host, p.Port, p.User, p.Password, p.DBName, log)
	if err != nil {
		panic(err)
	}

	taskService := service.NewTasksService(cfg, log, db)

	return &Server{cfg: cfg, gs: gs, tasksService: taskService, log: log}
}

func (s *Server) Run() error {
	ln, err := net.Listen("tcp", s.cfg.TasksAddress)
	if err != nil {
		return err
	}

	tasksPb.RegisterTasksServiceServer(s.gs, s.tasksService)

	return s.gs.Serve(ln)
}
