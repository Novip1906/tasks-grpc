package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/Novip1906/tasks-grpc/gateway/internal/config"
	"github.com/Novip1906/tasks-grpc/gateway/pkg/logging"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	limiter *redis_rate.Limiter
	rps     int
	burst   int
}

func NewRateLimiter(ctx context.Context, log *slog.Logger, redisCfg *config.Redis, rateLimiterCfg *config.RateLimiter) *RateLimiter {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisCfg.Address,
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(err)
	}
	log.Info("Connected to Redis successfully")

	return &RateLimiter{
		limiter: redis_rate.NewLimiter(rdb),
		rps:     rateLimiterCfg.RPS,
		burst:   rateLimiterCfg.Burst,
	}
}

func (rl *RateLimiter) Middleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			key := fmt.Sprintf("rate_limit:%s", ip)

			res, err := rl.limiter.Allow(ctx, key, redis_rate.PerSecond(rl.rps))
			if err != nil {
				log.Error("redis rate limiter error", logging.Err(err))
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.rps))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(res.ResetAfter.Seconds())))

			if res.Allowed == 0 {
				log.Warn("rate limit exceeded (redis)",
					slog.String("ip", ip),
					slog.Int("remaining", res.Remaining),
				)
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
