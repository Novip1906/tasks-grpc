package service

import (
	"context"
	"log/slog"
	"time"

	"errors"

	pb "github.com/Novip1906/tasks-grpc/auth/api/proto/gen"
	"github.com/Novip1906/tasks-grpc/auth/internal/config"
	"github.com/Novip1906/tasks-grpc/auth/internal/contextkeys"
	"github.com/Novip1906/tasks-grpc/auth/internal/models"
	"github.com/Novip1906/tasks-grpc/auth/internal/storage"
	"github.com/Novip1906/tasks-grpc/auth/pkg/logging"
	"github.com/Novip1906/tasks-grpc/auth/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserStorage interface {
	CheckUsernamePassword(username, password string) (userId int64, email string, err error)
	CheckEmailExists(email string) (bool, error)
	AddUser(username, password, email string) (int64, error)
	SetEmail(userId int64, email string) error
}

type CodeStorage interface {
	SetCode(ctx context.Context, email, code string, userId int64) error
	GetCode(ctx context.Context, email string) (string, int64, error)
	DeleteCode(ctx context.Context, email string) error
}

type EmailSender interface {
	SendVerificationEmail(ctx context.Context, message *models.EmailVerificationMessage) error
}

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	cfg         *config.Config
	log         *slog.Logger
	userDb      UserStorage
	codeDb      CodeStorage
	emailSender EmailSender
}

func NewAuthService(config *config.Config, log *slog.Logger, userDb UserStorage, codesDb CodeStorage, verProducer EmailSender) *AuthService {
	return &AuthService{cfg: config, log: log, userDb: userDb, codeDb: codesDb, emailSender: verProducer}
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	username, password := req.GetUsername(), req.GetPassword()

	log := contextkeys.GetLogger(ctx)

	log.Debug("login attempt")

	if username == "" || password == "" {
		log.Error("invalid username or password")
		return nil, status.Error(codes.InvalidArgument, "Username or pass is invalid")
	}

	userId, email, err := s.userDb.CheckUsernamePassword(username, password)

	if errors.Is(err, storage.ErrUserNotFound) {
		log.Error("user not found", logging.Err(err))
		return nil, status.Error(codes.Unauthenticated, "User not found")
	}
	if errors.Is(err, storage.ErrWrongPassword) {
		log.Error("wrong pass", logging.Err(err))
		return nil, status.Error(codes.Unauthenticated, "Wrong password")
	}
	if err != nil {
		log.Error("userDB error", logging.DbErr("CheckUsernamePassword", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	log.Info("user logged", "user id", userId)

	token, err := utils.EncodeJWTToken(userId, email, username, s.cfg.JWTSecretKey)
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

	if !utils.EmailIsValid(email) {
		log.Error("invalid email", "email", email)
		return nil, status.Error(codes.InvalidArgument, "Email is invalid")
	}

	if _, _, err := s.codeDb.GetCode(ctx, email); err == nil {
		log.Error("email in codesDB", "email", email)
		return nil, status.Error(codes.AlreadyExists, "Email is already exists")
	}

	emailExists, err := s.userDb.CheckEmailExists(email)
	if err != nil {
		log.Error("userDB error", logging.DbErr("CheckEmailExists", err))
	}
	if emailExists {
		log.Error("email in userDB", "email", email)
		return nil, status.Error(codes.AlreadyExists, "Email is already exists")
	}

	userId, err := s.userDb.AddUser(username, pass, "")
	if errors.Is(err, storage.ErrUserAlreadyExists) {
		log.Error(err.Error())
		return nil, status.Error(codes.AlreadyExists, "User is already exists")
	}
	if err != nil {
		log.Error("userDB error", logging.DbErr("AddUser", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	if email == "" {
		return &pb.RegisterResponse{}, nil
	}

	code := utils.GenerateVerificationCode()
	if err = s.codeDb.SetCode(ctx, email, code, userId); err != nil {
		log.Error("codesDB error", logging.DbErr("SetCode", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	log.Debug("starting sending kafka verification message")

	asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = s.emailSender.SendVerificationEmail(asyncCtx, &models.EmailVerificationMessage{
		Email:    email,
		Code:     code,
		Username: username,
	})

	if err != nil {
		log.Error("kafka error", "email", email, logging.Err(err))
	} else {
		log.Info("kafka verification message sent", "email", email)
	}

	return &pb.RegisterResponse{}, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	log := contextkeys.GetLogger(ctx)

	log.Debug("token validating attempt")

	if req.GetToken() == "" {
		log.Error("token empty")
		return nil, status.Error(codes.InvalidArgument, "Token is empty")
	}

	tokenClaims, err := utils.DecodeJWTToken(req.GetToken(), s.cfg.JWTSecretKey)
	if err != nil {
		log.Error("jwt decode error", logging.Err(err))
		return nil, status.Error(codes.Unauthenticated, "Invalid authorization params")
	}

	if time.Now().Unix() > tokenClaims.ExpiresAt.Unix() {
		log.Error("token expired", logging.Err(err))
		return nil, status.Error(codes.Unauthenticated, "Expired authorization")
	}

	log.Info("token is ok")

	return &pb.ValidateTokenResponse{
		UserId:   tokenClaims.UserId,
		Username: tokenClaims.Username,
		Email:    tokenClaims.Email,
	}, nil
}

func (s *AuthService) ValidateVerificationCode(ctx context.Context, req *pb.ValidateCodeRequest) (*pb.ValidateCodeResponse, error) {
	log := contextkeys.GetLogger(ctx)
	email := req.GetEmail()

	log.Debug("code validating attempt")

	codeDb, userId, err := s.codeDb.GetCode(ctx, email)
	if errors.Is(err, storage.ErrCodeNotFound) {
		log.Error("email key in redis is empty")
		return nil, status.Error(codes.NotFound, "Code is expired")
	}
	if err != nil {
		log.Error("codeDB error", logging.DbErr("GetCode", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	if req.Code != codeDb {
		log.Error("codes are different")
		return nil, status.Error(codes.NotFound, "Wrong code")
	}

	if err = s.userDb.SetEmail(userId, email); err != nil {
		log.Error("userDB error", logging.DbErr("SetEmail", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	if err = s.codeDb.DeleteCode(ctx, email); err != nil {
		log.Error("codeDB", logging.DbErr("DeleteCode", err))
	}

	log.Info("code validated")

	return &pb.ValidateCodeResponse{}, nil
}
