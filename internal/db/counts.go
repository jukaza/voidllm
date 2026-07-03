package db

import (
	"context"
	"fmt"
	"time"
)

// CountActiveKeys returns the number of non-deleted, non-expired API keys globally.
func (d *DB) CountActiveKeys(ctx context.Context) (int, error) {
	var count int
	err := d.sql.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM api_keys WHERE deleted_at IS NULL AND (expires_at IS NULL OR expires_at > "+d.dialect.Placeholder(1)+")",
		time.Now().UTC().Format(time.RFC3339),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("CountActiveKeys: %w", translateError(err))
	}
	return count, nil
}

// CountUserKeys returns the number of active, non-expired API keys owned by a user.
func (d *DB) CountUserKeys(ctx context.Context, userID string) (int, error) {
	p := d.dialect.Placeholder
	var count int
	err := d.sql.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM api_keys WHERE user_id = "+p(1)+
			" AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at > "+p(2)+")",
		userID, time.Now().UTC().Format(time.RFC3339),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("CountUserKeys user %s: %w", userID, translateError(err))
	}
	return count, nil
}
