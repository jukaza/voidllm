package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const providerConnectionSelectColumns = "id, provider_id, name, auth_type, priority, is_active, " +
	"api_key_encrypted, test_status, last_error, error_code, last_error_at, backoff_level, " +
	"locked_until, model_locks, last_used_at, consecutive_use_count, " +
	"created_at, updated_at, deleted_at"

// ProviderConnection is one API key or auth account under a business provider.
type ProviderConnection struct {
	ID                  string
	ProviderID          string
	Name                string
	AuthType            string
	Priority            int
	IsActive            bool
	APIKeyEncrypted     *string
	TestStatus          string
	LastError           string
	ErrorCode           *int
	LastErrorAt         *string
	BackoffLevel        int
	LockedUntil         *string
	ModelLocks          map[string]string
	LastUsedAt          *string
	ConsecutiveUseCount int
	CreatedAt           string
	UpdatedAt           string
	DeletedAt           *string
}

// CreateProviderConnectionParams holds input for creating a connection.
type CreateProviderConnectionParams struct {
	ProviderID      string
	Name            string
	AuthType        string
	Priority        int
	IsActive        bool
	APIKeyEncrypted *string
	TestStatus      string
}

// UpdateProviderConnectionParams holds optional fields for updating a connection.
type UpdateProviderConnectionParams struct {
	Name                *string
	AuthType            *string
	Priority            *int
	IsActive            *bool
	APIKeyEncrypted     *string
	ClearAPIKey         bool
	TestStatus          *string
	LastError           *string
	ErrorCode           *int
	ClearErrorCode      bool
	LastErrorAt         *string
	ClearLastErrorAt    bool
	BackoffLevel        *int
	LockedUntil         *string
	ClearLockedUntil    bool
	ModelLocks          map[string]string
	ClearModelLocks     bool
	LastUsedAt          *string
	ConsecutiveUseCount *int
}

// CreateProviderConnection inserts a new key/account for a provider.
func (d *DB) CreateProviderConnection(ctx context.Context, params CreateProviderConnectionParams) (*ProviderConnection, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("create provider connection: generate id: %w", err)
	}

	authType := params.AuthType
	if authType == "" {
		authType = "apikey"
	}
	testStatus := params.TestStatus
	if testStatus == "" {
		testStatus = "unknown"
	}
	priority := params.Priority
	if priority <= 0 {
		priority = nextConnectionPriority(ctx, d, params.ProviderID)
	}
	// New connections default to active; use Update to disable.
	isActive := 1

	locksJSON, err := json.Marshal(map[string]string{})
	if err != nil {
		return nil, fmt.Errorf("create provider connection: marshal locks: %w", err)
	}

	p := d.dialect.Placeholder
	insertQuery := "INSERT INTO provider_connections " +
		"(id, provider_id, name, auth_type, priority, is_active, api_key_encrypted, test_status, model_locks, created_at, updated_at) " +
		"VALUES (" + p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", " + p(5) + ", " + p(6) + ", " + p(7) + ", " + p(8) + ", " + p(9) + ", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"
	selectQuery := "SELECT " + providerConnectionSelectColumns + " FROM provider_connections WHERE id = " + p(1) + " AND deleted_at IS NULL"

	var conn *ProviderConnection
	err = d.WithTx(ctx, func(q Querier) error {
		if _, execErr := q.ExecContext(ctx, insertQuery,
			id.String(), params.ProviderID, params.Name, authType, priority, isActive,
			params.APIKeyEncrypted, testStatus, string(locksJSON),
		); execErr != nil {
			return translateError(execErr)
		}
		row := q.QueryRowContext(ctx, selectQuery, id.String())
		var scanErr error
		conn, scanErr = scanProviderConnection(row)
		return scanErr
	})
	if err != nil {
		return nil, fmt.Errorf("create provider connection: %w", err)
	}
	return conn, nil
}

func nextConnectionPriority(ctx context.Context, d *DB, providerID string) int {
	p := d.dialect.Placeholder
	query := "SELECT COALESCE(MAX(priority), 0) FROM provider_connections WHERE provider_id = " + p(1) + " AND deleted_at IS NULL"
	var max int
	if err := d.sql.QueryRowContext(ctx, query, providerID).Scan(&max); err != nil {
		return 1
	}
	return max + 1
}

// GetProviderConnection returns an active connection by ID.
func (d *DB) GetProviderConnection(ctx context.Context, id string) (*ProviderConnection, error) {
	query := "SELECT " + providerConnectionSelectColumns +
		" FROM provider_connections WHERE id = " + d.dialect.Placeholder(1) + " AND deleted_at IS NULL"
	row := d.sql.QueryRowContext(ctx, query, id)
	conn, err := scanProviderConnection(row)
	if err != nil {
		return nil, fmt.Errorf("GetProviderConnection %s: %w", id, translateError(err))
	}
	return conn, nil
}

// ListProviderConnections returns connections for a provider ordered by priority.
// When activeOnly is true, inactive connections are excluded.
func (d *DB) ListProviderConnections(ctx context.Context, providerID string, activeOnly bool) ([]ProviderConnection, error) {
	p := d.dialect.Placeholder
	conditions := []string{"provider_id = " + p(1), "deleted_at IS NULL"}
	args := []any{providerID}
	if activeOnly {
		conditions = append(conditions, "is_active = 1")
	}
	query := "SELECT " + providerConnectionSelectColumns +
		" FROM provider_connections WHERE " + strings.Join(conditions, " AND ") +
		" ORDER BY priority ASC, updated_at DESC"

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListProviderConnections query: %w", err)
	}
	defer rows.Close()

	var out []ProviderConnection
	for rows.Next() {
		conn, scanErr := scanProviderConnection(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("ListProviderConnections scan: %w", scanErr)
		}
		out = append(out, *conn)
	}
	return out, rows.Err()
}

// UpdateProviderConnection applies non-nil fields and returns the updated row.
func (d *DB) UpdateProviderConnection(ctx context.Context, id string, params UpdateProviderConnectionParams) (*ProviderConnection, error) {
	p := d.dialect.Placeholder
	var sets []string
	var args []any
	argN := 1

	addSet := func(col string, val any) {
		sets = append(sets, col+" = "+p(argN))
		args = append(args, val)
		argN++
	}

	if params.Name != nil {
		addSet("name", *params.Name)
	}
	if params.AuthType != nil {
		addSet("auth_type", *params.AuthType)
	}
	if params.Priority != nil {
		addSet("priority", *params.Priority)
	}
	if params.IsActive != nil {
		if *params.IsActive {
			addSet("is_active", 1)
		} else {
			addSet("is_active", 0)
		}
	}
	if params.ClearAPIKey {
		addSet("api_key_encrypted", nil)
	} else if params.APIKeyEncrypted != nil {
		if *params.APIKeyEncrypted == "" {
			addSet("api_key_encrypted", nil)
		} else {
			addSet("api_key_encrypted", *params.APIKeyEncrypted)
		}
	}
	if params.TestStatus != nil {
		addSet("test_status", *params.TestStatus)
	}
	if params.LastError != nil {
		addSet("last_error", *params.LastError)
	}
	if params.ClearErrorCode {
		addSet("error_code", nil)
	} else if params.ErrorCode != nil {
		addSet("error_code", *params.ErrorCode)
	}
	if params.ClearLastErrorAt {
		addSet("last_error_at", nil)
	} else if params.LastErrorAt != nil {
		addSet("last_error_at", *params.LastErrorAt)
	}
	if params.BackoffLevel != nil {
		addSet("backoff_level", *params.BackoffLevel)
	}
	if params.ClearLockedUntil {
		addSet("locked_until", nil)
	} else if params.LockedUntil != nil {
		addSet("locked_until", *params.LockedUntil)
	}
	if params.ClearModelLocks {
		addSet("model_locks", "{}")
	} else if params.ModelLocks != nil {
		raw, err := json.Marshal(params.ModelLocks)
		if err != nil {
			return nil, fmt.Errorf("UpdateProviderConnection: marshal model_locks: %w", err)
		}
		addSet("model_locks", string(raw))
	}
	if params.LastUsedAt != nil {
		addSet("last_used_at", *params.LastUsedAt)
	}
	if params.ConsecutiveUseCount != nil {
		addSet("consecutive_use_count", *params.ConsecutiveUseCount)
	}

	if len(sets) == 0 {
		return d.GetProviderConnection(ctx, id)
	}

	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	query := "UPDATE provider_connections SET " + strings.Join(sets, ", ") +
		" WHERE id = " + p(argN) + " AND deleted_at IS NULL"
	args = append(args, id)

	res, err := d.sql.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("UpdateProviderConnection %s: %w", id, translateError(err))
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("UpdateProviderConnection %s: rows affected: %w", id, err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("UpdateProviderConnection %s: %w", id, ErrNotFound)
	}
	return d.GetProviderConnection(ctx, id)
}

// ClearProviderConnectionLocks removes all cooldown locks and resets error state.
func (d *DB) ClearProviderConnectionLocks(ctx context.Context, id string) (*ProviderConnection, error) {
	zero := 0
	active := "active"
	empty := ""
	return d.UpdateProviderConnection(ctx, id, UpdateProviderConnectionParams{
		ClearModelLocks:  true,
		ClearLockedUntil: true,
		ClearErrorCode:   true,
		ClearLastErrorAt: true,
		TestStatus:       &active,
		LastError:        &empty,
		BackoffLevel:     &zero,
	})
}

// ReorderProviderConnections sets priorities from the ordered ID list (1-based).
func (d *DB) ReorderProviderConnections(ctx context.Context, providerID string, orderedIDs []string) error {
	return d.WithTx(ctx, func(q Querier) error {
		p := d.dialect.Placeholder
		for i, connID := range orderedIDs {
			priority := i + 1
			query := "UPDATE provider_connections SET priority = " + p(1) + ", updated_at = CURRENT_TIMESTAMP" +
				" WHERE id = " + p(2) + " AND provider_id = " + p(3) + " AND deleted_at IS NULL"
			res, err := q.ExecContext(ctx, query, priority, connID, providerID)
			if err != nil {
				return translateError(err)
			}
			affected, err := res.RowsAffected()
			if err != nil {
				return err
			}
			if affected == 0 {
				return fmt.Errorf("reorder connection %s: %w", connID, ErrNotFound)
			}
		}
		return nil
	})
}

// DeleteProviderConnection soft-deletes a connection.
func (d *DB) DeleteProviderConnection(ctx context.Context, id string) error {
	p := d.dialect.Placeholder
	query := "UPDATE provider_connections SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP" +
		" WHERE id = " + p(1) + " AND deleted_at IS NULL"
	res, err := d.sql.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("DeleteProviderConnection %s: %w", id, translateError(err))
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteProviderConnection %s: rows affected: %w", id, err)
	}
	if affected == 0 {
		return fmt.Errorf("DeleteProviderConnection %s: %w", id, ErrNotFound)
	}
	return nil
}

// SetProviderConnectionModelLock records a per-upstream-model cooldown expiry.
func (d *DB) SetProviderConnectionModelLock(ctx context.Context, id, upstreamModel string, until time.Time) (*ProviderConnection, error) {
	conn, err := d.GetProviderConnection(ctx, id)
	if err != nil {
		return nil, err
	}
	locks := conn.ModelLocks
	if locks == nil {
		locks = make(map[string]string)
	}
	locks[upstreamModel] = until.UTC().Format(time.RFC3339)
	unavailable := "unavailable"
	return d.UpdateProviderConnection(ctx, id, UpdateProviderConnectionParams{
		ModelLocks: locks,
		TestStatus: &unavailable,
	})
}

func scanProviderConnection(scanner interface{ Scan(...any) error }) (*ProviderConnection, error) {
	var c ProviderConnection
	var isActive int
	var locksRaw string
	var errorCode sql.NullInt64
	var lastErrorAt, lockedUntil, lastUsedAt sql.NullString

	err := scanner.Scan(
		&c.ID, &c.ProviderID, &c.Name, &c.AuthType, &c.Priority, &isActive,
		&c.APIKeyEncrypted, &c.TestStatus, &c.LastError, &errorCode, &lastErrorAt,
		&c.BackoffLevel, &lockedUntil, &locksRaw, &lastUsedAt, &c.ConsecutiveUseCount,
		&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	if err != nil {
		return nil, err
	}

	c.IsActive = isActive == 1
	if errorCode.Valid {
		v := int(errorCode.Int64)
		c.ErrorCode = &v
	}
	if lastErrorAt.Valid {
		s := lastErrorAt.String
		c.LastErrorAt = &s
	}
	if lockedUntil.Valid {
		s := lockedUntil.String
		c.LockedUntil = &s
	}
	if lastUsedAt.Valid {
		s := lastUsedAt.String
		c.LastUsedAt = &s
	}

	c.ModelLocks = map[string]string{}
	if locksRaw != "" && locksRaw != "{}" {
		_ = json.Unmarshal([]byte(locksRaw), &c.ModelLocks)
	}
	if c.ModelLocks == nil {
		c.ModelLocks = map[string]string{}
	}
	return &c, nil
}