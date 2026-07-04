package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const providerUpstreamModelSelectColumns = "id, provider_id, upstream_id, display_name, is_enabled, " +
	"cost_input_per_1m, cost_output_per_1m, metadata, created_at, updated_at"

// ProviderUpstreamModel is one upstream model ID in a provider's inventory.
type ProviderUpstreamModel struct {
	ID              string
	ProviderID      string
	UpstreamID      string
	DisplayName     string
	IsEnabled       bool
	CostInputPer1M  *float64
	CostOutputPer1M *float64
	Metadata        string
	CreatedAt       string
	UpdatedAt       string
}

// UpsertProviderUpstreamModelParams holds input for create-or-update by (provider, upstream_id).
type UpsertProviderUpstreamModelParams struct {
	ProviderID      string
	UpstreamID      string
	DisplayName     string
	IsEnabled       bool
	CostInputPer1M  *float64
	CostOutputPer1M *float64
	Metadata        string
}

// UpdateProviderUpstreamModelParams holds optional update fields.
type UpdateProviderUpstreamModelParams struct {
	DisplayName     *string
	IsEnabled       *bool
	CostInputPer1M  *float64
	CostOutputPer1M *float64
	Metadata        *string
}

// UpsertProviderUpstreamModel inserts or updates an upstream model row.
func (d *DB) UpsertProviderUpstreamModel(ctx context.Context, params UpsertProviderUpstreamModelParams) (*ProviderUpstreamModel, error) {
	existing, err := d.GetProviderUpstreamModelByUpstreamID(ctx, params.ProviderID, params.UpstreamID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		displayName := params.DisplayName
		if displayName == "" {
			displayName = existing.DisplayName
		}
		metadata := params.Metadata
		if metadata == "" {
			metadata = existing.Metadata
		}
		return d.UpdateProviderUpstreamModel(ctx, existing.ID, UpdateProviderUpstreamModelParams{
			DisplayName:     &displayName,
			CostInputPer1M:  params.CostInputPer1M,
			CostOutputPer1M: params.CostOutputPer1M,
			Metadata:        &metadata,
		})
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("upsert provider upstream model: generate id: %w", err)
	}

	displayName := params.DisplayName
	if displayName == "" {
		displayName = params.UpstreamID
	}
	metadata := params.Metadata
	if metadata == "" {
		metadata = "{}"
	}
	// New inventory rows default to enabled.
	isEnabled := 1

	p := d.dialect.Placeholder
	insertQuery := "INSERT INTO provider_upstream_models " +
		"(id, provider_id, upstream_id, display_name, is_enabled, cost_input_per_1m, cost_output_per_1m, metadata, created_at, updated_at) " +
		"VALUES (" + p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", " + p(5) + ", " + p(6) + ", " + p(7) + ", " + p(8) + ", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"
	selectQuery := "SELECT " + providerUpstreamModelSelectColumns + " FROM provider_upstream_models WHERE id = " + p(1)

	var row *ProviderUpstreamModel
	err = d.WithTx(ctx, func(q Querier) error {
		if _, execErr := q.ExecContext(ctx, insertQuery,
			id.String(), params.ProviderID, params.UpstreamID, displayName, isEnabled,
			params.CostInputPer1M, params.CostOutputPer1M, metadata,
		); execErr != nil {
			return translateError(execErr)
		}
		r := q.QueryRowContext(ctx, selectQuery, id.String())
		var scanErr error
		row, scanErr = scanProviderUpstreamModel(r)
		return scanErr
	})
	if err != nil {
		return nil, fmt.Errorf("upsert provider upstream model: %w", err)
	}
	return row, nil
}

// GetProviderUpstreamModel returns a row by primary key.
func (d *DB) GetProviderUpstreamModel(ctx context.Context, id string) (*ProviderUpstreamModel, error) {
	query := "SELECT " + providerUpstreamModelSelectColumns +
		" FROM provider_upstream_models WHERE id = " + d.dialect.Placeholder(1)
	row := d.sql.QueryRowContext(ctx, query, id)
	m, err := scanProviderUpstreamModel(row)
	if err != nil {
		return nil, fmt.Errorf("GetProviderUpstreamModel %s: %w", id, translateError(err))
	}
	return m, nil
}

// GetProviderUpstreamModelByUpstreamID looks up by provider + upstream model name.
func (d *DB) GetProviderUpstreamModelByUpstreamID(ctx context.Context, providerID, upstreamID string) (*ProviderUpstreamModel, error) {
	query := "SELECT " + providerUpstreamModelSelectColumns +
		" FROM provider_upstream_models WHERE provider_id = " + d.dialect.Placeholder(1) +
		" AND upstream_id = " + d.dialect.Placeholder(2)
	row := d.sql.QueryRowContext(ctx, query, providerID, upstreamID)
	m, err := scanProviderUpstreamModel(row)
	if err != nil {
		return nil, fmt.Errorf("GetProviderUpstreamModelByUpstreamID %s/%s: %w", providerID, upstreamID, translateError(err))
	}
	return m, nil
}

// ListProviderUpstreamModels returns inventory rows for a provider.
func (d *DB) ListProviderUpstreamModels(ctx context.Context, providerID string, enabledOnly bool) ([]ProviderUpstreamModel, error) {
	p := d.dialect.Placeholder
	conditions := []string{"provider_id = " + p(1)}
	args := []any{providerID}
	if enabledOnly {
		conditions = append(conditions, "is_enabled = 1")
	}
	query := "SELECT " + providerUpstreamModelSelectColumns +
		" FROM provider_upstream_models WHERE " + strings.Join(conditions, " AND ") +
		" ORDER BY upstream_id ASC"

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListProviderUpstreamModels query: %w", err)
	}
	defer rows.Close()

	var out []ProviderUpstreamModel
	for rows.Next() {
		m, scanErr := scanProviderUpstreamModel(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("ListProviderUpstreamModels scan: %w", scanErr)
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// ListAllProviderUpstreamModels returns inventory across active (non-deleted) providers.
func (d *DB) ListAllProviderUpstreamModels(ctx context.Context, enabledOnly bool) ([]ProviderUpstreamModel, error) {
	conditions := []string{"p.deleted_at IS NULL"}
	if enabledOnly {
		conditions = append(conditions, "m.is_enabled = 1")
	}
	where := " WHERE " + strings.Join(conditions, " AND ")
	query := "SELECT m.id, m.provider_id, m.upstream_id, m.display_name, m.is_enabled, " +
		"m.cost_input_per_1m, m.cost_output_per_1m, m.metadata, m.created_at, m.updated_at" +
		" FROM provider_upstream_models m" +
		" INNER JOIN providers p ON p.id = m.provider_id" + where +
		" ORDER BY m.provider_id ASC, m.upstream_id ASC"

	rows, err := d.sql.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ListAllProviderUpstreamModels query: %w", err)
	}
	defer rows.Close()

	var out []ProviderUpstreamModel
	for rows.Next() {
		m, scanErr := scanProviderUpstreamModel(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("ListAllProviderUpstreamModels scan: %w", scanErr)
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// UpdateProviderUpstreamModel applies non-nil fields.
func (d *DB) UpdateProviderUpstreamModel(ctx context.Context, id string, params UpdateProviderUpstreamModelParams) (*ProviderUpstreamModel, error) {
	p := d.dialect.Placeholder
	var sets []string
	var args []any
	argN := 1

	addSet := func(col string, val any) {
		sets = append(sets, col+" = "+p(argN))
		args = append(args, val)
		argN++
	}

	if params.DisplayName != nil {
		addSet("display_name", *params.DisplayName)
	}
	if params.IsEnabled != nil {
		if *params.IsEnabled {
			addSet("is_enabled", 1)
		} else {
			addSet("is_enabled", 0)
		}
	}
	if params.CostInputPer1M != nil {
		addSet("cost_input_per_1m", *params.CostInputPer1M)
	}
	if params.CostOutputPer1M != nil {
		addSet("cost_output_per_1m", *params.CostOutputPer1M)
	}
	if params.Metadata != nil {
		addSet("metadata", *params.Metadata)
	}

	if len(sets) == 0 {
		return d.GetProviderUpstreamModel(ctx, id)
	}

	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	query := "UPDATE provider_upstream_models SET " + strings.Join(sets, ", ") +
		" WHERE id = " + p(argN)
	args = append(args, id)

	res, err := d.sql.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("UpdateProviderUpstreamModel %s: %w", id, translateError(err))
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("UpdateProviderUpstreamModel %s: rows affected: %w", id, err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("UpdateProviderUpstreamModel %s: %w", id, ErrNotFound)
	}
	return d.GetProviderUpstreamModel(ctx, id)
}

// DeleteProviderUpstreamModel removes an inventory row.
func (d *DB) DeleteProviderUpstreamModel(ctx context.Context, id string) error {
	p := d.dialect.Placeholder
	query := "DELETE FROM provider_upstream_models WHERE id = " + p(1)
	res, err := d.sql.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("DeleteProviderUpstreamModel %s: %w", id, translateError(err))
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteProviderUpstreamModel %s: rows affected: %w", id, err)
	}
	if affected == 0 {
		return fmt.Errorf("DeleteProviderUpstreamModel %s: %w", id, ErrNotFound)
	}
	return nil
}

func scanProviderUpstreamModel(scanner interface{ Scan(...any) error }) (*ProviderUpstreamModel, error) {
	var m ProviderUpstreamModel
	var isEnabled int
	err := scanner.Scan(
		&m.ID, &m.ProviderID, &m.UpstreamID, &m.DisplayName, &isEnabled,
		&m.CostInputPer1M, &m.CostOutputPer1M, &m.Metadata, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	m.IsEnabled = isEnabled == 1
	return &m, nil
}