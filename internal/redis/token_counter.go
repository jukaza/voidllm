package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// AddTokens increments per-key token usage counters in both daily and monthly
// buckets using a single pipelined write. TTL is set on each key so counters
// are automatically evicted: daily keys expire after 48 hours, monthly keys
// after 35 days.
func (c *Client) AddTokens(ctx context.Context, keyID string, tokens int64) error {
	now := time.Now().UTC()
	dailyBucket := now.Format("2006-01-02")
	monthlyBucket := now.Format("2006-01")

	dailyKeyKey := c.key("token", "daily", "key", keyID, dailyBucket)
	monthlyKeyKey := c.key("token", "monthly", "key", keyID, monthlyBucket)

	pipe := c.rdb.Pipeline()

	pipe.IncrBy(ctx, dailyKeyKey, tokens)
	pipe.IncrBy(ctx, monthlyKeyKey, tokens)
	pipe.Expire(ctx, dailyKeyKey, 48*time.Hour)
	pipe.Expire(ctx, monthlyKeyKey, 35*24*time.Hour)

	_, err := pipe.Exec(ctx)
	return err
}

// GetTokenUsage returns the current token count for a key scope and window.
// window must be "daily" or "monthly". Returns 0 when no usage has been
// recorded yet (key does not exist in Redis).
func (c *Client) GetTokenUsage(ctx context.Context, keyID, window string) (int64, error) {
	now := time.Now().UTC()
	var bucket string
	switch window {
	case "daily":
		bucket = now.Format("2006-01-02")
	case "monthly":
		bucket = now.Format("2006-01")
	default:
		return 0, fmt.Errorf("unknown window: %s", window)
	}

	key := c.key("token", window, "key", keyID, bucket)
	val, err := c.rdb.Get(ctx, key).Int64()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return 0, nil // no usage recorded yet
		}
		return 0, fmt.Errorf("redis get token usage: %w", err)
	}
	return val, nil
}