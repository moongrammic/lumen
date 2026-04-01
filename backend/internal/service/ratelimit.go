package service

import (
	"context"
	"fmt"
	"time"

	"lumen/internal/config"

	"github.com/redis/go-redis/v9"
)

type MessageRateLimiter interface {
	AllowMessage(ctx context.Context, userID string, channelID uint) (bool, error)
}

type RedisMessageRateLimiter struct {
	rdb            *redis.Client
	messagesPer10s int64
	window         time.Duration
}

func NewRedisMessageRateLimiter(rdb *redis.Client, cfg config.RateLimitConfig) *RedisMessageRateLimiter {
	limit := int64(cfg.MessagesPer10s)
	if limit <= 0 {
		limit = 10
	}
	return &RedisMessageRateLimiter{
		rdb:            rdb,
		messagesPer10s: limit,
		window:         10 * time.Second,
	}
}

func (rl *RedisMessageRateLimiter) AllowMessage(ctx context.Context, userID string, channelID uint) (bool, error) {
	key := fmt.Sprintf("ratelimit:msg:%s:%d", userID, channelID)
	pipe := rl.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rl.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	return incr.Val() <= rl.messagesPer10s, nil
}
