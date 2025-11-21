package contextkeys

import (
	"context"
	"log/slog"
)

type contextKey string

const (
	RequestIDKey   contextKey = "request_id"
	LoggerKey      contextKey = "logger"
	UserIDKey      contextKey = "user_id"
	TokenClaimsKey contextKey = "token_claims"
)

type TokenClaims struct {
	UserId   int64  `json:"sub"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

func WithTokenClaims(ctx context.Context, claims *TokenClaims) context.Context {
	return context.WithValue(ctx, TokenClaimsKey, claims)
}

func GetTokenClaims(ctx context.Context) (*TokenClaims, bool) {
	claims, ok := ctx.Value(TokenClaimsKey).(*TokenClaims)
	return claims, ok
}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

func GetLogger(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(LoggerKey).(*slog.Logger)
	if !ok {
		return slog.Default()
	}
	return logger
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func GetRequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(RequestIDKey).(string)
	return id, ok
}
