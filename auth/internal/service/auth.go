package service

import (
	"context"
	"log/slog"
	"time"

	"errors"

	pb "github.com/Novip1906/tasks-grpc/auth/api/proto/gen"
	"github.com/Novip1906/tasks-grpc/auth/internal/config"
	"github.com/Novip1906/tasks-grpc/auth/internal/storage"
	"github.com/Novip1906/tasks-grpc/auth/pkg/logging"
	"github.com/Novip1906/tasks-grpc/auth/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserStorage interface {
	CheckUser(username, password string) (int64, error)
	AddUser(username, password string) error
}

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	cfg *config.Config
	log *slog.Logger
	db  UserStorage
}

func NewAuthService(config *config.Config, log *slog.Logger, db UserStorage) *AuthService {
	return &AuthService{cfg: config, log: log, db: db}
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	username, password := req.GetUsername(), req.GetPassword()

	log := s.log.With(
		slog.String("op", "Auth.Login"),
		slog.String("username", username),
	)

	log.Info("login attempt")

	if username == "" || password == "" {
		log.Error("invalid username or password")
		return nil, status.Error(codes.InvalidArgument, "Username or pass is invalid")
	}

	userId, err := s.db.CheckUser(username, password)

	if errors.Is(err, storage.ErrUserNotFound) {
		log.Error(err.Error())
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if errors.Is(err, storage.ErrWrongPassword) {
		log.Error(err.Error())
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if err != nil {
		log.Error("db.checkUser", logging.Err(err))
		return nil, status.Error(codes.Internal, ErrInternal.Error())
	}

	log.Info("user logged")

	token, err := utils.EncodeJWTToken(userId, s.cfg.JWTSecretKey)
	if err != nil {
		log.Error("jwt error", logging.Err(err))
		return nil, status.Error(codes.Internal, ErrInternal.Error())
	}

	return &pb.LoginResponse{Token: token}, nil
}

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	username, pass := req.GetUsername(), req.GetPassword()

	log := s.log.With(
		slog.String("op", "Auth.Register"),
		slog.String("username", username),
	)

	log.Info("register attempt")

	if len(username) < s.cfg.Params.Username.Min || len(username) > s.cfg.Params.Username.Max {
		log.Error("invalid username")
		return nil, status.Error(codes.InvalidArgument, "Username is invalid")
	}
	if len(pass) < s.cfg.Params.Password.Min || len(pass) > s.cfg.Params.Password.Max {
		log.Error("invalid password")
		return nil, status.Error(codes.InvalidArgument, "Password is invalid")
	}

	err := s.db.AddUser(username, pass)
	if errors.Is(err, storage.ErrUserAlreadyExists) {
		log.Error(err.Error())
		return nil, status.Error(codes.AlreadyExists, err.Error())
	}
	if err != nil {
		log.Error("db.addUser", logging.Err(err))
		return nil, status.Error(codes.Internal, ErrInternal.Error())
	}

	log.Info("user registered")

	return &pb.RegisterResponse{}, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	log := s.log.With(
		slog.String("op", "Auth.ValidateToken"),
		slog.String("token", req.GetToken()),
	)

	log.Info("token validating attempt")

	if req.GetToken() == "" {
		log.Error("token empty")
		return nil, status.Error(codes.InvalidArgument, "Token is empty")
	}

	userId, exp, err := utils.DecodeJWTToken(req.GetToken(), s.cfg.JWTSecretKey)
	if err != nil {
		log.Error("jwt decode error", logging.Err(err))
		return nil, status.Error(codes.Unauthenticated, storage.ErrInvalidToken.Error())
	}

	if time.Now().Unix() > exp {
		log.Error("token expired", logging.Err(err))
		return nil, status.Error(codes.Unauthenticated, storage.ErrExpiredToken.Error())
	}

	log.Info("token is ok")

	return &pb.ValidateTokenResponse{
		UserId: userId,
	}, nil
}
