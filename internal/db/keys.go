package db

import (
	"context"
	"fmt"
	"time"
)

// KeyRecord holds all columns returned by LoadAllActiveKeys. It carries the
// raw database values needed to populate the in-memory key cache including
// the org, team, and user metadata that are resolved via JOIN at load time.
// The KeyHash field is intentionally included here because the cache is keyed
// on hash; callers must never expose it in API responses.
type KeyRecord struct {
	// ID is the UUIDv7 primary key of the api_keys row.
	ID string
	// KeyHash is the HMAC-SHA256 hash of the raw key used as the cache key.
	KeyHash string
	// KeyType is one of the keygen package constants (user_key, team_key, sa_key, session_key).
	KeyType string
	// Name is the human-readable label assigned to the key.
	Name string
	// UserID is set only for user and session keys.
	UserID *string

	// Key-level rate and token limits.
	DailyTokenLimit   int64
	MonthlyTokenLimit int64
	RequestsPerMinute int
	RequestsPerDay    int

	// ExpiresAt is the optional key expiry, stored as RFC3339 in the DB.
	ExpiresAt *time.Time

	// IsSystemAdmin is 1 when the owning user has users.is_system_admin set.
	IsSystemAdmin int
}

// LoadAllActiveKeys returns all non-deleted, non-expired API keys with their
// associated user metadata for populating the in-memory key
// cache. Results are ordered by k.id ascending. Rows that fail to scan (due to
// data corruption or an unparseable expires_at value) are skipped; the errors
// for each skipped row are collected and returned so callers can log them
// individually. Each call issues a single query against the database;
// results are not cached by this method.
func (d *DB) LoadAllActiveKeys(ctx context.Context) ([]KeyRecord, []error, error) {
	// The query accepts one parameter: the current UTC time in RFC3339 format,
	// used to filter out keys whose expires_at has already passed.
	q := fmt.Sprintf(`
SELECT
    k.id, k.key_hash, k.key_type, k.name,
    k.user_id,
    k.daily_token_limit, k.monthly_token_limit,
    k.requests_per_minute, k.requests_per_day,
    k.expires_at,
    COALESCE(u.is_system_admin, 0)
FROM api_keys k
LEFT JOIN users u ON u.id = k.user_id AND u.deleted_at IS NULL
WHERE k.deleted_at IS NULL
  AND (k.expires_at IS NULL OR k.expires_at > %s)
ORDER BY k.id ASC`, d.dialect.Placeholder(1))

	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := d.sql.QueryContext(ctx, q, now)
	if err != nil {
		return nil, nil, fmt.Errorf("load all active keys: query: %w", err)
	}
	defer rows.Close()

	var records []KeyRecord
	var skipErrors []error

	for rows.Next() {
		var (
			r            KeyRecord
			expiresAtRaw *string
		)

		if err := rows.Scan(
			&r.ID, &r.KeyHash, &r.KeyType, &r.Name,
			&r.UserID,
			&r.DailyTokenLimit, &r.MonthlyTokenLimit,
			&r.RequestsPerMinute, &r.RequestsPerDay,
			&expiresAtRaw,
			&r.IsSystemAdmin,
		); err != nil {
			skipErrors = append(skipErrors, fmt.Errorf("scan row: %w", err))
			continue
		}

		if expiresAtRaw != nil {
			t, err := time.Parse(time.RFC3339, *expiresAtRaw)
			if err != nil {
				skipErrors = append(skipErrors, fmt.Errorf("parse expires_at %q: %w", *expiresAtRaw, err))
				continue
			}
			r.ExpiresAt = &t
		}

		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, skipErrors, fmt.Errorf("load all active keys: rows: %w", err)
	}

	return records, skipErrors, nil
}
