package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Limiter {
	return &Limiter{rdb: rdb}
}

func (l *Limiter) Allow(ctx context.Context, key string, limit int) (bool, error) {
	if limit <= 0 {
		return true, nil
	}
	if l == nil || l.rdb == nil {
		return true, nil
	}

	n, err := l.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("rate incr: %w", err)
	}
	if n == 1 {
		if err := l.rdb.Expire(ctx, key, time.Minute).Err(); err != nil {
			return false, fmt.Errorf("rate expire: %w", err)
		}
	}
	return n <= int64(limit), nil
}