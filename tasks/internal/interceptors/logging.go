package interceptors

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Novip1906/tasks-grpc/tasks/internal/contextkeys"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		requestID := generateRequestID()

		log := logger.With(
			slog.String("method", info.FullMethod),
			slog.String("request_id", requestID),
		)

		ctx = contextkeys.WithLogger(ctx, log)
		ctx = contextkeys.WithRequestID(ctx, requestID)

		log.Info("request started")

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		statusCode := status.Code(err)

		attributes := []any{
			slog.Duration("duration", duration),
			slog.String("status", statusCode.String()),
		}

		if err != nil {
			attributes = append(attributes, slog.String("error", err.Error()))
			log.Error("request failed", attributes...)
		} else {
			log.Info("request completed", attributes...)
		}

		return resp, err
	}
}

func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}
