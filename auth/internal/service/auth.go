package service

import (
	pb "github.com/Novip1906/tasks-grpc/auth/api/proto/gen"
	"context"
)

type AuthService struct {
	pb.UnimplementedAuthServiceServer
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
}

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {


func (s *AuthService) ValidateToken(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {

}
