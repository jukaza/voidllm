package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const loginChallengeKeyPart = "login-2fa"

// SetLoginChallenge stores a pending 2FA login challenge.
func (c *Client) SetLoginChallenge(ctx context.Context, token string, payload []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, c.key(loginChallengeKeyPart, token), payload, ttl).Err()
}

// ConsumeLoginChallenge atomically reads and deletes a login challenge.
func (c *Client) ConsumeLoginChallenge(ctx context.Context, token string) ([]byte, bool, error) {
	val, err := c.rdb.GetDel(ctx, c.key(loginChallengeKeyPart, token)).Bytes()
	if err == goredis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}