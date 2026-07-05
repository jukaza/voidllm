package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const oauthExchangeKeyPart = "oauth-exchange"

// SetOAuthExchange stores a single-use OAuth code payload with TTL.
func (c *Client) SetOAuthExchange(ctx context.Context, code string, payload []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, c.key(oauthExchangeKeyPart, code), payload, ttl).Err()
}

// ConsumeOAuthExchange atomically reads and deletes a stored OAuth code payload.
func (c *Client) ConsumeOAuthExchange(ctx context.Context, code string) ([]byte, bool, error) {
	val, err := c.rdb.GetDel(ctx, c.key(oauthExchangeKeyPart, code)).Bytes()
	if err == goredis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}