package app

import (
	"log/slog"
	"net"

	"github.com/Novip1906/tasks-grpc/auth/internal/config"
	"github.com/Novip1906/tasks-grpc/auth/internal/interceptors"
	"github.com/Novip1906/tasks-grpc/auth/internal/storage"
	"google.golang.org/grpc"

	pb "github.com/Novip1906/tasks-grpc/auth/api/proto/gen"
	"github.com/Novip1906/tasks-grpc/auth/internal/service"
)

type Server struct {
	cfg         *config.Config
	gs          *grpc.Server
	log         *slog.Logger
	authService *service.AuthService
}

func NewServer(cfg *config.Config, log *slog.Logger) *Server {
	gs := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.LoggingInterceptor(log)),
	)

	p := cfg.DB
	db, err := storage.NewPostgresStorage(p.Host, p.Port, p.User, p.Password, p.DBName, log)
	if err != nil {
		panic(err)
	}

	authService := service.NewAuthService(cfg, log, db)

	return &Server{cfg: cfg, gs: gs, authService: authService, log: log}
}

func (s *Server) Run() error {
	ln, err := net.Listen("tcp", s.cfg.Address)
	if err != nil {
		return err
	}

	pb.RegisterAuthServiceServer(s.gs, s.authService)

	return s.gs.Serve(ln)
}
