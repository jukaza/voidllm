package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UserSecurityProfile holds account security fields for GET /me.
type UserSecurityProfile struct {
	AuthProvider  string
	HasPassword   bool
	TwoFAEnabled  bool
}

// GetUserSecurityProfile loads security-related user fields.
func (d *DB) GetUserSecurityProfile(ctx context.Context, userID string) (UserSecurityProfile, error) {
	p := d.dialect.Placeholder
	query := "SELECT auth_provider, password_hash, totp_enabled_at FROM users WHERE id = " + p(1) + " AND deleted_at IS NULL"

	var authProvider string
	var passwordHash *string
	var totpEnabledAt *string
	err := d.sql.QueryRowContext(ctx, query, userID).Scan(&authProvider, &passwordHash, &totpEnabledAt)
	if err != nil {
		return UserSecurityProfile{}, fmt.Errorf("get user security profile: %w", translateError(err))
	}
	return UserSecurityProfile{
		AuthProvider: authProvider,
		HasPassword:  passwordHash != nil,
		TwoFAEnabled: totpEnabledAt != nil && *totpEnabledAt != "",
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

// GetUserTOTPEncrypted returns encrypted TOTP secret and whether 2FA is enabled.
func (d *DB) GetUserTOTPEncrypted(ctx context.Context, userID string) (encrypted *string, enabled bool, err error) {
	p := d.dialect.Placeholder
	query := "SELECT totp_secret_encrypted, totp_enabled_at FROM users WHERE id = " + p(1) + " AND deleted_at IS NULL"

	var enabledAt *string
	err = d.sql.QueryRowContext(ctx, query, userID).Scan(&encrypted, &enabledAt)
	if err != nil {
		return nil, false, fmt.Errorf("get user totp: %w", translateError(err))
	}
	return encrypted, enabledAt != nil && *enabledAt != "", nil
}

// EnableUserTOTP stores encrypted secret and marks 2FA enabled.
func (d *DB) EnableUserTOTP(ctx context.Context, userID, encryptedSecret string) error {
	p := d.dialect.Placeholder
	now := time.Now().UTC().Format(time.RFC3339)
	query := "UPDATE users SET totp_secret_encrypted = " + p(1) +
		", totp_enabled_at = " + p(2) +
		", updated_at = CURRENT_TIMESTAMP WHERE id = " + p(3) + " AND deleted_at IS NULL"

	result, err := d.sql.ExecContext(ctx, query, encryptedSecret, now, userID)
	if err != nil {
		return fmt.Errorf("enable user totp: %w", translateError(err))
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("enable user totp rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DisableUserTOTP clears TOTP fields for a user.
func (d *DB) DisableUserTOTP(ctx context.Context, userID string) error {
	p := d.dialect.Placeholder
	query := "UPDATE users SET totp_secret_encrypted = NULL, totp_enabled_at = NULL, updated_at = CURRENT_TIMESTAMP " +
		"WHERE id = " + p(1) + " AND deleted_at IS NULL"

	result, err := d.sql.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("disable user totp: %w", translateError(err))
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("disable user totp rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// InsertTOTPBackupCode stores a hashed backup code.
func (d *DB) InsertTOTPBackupCode(ctx context.Context, userID, codeHash string) error {
	id, err := uuidNewV7()
	if err != nil {
		return err
	}
	p := d.dialect.Placeholder
	query := "INSERT INTO user_totp_backup_codes (id, user_id, code_hash, created_at) VALUES (" +
		p(1) + ", " + p(2) + ", " + p(3) + ", CURRENT_TIMESTAMP)"
	_, err = d.sql.ExecContext(ctx, query, id, userID, codeHash)
	if err != nil {
		return fmt.Errorf("insert backup code: %w", translateError(err))
	}
	return nil
}

// DeleteTOTPBackupCodes removes all backup codes for a user.
func (d *DB) DeleteTOTPBackupCodes(ctx context.Context, userID string) error {
	p := d.dialect.Placeholder
	_, err := d.sql.ExecContext(ctx, "DELETE FROM user_totp_backup_codes WHERE user_id = "+p(1), userID)
	if err != nil {
		return fmt.Errorf("delete backup codes: %w", translateError(err))
	}
	return nil
}

// ConsumeTOTPBackupCode marks a matching unused backup code as used. Returns true if consumed.
func (d *DB) ConsumeTOTPBackupCode(ctx context.Context, userID, codeHash string) (bool, error) {
	p := d.dialect.Placeholder
	now := time.Now().UTC().Format(time.RFC3339)
	query := "UPDATE user_totp_backup_codes SET used_at = " + p(1) +
		" WHERE user_id = " + p(2) + " AND code_hash = " + p(3) + " AND used_at IS NULL"

	result, err := d.sql.ExecContext(ctx, query, now, userID, codeHash)
	if err != nil {
		return false, fmt.Errorf("consume backup code: %w", translateError(err))
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("consume backup code rows: %w", err)
	}
	return n > 0, nil
}

// UserHasTOTPEnabled reports whether totp_enabled_at is set.
func (d *DB) UserHasTOTPEnabled(ctx context.Context, userID string) (bool, error) {
	p := d.dialect.Placeholder
	var enabledAt *string
	err := d.sql.QueryRowContext(ctx,
		"SELECT totp_enabled_at FROM users WHERE id = "+p(1)+" AND deleted_at IS NULL",
		userID,
	).Scan(&enabledAt)
	if err == sql.ErrNoRows {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("user totp enabled: %w", translateError(err))
	}
	return enabledAt != nil && *enabledAt != "", nil
}

func uuidNewV7() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return id.String(), nil
}