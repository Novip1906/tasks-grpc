package service

import (
	"context"
	"time"

	"errors"

	pb "github.com/Novip1906/tasks-grpc/auth/api/proto/gen"
	"github.com/Novip1906/tasks-grpc/auth/internal/config"
	appErrors "github.com/Novip1906/tasks-grpc/auth/internal/errors"
	"github.com/Novip1906/tasks-grpc/auth/internal/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserStorage interface {
	CheckUser(username, password string) error
	AddUser(username, password string) error
}

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	cfg     *config.Config
	storage UserStorage
}

func NewAuthService(config *config.Config, storage UserStorage) *AuthService {
	return &AuthService{cfg: config, storage: storage}
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	username, password := req.GetUsername(), req.GetPassword()
	if username == "" || password == "" {
		return nil, status.Error(codes.InvalidArgument, "Username or pass is invalid")
	}

	err := s.storage.CheckUser(username, password)

	if errors.Is(err, appErrors.ErrUserNotFound) {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if errors.Is(err, appErrors.ErrWrongPassword) {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, appErrors.ErrInternal.Error())
	}

	token, err := utils.EncodeJWTToken(username, s.cfg.JWTSecretKey)
	if err != nil {
		return nil, status.Error(codes.Internal, appErrors.ErrInternal.Error())
	}

	return &pb.LoginResponse{Token: token}, nil
}

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	username, pass := req.GetUsername(), req.GetPassword()
	if len(username) < s.cfg.Params.Username.Min || len(username) < s.cfg.Params.Username.Max {
		return nil, status.Error(codes.InvalidArgument, "Username is invalid")
	}
	if len(pass) < s.cfg.Params.Password.Min || len(pass) > s.cfg.Params.Password.Max {
		return nil, status.Error(codes.InvalidArgument, "Password is invalid")
	}

	err := s.storage.AddUser(username, pass)
	if errors.Is(err, appErrors.ErrUserAlreadyExists) {
		return nil, status.Error(codes.AlreadyExists, err.Error())
	}
	if err == nil {
		return nil, status.Error(codes.Internal, appErrors.ErrInternal.Error())
	}

	return &pb.RegisterResponse{}, nil
}

func (s *AuthService) ValidateTokenValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if req.GetToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "Token is empty")
	}

	username, exp, err := utils.DecodeJWTToken(req.GetToken(), s.cfg.JWTSecretKey)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, appErrors.ErrInvalidToken.Error())
	}

	if time.Now().Unix() > exp {
		return nil, status.Error(codes.Unauthenticated, appErrors.ErrExpiredToken.Error())
	}

	return &pb.ValidateTokenResponse{
		Username: username,
		Exp:      exp,
	}, nil
}
