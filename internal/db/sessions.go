package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// UserSessionRow is a login session (session_key) for session management APIs.
type UserSessionRow struct {
	ID         string
	LoginIP    *string
	UserAgent  *string
	CreatedAt  string
	LastUsedAt *string
}

const sessionKeyType = "session_key"

// ListUserSessions returns active session keys for a user, newest first.
func (d *DB) ListUserSessions(ctx context.Context, userID string) ([]UserSessionRow, error) {
	p := d.dialect.Placeholder
	now := time.Now().UTC().Format(time.RFC3339)
	query := "SELECT id, login_ip, user_agent, created_at, last_used_at FROM api_keys " +
		"WHERE user_id = " + p(1) + " AND key_type = " + p(2) +
		" AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at > " + p(3) + ") " +
		"ORDER BY created_at DESC"

	rows, err := d.sql.QueryContext(ctx, query, userID, sessionKeyType, now)
	if err != nil {
		return nil, fmt.Errorf("list user sessions: %w", translateError(err))
	}
	defer rows.Close()

	var out []UserSessionRow
	for rows.Next() {
		var row UserSessionRow
		if err := rows.Scan(&row.ID, &row.LoginIP, &row.UserAgent, &row.CreatedAt, &row.LastUsedAt); err != nil {
			return nil, fmt.Errorf("list user sessions scan: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list user sessions rows: %w", err)
	}
	return out, nil
}

// CountUserSessions returns the number of active session keys for a user.
func (d *DB) CountUserSessions(ctx context.Context, userID string) (int, error) {
	p := d.dialect.Placeholder
	now := time.Now().UTC().Format(time.RFC3339)
	query := "SELECT COUNT(*) FROM api_keys " +
		"WHERE user_id = " + p(1) + " AND key_type = " + p(2) +
		" AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at > " + p(3) + ")"

	var count int
	if err := d.sql.QueryRowContext(ctx, query, userID, sessionKeyType, now).Scan(&count); err != nil {
		return 0, fmt.Errorf("count user sessions: %w", translateError(err))
	}
	return count, nil
}

// RevokeUserSessionByID hard-deletes one session key owned by userID.
func (d *DB) RevokeUserSessionByID(ctx context.Context, userID, sessionID string) error {
	p := d.dialect.Placeholder
	query := "DELETE FROM api_keys WHERE id = " + p(1) +
		" AND user_id = " + p(2) + " AND key_type = " + p(3)

	result, err := d.sql.ExecContext(ctx, query, sessionID, userID, sessionKeyType)
	if err != nil {
		return fmt.Errorf("revoke user session: %w", translateError(err))
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke user session rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// RevokeOtherUserSessions hard-deletes all session keys except exceptKeyID.
func (d *DB) RevokeOtherUserSessions(ctx context.Context, userID, exceptKeyID string) (int64, error) {
	p := d.dialect.Placeholder
	query := "DELETE FROM api_keys WHERE user_id = " + p(1) +
		" AND key_type = " + p(2) + " AND id != " + p(3)

	result, err := d.sql.ExecContext(ctx, query, userID, sessionKeyType, exceptKeyID)
	if err != nil {
		return 0, fmt.Errorf("revoke other sessions: %w", translateError(err))
	}
	return result.RowsAffected()
}

// TrimOldestUserSessions deletes the oldest active sessions until count <= keep.
func (d *DB) TrimOldestUserSessions(ctx context.Context, userID string, keep int) error {
	if keep < 0 {
		keep = 0
	}
	count, err := d.CountUserSessions(ctx, userID)
	if err != nil {
		return err
	}
	for count > keep {
		oldestID, err := d.oldestUserSessionID(ctx, userID)
		if err != nil {
			return err
		}
		if oldestID == "" {
			return nil
		}
		if err := d.RevokeUserSessionByID(ctx, userID, oldestID); err != nil {
			return err
		}
		count--
	}
	return nil
}

func (d *DB) oldestUserSessionID(ctx context.Context, userID string) (string, error) {
	p := d.dialect.Placeholder
	now := time.Now().UTC().Format(time.RFC3339)
	query := "SELECT id FROM api_keys " +
		"WHERE user_id = " + p(1) + " AND key_type = " + p(2) +
		" AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at > " + p(3) + ") " +
		"ORDER BY created_at ASC LIMIT 1"

	var id string
	err := d.sql.QueryRowContext(ctx, query, userID, sessionKeyType, now).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("oldest session: %w", translateError(err))
	}
	return id, nil
}

// TouchSessionLastUsed updates last_used_at when stale by at least minInterval.
func (d *DB) TouchSessionLastUsed(ctx context.Context, keyID string, minInterval time.Duration) error {
	p := d.dialect.Placeholder
	cutoff := time.Now().UTC().Add(-minInterval).Format(time.RFC3339)
	query := "UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP " +
		"WHERE id = " + p(1) + " AND key_type = " + p(2) +
		" AND deleted_at IS NULL AND (last_used_at IS NULL OR last_used_at < " + p(3) + ")"

	_, err := d.sql.ExecContext(ctx, query, keyID, sessionKeyType, cutoff)
	if err != nil {
		return fmt.Errorf("touch session last used: %w", translateError(err))
	}
	return nil
}