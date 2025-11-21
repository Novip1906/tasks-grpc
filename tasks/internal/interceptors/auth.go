package interceptors

import (
	"context"
	"log/slog"
	"time"

	authpb "github.com/Novip1906/tasks-grpc/tasks/internal/auth_gen"
	"github.com/Novip1906/tasks-grpc/tasks/internal/contextkeys"
	"github.com/Novip1906/tasks-grpc/tasks/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthUnaryInterceptor(authClient authpb.AuthServiceClient, timeout time.Duration, log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		log := log.With(slog.String("interceptor", "auth"))

		log.Info("begin")
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

		token := authHeaders[0]

		ctxAuth, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		tokenResp, err := authClient.ValidateToken(ctxAuth, &authpb.ValidateTokenRequest{Token: token})
		if err != nil {
			log.Error("auth request error", logging.Err(err))
			return nil, status.Error(codes.Unauthenticated, "Unauthenticated")
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
