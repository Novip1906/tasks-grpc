package service

import (
	"context"
	"log/slog"
	"time"

	"errors"

	pb "github.com/Novip1906/tasks-grpc/auth/api/proto/gen"
	"github.com/Novip1906/tasks-grpc/auth/internal/config"
	"github.com/Novip1906/tasks-grpc/auth/internal/contextkeys"
	"github.com/Novip1906/tasks-grpc/auth/internal/kafka"
	"github.com/Novip1906/tasks-grpc/auth/internal/models"
	"github.com/Novip1906/tasks-grpc/auth/internal/storage"
	"github.com/Novip1906/tasks-grpc/auth/pkg/logging"
	"github.com/Novip1906/tasks-grpc/auth/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserStorage interface {
	CheckUser(username, password string) (int64, error)
	AddUser(username, password string) (int64, error)
	SetEmail(userId int64, email string) error
}

type CodeStorage interface {
	SetCode(ctx context.Context, email, code string, userId int64) error
	GetCode(ctx context.Context, email string) (string, int64, error)
}

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	cfg                  *config.Config
	log                  *slog.Logger
	userDb               UserStorage
	codesDb              CodeStorage
	verificationProducer *kafka.EmailVerificationProducer
}

func NewAuthService(config *config.Config, log *slog.Logger, userDb UserStorage, codesDb CodeStorage, verProducer *kafka.EmailVerificationProducer) *AuthService {
	return &AuthService{cfg: config, log: log, userDb: userDb, codesDb: codesDb, verificationProducer: verProducer}
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	username, password := req.GetUsername(), req.GetPassword()

	log := contextkeys.GetLogger(ctx)

	log.Debug("login attempt")

	if username == "" || password == "" {
		log.Error("invalid username or password")
		return nil, status.Error(codes.InvalidArgument, "Username or pass is invalid")
	}

	userId, err := s.userDb.CheckUser(username, password)

	if errors.Is(err, storage.ErrUserNotFound) {
		log.Error("user not found", logging.Err(err))
		return nil, status.Error(codes.Unauthenticated, "User not found")
	}
	if errors.Is(err, storage.ErrWrongPassword) {
		log.Error("wrong pass", logging.Err(err))
		return nil, status.Error(codes.Unauthenticated, "Wrong password")
	}
	if err != nil {
		log.Error("postgres error", logging.DbErr("CheckUser", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	log.Info("user logged", "user id", userId)

	token, err := utils.EncodeJWTToken(userId, s.cfg.JWTSecretKey)
	if err != nil {
		log.Error("jwt error", logging.Err(err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	return &pb.LoginResponse{Token: token}, nil
}

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	username, email, pass := req.GetUsername(), req.GetEmail(), req.GetPassword()

	log := contextkeys.GetLogger(ctx)

	log.Debug("register attempt")

	if !utils.UsernameIsValid(username, s.cfg) {
		log.Error("invalid username", "username", username)
		return nil, status.Error(codes.InvalidArgument, "Username is invalid")
	}
	if !utils.PasswordIsValid(pass, s.cfg) {
		log.Error("invalid password")
		return nil, status.Error(codes.InvalidArgument, "Password is invalid")
	}

	if !utils.EmailIsValid(email, s.cfg) {
		log.Error("invalid email", "email", email)
		return nil, status.Error(codes.InvalidArgument, "Email is invalid")
	}

	userId, err := s.userDb.AddUser(username, pass)
	if errors.Is(err, storage.ErrUserAlreadyExists) {
		log.Error(err.Error())
		return nil, status.Error(codes.AlreadyExists, "User is already exists")
	}
	if err != nil {
		log.Error("postgres error", logging.DbErr("AddUser", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	code := utils.GenerateVerificationCode()
	if err = s.codesDb.SetCode(ctx, email, code, userId); err != nil {
		log.Error("redis error", logging.DbErr("SetCode", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	log.Debug("starting sending kafka verification message")

	go func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := s.verificationProducer.SendVerificationEmail(asyncCtx, &models.EmailVerificationMessage{
			Email:    email,
			Code:     code,
			Username: username,
		})

		if err != nil {
			s.log.Error("async kafka error", slog.String("email", email), logging.Err(err))
		} else {
			s.log.Info("async kafka verification message sent", slog.String("email", email))
		}
	}()

	return &pb.RegisterResponse{}, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	log := contextkeys.GetLogger(ctx)

	log.Debug("token validating attempt")

	if req.GetToken() == "" {
		log.Error("token empty")
		return nil, status.Error(codes.InvalidArgument, "Token is empty")
	}

	userId, exp, err := utils.DecodeJWTToken(req.GetToken(), s.cfg.JWTSecretKey)
	if err != nil {
		log.Error("jwt decode error", logging.Err(err))
		return nil, status.Error(codes.Unauthenticated, "Invalid authorization params")
	}

	if time.Now().Unix() > exp {
		log.Error("token expired", logging.Err(err))
		return nil, status.Error(codes.Unauthenticated, "Expired authorization")
	}

	log.Info("token is ok")

	return &pb.ValidateTokenResponse{
		UserId: userId,
	}, nil
}

func (s *AuthService) ValidateVerificationCode(ctx context.Context, req *pb.ValidateCodeRequest) (*pb.ValidateCodeResponse, error) {
	log := contextkeys.GetLogger(ctx)

	log.Debug("code validating attempt")

	codeDb, userId, err := s.codesDb.GetCode(ctx, req.Email)
	if errors.Is(err, storage.ErrCodeNotFound) {
		log.Error("email key in redis is empty")
		return nil, status.Error(codes.NotFound, "Code is expired")
	}
	if err != nil {
		log.Error("redis error", logging.DbErr("GetCode", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	if req.Code != codeDb {
		log.Error("codes are different")
		return nil, status.Error(codes.NotFound, "Wrong code")
	}

	if err = s.userDb.SetEmail(userId, req.Email); err != nil {
		log.Error("postgres error", logging.DbErr("SetEmail", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	log.Info("code validated")

	return &pb.ValidateCodeResponse{}, nil
}
