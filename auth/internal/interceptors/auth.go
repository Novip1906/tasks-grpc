package interceptors

import (
	"context"
	"log/slog"
	"strings"

	pb "github.com/Novip1906/tasks-grpc/auth/api/proto/gen"
	"github.com/Novip1906/tasks-grpc/auth/internal/contextkeys"
	"github.com/Novip1906/tasks-grpc/auth/internal/service"
	"github.com/Novip1906/tasks-grpc/auth/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthUnaryInterceptor(authService service.AuthService, log *slog.Logger) grpc.UnaryServerInterceptor {
	authRequiredMethods := map[string]bool{
		"/auth.AuthService/ChangeEmail": true,
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if !authRequiredMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		log := log.With(slog.String("interceptor", "auth"))

		log.Debug("begin")
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Error("context error")
			return nil, status.Error(codes.Unauthenticated, "Missing data")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			log.Error("auth headers empty")
			return nil, status.Error(codes.Unauthenticated, "Authorization header required")
		}

		authHeader := authHeaders[0]
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			log.Error("invalid authorization format", slog.String("header", authHeader))
			return nil, status.Error(codes.Unauthenticated, "Invalid authorization format. Expected 'Bearer <token>'")
		}

		token := strings.TrimPrefix(authHeader, bearerPrefix)
		if token == "" {
			log.Error("missing token after bearer prefix")
			return nil, status.Error(codes.Unauthenticated, "Missing token after 'Bearer '")
		}

		tokenResp, err := authService.ValidateToken(ctx, &pb.ValidateTokenRequest{Token: token})
		if err != nil {
			log.Error("validate token error", logging.Err(err))
			return nil, status.Error(codes.Unauthenticated, "Invalid token")
		}

		userId, username, email := tokenResp.GetUserId(), tokenResp.GetUsername(), tokenResp.GetEmail()

		log.Info("token is ok:",
			slog.Int64("user_id", userId),
			slog.String("email", email),
			slog.String("username", username),
		)

		claims := &contextkeys.TokenClaims{
			UserId:   userId,
			Email:    email,
			Username: username,
		}

		ctx = contextkeys.WithTokenClaims(ctx, claims)

		log = contextkeys.GetLogger(ctx).With(slog.Int64("user_id", tokenResp.UserId))
		ctx = contextkeys.WithLogger(ctx, log)

		return handler(ctx, req)
	}
}
