package ratelimit

import (
	"context"
	"log/slog"
	"time"

	voidredis "github.com/voidmind-io/voidllm/internal/redis"
)

var _ Checker = (*RedisChecker)(nil)

type rateCheck struct {
	scope  string
	id     string
	limit  int
	window time.Duration
}

// RedisChecker evaluates per-key rate limits using Redis counters.
type RedisChecker struct {
	client *voidredis.Client
	log    *slog.Logger
}

// NewRedisChecker returns a RedisChecker backed by the given Redis client.
func NewRedisChecker(client *voidredis.Client, log *slog.Logger) *RedisChecker {
	return &RedisChecker{client: client, log: log}
}

// CheckRate verifies per-key RPM/RPD limits against Redis.
func (r *RedisChecker) CheckRate(keyID string, keyLimits Limits) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	checks := []rateCheck{
		{"key", keyID, keyLimits.RequestsPerMinute, time.Minute},
		{"key", keyID, keyLimits.RequestsPerDay, 24 * time.Hour},
	}

	for _, c := range checks {
		if c.limit <= 0 {
			continue
		}
		allowed, err := r.client.CheckRate(ctx, c.scope, c.id, c.limit, c.window)
		if err != nil {
			r.log.Warn("redis rate check failed, allowing request",
				slog.String("scope", c.scope),
				slog.String("error", err.Error()),
			)
			continue
		}
		if !allowed {
			return ErrRateLimitExceeded
		}
	}
	return nil
}