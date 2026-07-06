package db

import (
	"context"
	"fmt"
)

// UserSecurityProfile holds account security fields for GET /me.
type UserSecurityProfile struct {
	AuthProvider string
	HasPassword  bool
}

// GetUserSecurityProfile loads security-related user fields.
func (d *DB) GetUserSecurityProfile(ctx context.Context, userID string) (UserSecurityProfile, error) {
	p := d.dialect.Placeholder
	query := "SELECT auth_provider, password_hash FROM users WHERE id = " + p(1) + " AND deleted_at IS NULL"

	var authProvider string
	var passwordHash *string
	err := d.sql.QueryRowContext(ctx, query, userID).Scan(&authProvider, &passwordHash)
	if err != nil {
		return UserSecurityProfile{}, fmt.Errorf("get user security profile: %w", translateError(err))
	}
	return UserSecurityProfile{
		AuthProvider: authProvider,
		HasPassword:  passwordHash != nil,
	}, nil
}

// SetUserPasswordHash sets password_hash and optionally switches auth_provider to local.
func (d *DB) SetUserPasswordHash(ctx context.Context, userID, passwordHash string, setLocalProvider bool) error {
	p := d.dialect.Placeholder
	query := "UPDATE users SET password_hash = " + p(1) + ", updated_at = CURRENT_TIMESTAMP"
	args := []any{passwordHash}
	if setLocalProvider {
		query += ", auth_provider = " + p(2)
		args = append(args, "local")
		query += " WHERE id = " + p(3) + " AND deleted_at IS NULL"
		args = append(args, userID)
	} else {
		query += " WHERE id = " + p(2) + " AND deleted_at IS NULL"
		args = append(args, userID)
	}

	result, err := d.sql.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("set user password hash: %w", translateError(err))
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("set user password hash rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}