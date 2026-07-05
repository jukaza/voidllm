package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const totpPendingKeyPart = "totp-pending"

// SetTOTPPending stores a pending TOTP setup secret for a user.
func (c *Client) SetTOTPPending(ctx context.Context, userID string, payload []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, c.key(totpPendingKeyPart, userID), payload, ttl).Err()
}

// GetTOTPPending reads a pending TOTP setup secret.
func (c *Client) GetTOTPPending(ctx context.Context, userID string) ([]byte, bool, error) {
	val, err := c.rdb.Get(ctx, c.key(totpPendingKeyPart, userID)).Bytes()
	if err == goredis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

// DeleteTOTPPending removes a pending TOTP setup secret.
func (c *Client) DeleteTOTPPending(ctx context.Context, userID string) error {
	return c.rdb.Del(ctx, c.key(totpPendingKeyPart, userID)).Err()
}