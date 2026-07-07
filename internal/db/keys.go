package db

import (
	"context"
	"fmt"
	"time"
)

// KeyRecord holds all columns returned by LoadAllActiveKeys. It carries the
// raw database values needed to populate the in-memory key cache including
// the user metadata that are resolved via JOIN at load time.
// The KeyHash field is intentionally included here because the cache is keyed
// on hash; callers must never expose it in API responses.
type KeyRecord struct {
	// ID is the UUIDv7 primary key of the api_keys row.
	ID string
	// KeyHash is the HMAC-SHA256 hash of the raw key used as the cache key.
	KeyHash string
	// KeyType is one of the keygen package constants (user_key, session_key).
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

	Status             string
	SpendCap           float64
	SpendUsed          float64
	IPWhitelist        string
	IPBlacklist        string
	ModelLimitsEnabled bool
	ModelLimits        string

	// ExpiresAt is the optional key expiry, stored as RFC3339 in the DB.
	ExpiresAt *time.Time

	// UserRole is the RBAC role of the owning user (member, admin, root).
	UserRole string
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
    k.status, k.spend_cap, k.spend_used, k.ip_whitelist, k.ip_blacklist,
    k.model_limits_enabled, k.model_limits,
    k.expires_at,
    COALESCE(u.role, 'member')
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
			r                  KeyRecord
			expiresAtRaw       *string
			modelLimitsEnabled int
		)

		if err := rows.Scan(
			&r.ID, &r.KeyHash, &r.KeyType, &r.Name,
			&r.UserID,
			&r.DailyTokenLimit, &r.MonthlyTokenLimit,
			&r.RequestsPerMinute, &r.RequestsPerDay,
			&r.Status, &r.SpendCap, &r.SpendUsed,
			&r.IPWhitelist, &r.IPBlacklist,
			&modelLimitsEnabled, &r.ModelLimits,
			&expiresAtRaw,
			&r.UserRole,
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
		r.ModelLimitsEnabled = modelLimitsEnabled == 1

		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, skipErrors, fmt.Errorf("load all active keys: rows: %w", err)
	}

	return records, skipErrors, nil
}

// LookupActiveKeyByHash returns a single active key record for auth cache fallback.
func (d *DB) LookupActiveKeyByHash(ctx context.Context, keyHash string) (KeyRecord, error) {
	q := fmt.Sprintf(`
SELECT
    k.id, k.key_hash, k.key_type, k.name,
    k.user_id,
    k.daily_token_limit, k.monthly_token_limit,
    k.requests_per_minute, k.requests_per_day,
    k.status, k.spend_cap, k.spend_used, k.ip_whitelist, k.ip_blacklist,
    k.model_limits_enabled, k.model_limits,
    k.expires_at,
    COALESCE(u.role, 'member')
FROM api_keys k
LEFT JOIN users u ON u.id = k.user_id AND u.deleted_at IS NULL
WHERE k.deleted_at IS NULL
  AND k.key_hash = %s
  AND (k.expires_at IS NULL OR k.expires_at > %s)`,
		d.dialect.Placeholder(1), d.dialect.Placeholder(2))

	now := time.Now().UTC().Format(time.RFC3339)
	var (
		r                  KeyRecord
		expiresAtRaw       *string
		modelLimitsEnabled int
	)
	err := d.sql.QueryRowContext(ctx, q, keyHash, now).Scan(
		&r.ID, &r.KeyHash, &r.KeyType, &r.Name,
		&r.UserID,
		&r.DailyTokenLimit, &r.MonthlyTokenLimit,
		&r.RequestsPerMinute, &r.RequestsPerDay,
		&r.Status, &r.SpendCap, &r.SpendUsed,
		&r.IPWhitelist, &r.IPBlacklist,
		&modelLimitsEnabled, &r.ModelLimits,
		&expiresAtRaw,
		&r.UserRole,
	)
	if err != nil {
		return KeyRecord{}, fmt.Errorf("lookup active key: %w", translateError(err))
	}
	if expiresAtRaw != nil {
		t, parseErr := time.Parse(time.RFC3339, *expiresAtRaw)
		if parseErr != nil {
			return KeyRecord{}, fmt.Errorf("parse expires_at %q: %w", *expiresAtRaw, parseErr)
		}
		r.ExpiresAt = &t
	}
	r.ModelLimitsEnabled = modelLimitsEnabled == 1
	return r, nil
}
