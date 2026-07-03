package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// modelAliasSelectColumns is the ordered column list used in all model_aliases SELECT queries.
// It must match the scan order in scanModelAlias exactly.
const modelAliasSelectColumns = "id, alias, model_name, created_by, created_at, updated_at"

// ModelAlias represents a model alias record in the database.
type ModelAlias struct {
	ID        string
	Alias     string
	ModelName string
	CreatedBy string
	CreatedAt string
	UpdatedAt string
}

// CreateModelAliasParams holds the input for creating a model alias.
type CreateModelAliasParams struct {
	Alias     string
	ModelName string
	CreatedBy string
}

// CreateModelAlias inserts a new model alias and returns the persisted record.
func (d *DB) CreateModelAlias(ctx context.Context, params CreateModelAliasParams) (*ModelAlias, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("create model alias: generate id: %w", err)
	}

	p := d.dialect.Placeholder
	insertQuery := "INSERT INTO model_aliases " +
		"(id, alias, model_name, created_by, created_at, updated_at) " +
		"VALUES (" +
		p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", " +
		"CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"

	selectQuery := "SELECT " + modelAliasSelectColumns +
		" FROM model_aliases WHERE id = " + p(1)

	var alias *ModelAlias
	err = d.WithTx(ctx, func(q Querier) error {
		_, execErr := q.ExecContext(ctx, insertQuery,
			id.String(),
			params.Alias,
			params.ModelName,
			params.CreatedBy,
		)
		if execErr != nil {
			return translateError(execErr)
		}

		row := q.QueryRowContext(ctx, selectQuery, id.String())
		var scanErr error
		alias, scanErr = scanModelAlias(row)
		return scanErr
	})
	if err != nil {
		return nil, fmt.Errorf("create model alias: %w", err)
	}
	return alias, nil
}

// ListModelAliases returns all aliases, ordered by alias name.
func (d *DB) ListModelAliases(ctx context.Context, scopeType, scopeID string) ([]ModelAlias, error) {
	query := "SELECT " + modelAliasSelectColumns +
		" FROM model_aliases ORDER BY alias"

	rows, err := d.sql.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list model aliases query: %w", err)
	}
	defer rows.Close()

	var aliases []ModelAlias
	for rows.Next() {
		a, scanErr := scanModelAlias(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("list model aliases scan: %w", scanErr)
		}
		aliases = append(aliases, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list model aliases rows: %w", err)
	}

	return aliases, nil
}

// DeleteModelAlias hard-deletes a model alias by its ID.
func (d *DB) DeleteModelAlias(ctx context.Context, id, scopeType, scopeID string) error {
	p := d.dialect.Placeholder
	query := "DELETE FROM model_aliases WHERE id = " + p(1)

	result, err := d.sql.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete model alias %s: %w", id, translateError(err))
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete model alias %s rows affected: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("delete model alias %s: %w", id, ErrNotFound)
	}
	return nil
}

// LoadAllModelAliases returns all model aliases grouped under empty string key.
func (d *DB) LoadAllModelAliases(ctx context.Context) (
	orgAliases map[string]map[string]string,
	teamAliases map[string]map[string]string,
	err error,
) {
	rows, err := d.sql.QueryContext(ctx, "SELECT alias, model_name FROM model_aliases")
	if err != nil {
		return nil, nil, fmt.Errorf("load all model aliases: %w", err)
	}
	defer rows.Close()

	global := make(map[string]string)
	for rows.Next() {
		var alias, modelName string
		if err := rows.Scan(&alias, &modelName); err != nil {
			return nil, nil, err
		}
		global[alias] = modelName
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	orgAliases = map[string]map[string]string{"": global}
	teamAliases = make(map[string]map[string]string)
	return orgAliases, teamAliases, nil
}

// scanModelAlias scans a single model_aliases row.
func scanModelAlias(scanner interface{ Scan(...any) error }) (*ModelAlias, error) {
	var a ModelAlias
	err := scanner.Scan(
		&a.ID, &a.Alias, &a.ModelName,
		&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
