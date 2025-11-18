package storage

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStorage struct {
	client  *redis.Client
	codeExp time.Duration
}

var (
	emailPrefix = "code:"
)

func NewRedisStorage(ctx context.Context, addr, password string, db int, log *slog.Logger, codeExp time.Duration) *RedisStorage {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(err)
	}
	log.Info("Connected to Redis successfully")

	return &RedisStorage{client: rdb, codeExp: codeExp}
}

func (r *RedisStorage) SetCode(ctx context.Context, email, code string, userId int64) error {
	err := r.client.HSet(ctx, emailPrefix+email, map[string]interface{}{
		"code":    code,
		"user_id": userId,
	}).Err()
	if err != nil {
		return err
	}

	return r.client.Expire(ctx, emailPrefix+email, r.codeExp).Err()
}

func (r *RedisStorage) GetCode(ctx context.Context, email string) (string, int64, error) {
	res, err := r.client.HGetAll(ctx, emailPrefix+email).Result()
	if err != nil {
		return "", 0, err
	}

	if len(res) == 0 {
		return "", 0, ErrCodeNotFound
	}

	code := res["code"]
	userId, err := strconv.ParseInt(res["user_id"], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("userId parse error: %s", res["user_id"])
	}

	return code, int64(userId), nil
}

func (r *RedisStorage) DeleteCode(ctx context.Context, email string) error {
	count, err := r.client.Del(ctx, email).Result()
	if count == 0 {
		return ErrCodeNotFound
	}
	return err
}
