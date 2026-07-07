package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// apiKeySelectColumns is the ordered column list used in all api_keys SELECT queries.
// It must match the scan order in scanAPIKey exactly.
// key_hash is included for cache population; it must never be exposed in API responses.
const apiKeySelectColumns = "id, key_hash, key_hint, key_encrypted, key_type, name, " +
	"user_id, " +
	"daily_token_limit, monthly_token_limit, requests_per_minute, requests_per_day, " +
	"status, spend_cap, spend_used, ip_whitelist, ip_blacklist, " +
	"model_limits_enabled, model_limits, " +
	"expires_at, last_used_at, created_by, created_at, updated_at, deleted_at"

// APIKey represents an API key record in the database.
// KeyHash is included for cache keying and must never be included in API responses.
type APIKey struct {
	ID                  string
	KeyHash             string
	KeyHint             string
	KeyEncrypted        *string
	KeyType             string
	Name                string
	UserID              *string
	DailyTokenLimit     int64
	MonthlyTokenLimit   int64
	RequestsPerMinute   int
	RequestsPerDay      int
	Status              string
	SpendCap            float64
	SpendUsed           float64
	IPWhitelist         string
	IPBlacklist         string
	ModelLimitsEnabled  bool
	ModelLimits         string
	ExpiresAt           *string
	LastUsedAt          *string
	CreatedBy           string
	CreatedAt           string
	UpdatedAt           string
	DeletedAt           *string
}

// CreateAPIKeyParams holds the input for creating an API key.
type CreateAPIKeyParams struct {
	// ID is optional; when empty a new UUIDv7 is generated.
	ID string
	// KeyHash is the HMAC-SHA256 hash of the raw key. Never store the raw key.
	KeyHash            string
	// KeyEncrypted is the AES-GCM ciphertext of the raw key for owner reveal (user keys only).
	KeyEncrypted       *string
	KeyHint            string
	KeyType            string
	Name               string
	UserID             *string
	DailyTokenLimit    int64
	MonthlyTokenLimit  int64
	RequestsPerMinute  int
	RequestsPerDay     int
	Status             string
	SpendCap           float64
	IPWhitelist        string
	IPBlacklist        string
	ModelLimitsEnabled bool
	ModelLimits        string
	ExpiresAt          *string
	LoginIP            *string
	UserAgent          *string
	CreatedBy          string
}

// UpdateAPIKeyParams holds optional fields for updating an API key.
// A nil pointer means the field is not changed.
type UpdateAPIKeyParams struct {
	Name               *string
	DailyTokenLimit    *int64
	MonthlyTokenLimit  *int64
	RequestsPerMinute  *int
	RequestsPerDay     *int
	Status             *string
	SpendCap           *float64
	SpendUsed          *float64
	IPWhitelist        *string
	IPBlacklist        *string
	ModelLimitsEnabled *bool
	ModelLimits        *string
	ExpiresAt          *string
}

// CreateAPIKey inserts a new API key and returns the persisted record.
func (d *DB) CreateAPIKey(ctx context.Context, params CreateAPIKeyParams) (*APIKey, error) {
	idStr := strings.TrimSpace(params.ID)
	if idStr == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("create api key: generate id: %w", err)
		}
		idStr = id.String()
	}

	p := d.dialect.Placeholder
	status := params.Status
	if status == "" {
		status = "active"
	}
	insertQuery := "INSERT INTO api_keys " +
		"(id, key_hash, key_hint, key_encrypted, key_type, name, " +
		"user_id, " +
		"daily_token_limit, monthly_token_limit, requests_per_minute, requests_per_day, " +
		"status, spend_cap, spend_used, ip_whitelist, ip_blacklist, " +
		"model_limits_enabled, model_limits, " +
		"expires_at, login_ip, user_agent, created_by, created_at, updated_at) " +
		"VALUES (" +
		p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", " + p(5) + ", " + p(6) + ", " +
		p(7) + ", " +
		p(8) + ", " + p(9) + ", " + p(10) + ", " + p(11) + ", " +
		p(12) + ", " + p(13) + ", 0, " + p(14) + ", " + p(15) + ", " +
		p(16) + ", " + p(17) + ", " +
		p(18) + ", " + p(19) + ", " + p(20) + ", " + p(21) + ", " +
		"CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"

	selectQuery := "SELECT " + apiKeySelectColumns +
		" FROM api_keys WHERE id = " + p(1) + " AND deleted_at IS NULL"

	var key *APIKey
	err := d.WithTx(ctx, func(q Querier) error {
		modelLimitsEnabled := 0
		if params.ModelLimitsEnabled {
			modelLimitsEnabled = 1
		}
		_, execErr := q.ExecContext(ctx, insertQuery,
			idStr,
			params.KeyHash,
			params.KeyHint,
			params.KeyEncrypted,
			params.KeyType,
			params.Name,
			params.UserID,
			params.DailyTokenLimit,
			params.MonthlyTokenLimit,
			params.RequestsPerMinute,
			params.RequestsPerDay,
			status,
			params.SpendCap,
			params.IPWhitelist,
			params.IPBlacklist,
			modelLimitsEnabled,
			params.ModelLimits,
			params.ExpiresAt,
			params.LoginIP,
			params.UserAgent,
			params.CreatedBy,
		)
		if execErr != nil {
			return translateError(execErr)
		}

		row := q.QueryRowContext(ctx, selectQuery, idStr)
		var scanErr error
		key, scanErr = scanAPIKey(row)
		return scanErr
	})
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return key, nil
}

// GetAPIKey retrieves an active API key by its ID.
// It returns ErrNotFound if the key does not exist or has been soft-deleted.
func (d *DB) GetAPIKey(ctx context.Context, id string) (*APIKey, error) {
	query := "SELECT " + apiKeySelectColumns +
		" FROM api_keys WHERE id = " + d.dialect.Placeholder(1) + " AND deleted_at IS NULL"

	row := d.sql.QueryRowContext(ctx, query, id)
	key, err := scanAPIKey(row)
	if err != nil {
		return nil, fmt.Errorf("get api key %s: %w", id, translateError(err))
	}
	return key, nil
}

// ListAPIKeys returns a page of API keys for the given user, ordered by ID ascending.
// cursor is an exclusive lower bound on ID for keyset pagination; pass "" to start from the beginning.
// limit controls the maximum number of records returned.
// includeDeleted controls whether soft-deleted keys are included.
func (d *DB) ListAPIKeys(ctx context.Context, userID, cursor string, limit int, includeDeleted bool) ([]APIKey, error) {
	p := d.dialect.Placeholder
	argN := 1
	var conditions []string
	var args []any

	// Session keys are internal (login tokens) — never expose them in the API list.
	conditions = append(conditions, "key_type != 'session_key'")

	if !includeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if userID != "" {
		conditions = append(conditions, "user_id = "+p(argN))
		args = append(args, userID)
		argN++
	}
	if cursor != "" {
		conditions = append(conditions, "id > "+p(argN))
		args = append(args, cursor)
		argN++
	}

	query := "SELECT " + apiKeySelectColumns + " FROM api_keys" +
		" WHERE " + strings.Join(conditions, " AND ") +
		" ORDER BY id ASC LIMIT " + p(argN)
	args = append(args, limit)

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list api keys query: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		k, err := scanAPIKeyFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("list api keys scan: %w", err)
		}
		keys = append(keys, *k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list api keys rows: %w", err)
	}

	return keys, nil
}

// UpdateAPIKey applies a partial update to an active API key.
// Only non-nil fields in params are written. If all fields are nil the record
// is returned unchanged without issuing an UPDATE.
// It returns ErrNotFound if the key does not exist or has been soft-deleted.
func (d *DB) UpdateAPIKey(ctx context.Context, id string, params UpdateAPIKeyParams) (*APIKey, error) {
	p := d.dialect.Placeholder
	argN := 1
	var setClauses []string
	var args []any

	if params.Name != nil {
		setClauses = append(setClauses, "name = "+p(argN))
		args = append(args, *params.Name)
		argN++
	}
	if params.DailyTokenLimit != nil {
		setClauses = append(setClauses, "daily_token_limit = "+p(argN))
		args = append(args, *params.DailyTokenLimit)
		argN++
	}
	if params.MonthlyTokenLimit != nil {
		setClauses = append(setClauses, "monthly_token_limit = "+p(argN))
		args = append(args, *params.MonthlyTokenLimit)
		argN++
	}
	if params.RequestsPerMinute != nil {
		setClauses = append(setClauses, "requests_per_minute = "+p(argN))
		args = append(args, *params.RequestsPerMinute)
		argN++
	}
	if params.RequestsPerDay != nil {
		setClauses = append(setClauses, "requests_per_day = "+p(argN))
		args = append(args, *params.RequestsPerDay)
		argN++
	}
	if params.ExpiresAt != nil {
		setClauses = append(setClauses, "expires_at = "+p(argN))
		args = append(args, *params.ExpiresAt)
		argN++
	}
	if params.Status != nil {
		setClauses = append(setClauses, "status = "+p(argN))
		args = append(args, *params.Status)
		argN++
	}
	if params.SpendCap != nil {
		setClauses = append(setClauses, "spend_cap = "+p(argN))
		args = append(args, *params.SpendCap)
		argN++
	}
	if params.SpendUsed != nil {
		setClauses = append(setClauses, "spend_used = "+p(argN))
		args = append(args, *params.SpendUsed)
		argN++
	}
	if params.IPWhitelist != nil {
		setClauses = append(setClauses, "ip_whitelist = "+p(argN))
		args = append(args, *params.IPWhitelist)
		argN++
	}
	if params.IPBlacklist != nil {
		setClauses = append(setClauses, "ip_blacklist = "+p(argN))
		args = append(args, *params.IPBlacklist)
		argN++
	}
	if params.ModelLimitsEnabled != nil {
		enabled := 0
		if *params.ModelLimitsEnabled {
			enabled = 1
		}
		setClauses = append(setClauses, "model_limits_enabled = "+p(argN))
		args = append(args, enabled)
		argN++
	}
	if params.ModelLimits != nil {
		setClauses = append(setClauses, "model_limits = "+p(argN))
		args = append(args, *params.ModelLimits)
		argN++
	}

	if len(setClauses) == 0 {
		return d.GetAPIKey(ctx, id)
	}

	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")

	updateQuery := "UPDATE api_keys SET " + strings.Join(setClauses, ", ") +
		" WHERE id = " + p(argN) + " AND deleted_at IS NULL"
	args = append(args, id)

	selectQuery := "SELECT " + apiKeySelectColumns +
		" FROM api_keys WHERE id = " + p(1) + " AND deleted_at IS NULL"

	var key *APIKey
	err := d.WithTx(ctx, func(q Querier) error {
		result, execErr := q.ExecContext(ctx, updateQuery, args...)
		if execErr != nil {
			return translateError(execErr)
		}

		n, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return fmt.Errorf("rows affected: %w", rowsErr)
		}
		if n == 0 {
			return ErrNotFound
		}

		row := q.QueryRowContext(ctx, selectQuery, id)
		var scanErr error
		key, scanErr = scanAPIKey(row)
		return scanErr
	})
	if err != nil {
		return nil, fmt.Errorf("update api key %s: %w", id, err)
	}
	return key, nil
}

// DeleteAPIKey soft-deletes an active API key by setting deleted_at.
// It returns ErrNotFound if the key does not exist or is already deleted.

// RevokeUserSessions hard-deletes all session keys for a user.
// Called during login and OIDC callback to ensure only one session exists per
// user. Session keys are ephemeral — no audit trail needed, hard delete
// prevents the api_keys table from filling up with dead sessions.
func (d *DB) RevokeUserSessions(ctx context.Context, userID string) error {
	p := d.dialect.Placeholder
	query := "DELETE FROM api_keys WHERE user_id = " + p(1) + " AND key_type = " + p(2)

	_, err := d.sql.ExecContext(ctx, query, userID, "session_key")
	if err != nil {
		return fmt.Errorf("revoke user sessions %s: %w", userID, translateError(err))
	}
	return nil
}

// ChangePasswordAndRevokeOtherSessions atomically updates a user's password hash
// and hard-deletes all session keys for that user except the one identified by
// exceptKeyID. Both operations run in a single transaction: if either fails the
// whole operation is rolled back so the DB is never left in a partially-updated
// state. Returns ErrNotFound if the user does not exist or has been soft-deleted.
func (d *DB) ChangePasswordAndRevokeOtherSessions(ctx context.Context, userID, newPasswordHash, exceptKeyID string) error {
	p := d.dialect.Placeholder

	updateQuery := "UPDATE users SET password_hash = " + p(1) +
		", updated_at = CURRENT_TIMESTAMP" +
		" WHERE id = " + p(2) + " AND deleted_at IS NULL"

	deleteQuery := "DELETE FROM api_keys WHERE user_id = " + p(1) +
		" AND key_type = " + p(2) +
		" AND id != " + p(3)

	return d.WithTx(ctx, func(q Querier) error {
		result, err := q.ExecContext(ctx, updateQuery, newPasswordHash, userID)
		if err != nil {
			return fmt.Errorf("update password hash: %w", translateError(err))
		}
		n, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("update password hash rows affected: %w", err)
		}
		if n == 0 {
			return ErrNotFound
		}

		if _, err := q.ExecContext(ctx, deleteQuery, userID, "session_key", exceptKeyID); err != nil {
			return fmt.Errorf("revoke other sessions: %w", translateError(err))
		}
		return nil
	})
}

func (d *DB) DeleteAPIKey(ctx context.Context, id string) error {
	p := d.dialect.Placeholder
	query := "UPDATE api_keys SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP " +
		"WHERE id = " + p(1) + " AND deleted_at IS NULL"

	result, err := d.sql.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete api key %s: %w", id, translateError(err))
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete api key %s rows affected: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("delete api key %s: %w", id, ErrNotFound)
	}

	return nil
}

// RotateKeyTxResult holds both records produced by RotateKeyTx.
type RotateKeyTxResult struct {
	NewKey *APIKey
	OldKey *APIKey
}

// RotateKeyTx atomically inserts a new API key and updates the old key's expiry
// in a single transaction. newParams describes the replacement key; oldID is the
// ID of the key being rotated; oldExpiresAt is the grace-period deadline to write
// onto the old row. Both records are returned on success.
func (d *DB) RotateKeyTx(ctx context.Context, oldID string, oldExpiresAt string, newParams CreateAPIKeyParams) (*RotateKeyTxResult, error) {
	newIDStr := strings.TrimSpace(newParams.ID)
	if newIDStr == "" {
		newID, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("rotate key tx: generate id: %w", err)
		}
		newIDStr = newID.String()
	}

	p := d.dialect.Placeholder

	rotateStatus := newParams.Status
	if rotateStatus == "" {
		rotateStatus = "active"
	}
	modelLimitsEnabled := 0
	if newParams.ModelLimitsEnabled {
		modelLimitsEnabled = 1
	}
	insertQuery := "INSERT INTO api_keys " +
		"(id, key_hash, key_hint, key_encrypted, key_type, name, " +
		"user_id, " +
		"daily_token_limit, monthly_token_limit, requests_per_minute, requests_per_day, " +
		"status, spend_cap, spend_used, ip_whitelist, ip_blacklist, " +
		"model_limits_enabled, model_limits, " +
		"expires_at, created_by, created_at, updated_at) " +
		"VALUES (" +
		p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", " + p(5) + ", " + p(6) + ", " +
		p(7) + ", " +
		p(8) + ", " + p(9) + ", " + p(10) + ", " + p(11) + ", " +
		p(12) + ", " + p(13) + ", 0, " + p(14) + ", " + p(15) + ", " +
		p(16) + ", " + p(17) + ", " +
		p(18) + ", " + p(19) + ", " +
		"CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"

	updateQuery := "UPDATE api_keys SET expires_at = " + p(1) + ", key_encrypted = NULL, updated_at = CURRENT_TIMESTAMP" +
		" WHERE id = " + p(2) + " AND deleted_at IS NULL"

	selectQuery := "SELECT " + apiKeySelectColumns +
		" FROM api_keys WHERE id = " + p(1)

	var result RotateKeyTxResult
	txErr := d.WithTx(ctx, func(q Querier) error {
		_, execErr := q.ExecContext(ctx, insertQuery,
			newIDStr,
			newParams.KeyHash,
			newParams.KeyHint,
			newParams.KeyEncrypted,
			newParams.KeyType,
			newParams.Name,
			newParams.UserID,
			newParams.DailyTokenLimit,
			newParams.MonthlyTokenLimit,
			newParams.RequestsPerMinute,
			newParams.RequestsPerDay,
			rotateStatus,
			newParams.SpendCap,
			newParams.IPWhitelist,
			newParams.IPBlacklist,
			modelLimitsEnabled,
			newParams.ModelLimits,
			newParams.ExpiresAt,
			newParams.CreatedBy,
		)
		if execErr != nil {
			return fmt.Errorf("insert new key: %w", translateError(execErr))
		}

		updateResult, execErr := q.ExecContext(ctx, updateQuery, oldExpiresAt, oldID)
		if execErr != nil {
			return fmt.Errorf("update old key expiry: %w", translateError(execErr))
		}
		n, rowsErr := updateResult.RowsAffected()
		if rowsErr != nil {
			return fmt.Errorf("update old key rows affected: %w", rowsErr)
		}
		if n == 0 {
			return fmt.Errorf("update old key expiry: %w", ErrNotFound)
		}

		newRow := q.QueryRowContext(ctx, selectQuery, newIDStr)
		var scanErr error
		result.NewKey, scanErr = scanAPIKey(newRow)
		if scanErr != nil {
			return fmt.Errorf("scan new key: %w", scanErr)
		}

		oldRow := q.QueryRowContext(ctx, selectQuery, oldID)
		result.OldKey, scanErr = scanAPIKey(oldRow)
		if scanErr != nil {
			return fmt.Errorf("scan old key: %w", scanErr)
		}

		return nil
	})
	if txErr != nil {
		return nil, fmt.Errorf("rotate key tx: %w", txErr)
	}
	return &result, nil
}

// GetUserRole resolves the effective RBAC role for a user.
func (d *DB) GetUserRole(ctx context.Context, userID string) (string, error) {
	role, status, err := d.GetUserAuthState(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user role user %s: %w", userID, err)
	}
	if status == UserStatusDisabled {
		return "", ErrNotFound
	}
	return role, nil
}

type apiKeyScanner interface {
	Scan(dest ...any) error
}

func scanAPIKeyRow(scanner apiKeyScanner, dest ...any) error {
	return scanner.Scan(dest...)
}

func scanAPIKeyFields(scanner apiKeyScanner) (*APIKey, error) {
	var (
		k                  APIKey
		modelLimitsEnabled int
		keyEncrypted       sql.NullString
	)
	err := scanAPIKeyRow(
		scanner,
		&k.ID, &k.KeyHash, &k.KeyHint, &keyEncrypted, &k.KeyType, &k.Name,
		&k.UserID,
		&k.DailyTokenLimit, &k.MonthlyTokenLimit,
		&k.RequestsPerMinute, &k.RequestsPerDay,
		&k.Status, &k.SpendCap, &k.SpendUsed,
		&k.IPWhitelist, &k.IPBlacklist,
		&modelLimitsEnabled, &k.ModelLimits,
		&k.ExpiresAt, &k.LastUsedAt,
		&k.CreatedBy, &k.CreatedAt, &k.UpdatedAt, &k.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	if keyEncrypted.Valid && strings.TrimSpace(keyEncrypted.String) != "" {
		v := keyEncrypted.String
		k.KeyEncrypted = &v
	}
	k.ModelLimitsEnabled = modelLimitsEnabled == 1
	return &k, nil
}

func scanAPIKeyFromRows(rows *sql.Rows) (*APIKey, error) {
	return scanAPIKeyFields(rows)
}

// scanAPIKey scans a single API key row returned by QueryRowContext.
func scanAPIKey(row *sql.Row) (*APIKey, error) {
	return scanAPIKeyFields(row)
}

// CountUserAPIKeys returns active user_key count for a user.
func (d *DB) CountUserAPIKeys(ctx context.Context, userID string) (int, error) {
	p := d.dialect.Placeholder
	var count int
	err := d.sql.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM api_keys WHERE user_id = "+p(1)+
			" AND key_type = 'user_key' AND deleted_at IS NULL",
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user api keys: %w", translateError(err))
	}
	return count, nil
}

// IncrementKeySpend adds revenue to spend_used and returns the new total.
func (d *DB) IncrementKeySpend(ctx context.Context, keyID string, amount float64) (float64, error) {
	if amount <= 0 {
		key, err := d.GetAPIKey(ctx, keyID)
		if err != nil {
			return 0, err
		}
		return key.SpendUsed, nil
	}
	p := d.dialect.Placeholder
	query := "UPDATE api_keys SET spend_used = spend_used + " + p(1) +
		", updated_at = CURRENT_TIMESTAMP WHERE id = " + p(2) +
		" AND deleted_at IS NULL RETURNING spend_used, spend_cap, status"
	var spendUsed, spendCap float64
	var status string
	err := d.sql.QueryRowContext(ctx, query, amount, keyID).Scan(&spendUsed, &spendCap, &status)
	if err != nil {
		return 0, fmt.Errorf("increment key spend %s: %w", keyID, translateError(err))
	}
	if spendCap > 0 && spendUsed >= spendCap && status == "active" {
		_, _ = d.sql.ExecContext(ctx,
			"UPDATE api_keys SET status = 'quota_exhausted', updated_at = CURRENT_TIMESTAMP WHERE id = "+p(1),
			keyID,
		)
	}
	return spendUsed, nil
}

// RevokeAllUserAPIKeys soft-deletes all user_key rows for a user.
func (d *DB) RevokeAllUserAPIKeys(ctx context.Context, userID string) (int64, error) {
	p := d.dialect.Placeholder
	result, err := d.sql.ExecContext(ctx,
		"UPDATE api_keys SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP, status = 'disabled' "+
			"WHERE user_id = "+p(1)+" AND key_type = 'user_key' AND deleted_at IS NULL",
		userID,
	)
	if err != nil {
		return 0, fmt.Errorf("revoke all user api keys: %w", translateError(err))
	}
	return result.RowsAffected()
}
