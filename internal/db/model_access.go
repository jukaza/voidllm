package db

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
)

// GetOrgModelAccess returns empty slice (deprecated).
func (d *DB) GetOrgModelAccess(ctx context.Context, orgID string) ([]string, error) {
	return []string{}, nil
}

// GetTeamModelAccess returns empty slice (deprecated).
func (d *DB) GetTeamModelAccess(ctx context.Context, teamID string) ([]string, error) {
	return []string{}, nil
}

// GetKeyModelAccess returns the allowed model names for an API key, ordered alphabetically.
func (d *DB) GetKeyModelAccess(ctx context.Context, keyID string) ([]string, error) {
	query := "SELECT model_name FROM key_model_access WHERE key_id = " + d.dialect.Placeholder(1) + " ORDER BY model_name"

	rows, err := d.sql.QueryContext(ctx, query, keyID)
	if err != nil {
		return nil, fmt.Errorf("get key model access %s: %w", keyID, err)
	}
	defer rows.Close()

	var models []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("get key model access %s scan: %w", keyID, err)
		}
		models = append(models, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get key model access %s rows: %w", keyID, err)
	}

	return models, nil
}

// SetOrgModelAccess is deprecated.
func (d *DB) SetOrgModelAccess(ctx context.Context, orgID string, models []string) error {
	return nil
}

// SetTeamModelAccess is deprecated.
func (d *DB) SetTeamModelAccess(ctx context.Context, teamID string, models []string) error {
	return nil
}

// SetKeyModelAccess atomically replaces the model allowlist for an API key.
func (d *DB) SetKeyModelAccess(ctx context.Context, keyID string, models []string) error {
	p := d.dialect.Placeholder
	deleteQuery := "DELETE FROM key_model_access WHERE key_id = " + p(1)
	insertQuery := "INSERT INTO key_model_access (id, key_id, model_name) VALUES (" + p(1) + ", " + p(2) + ", " + p(3) + ")"

	err := d.WithTx(ctx, func(q Querier) error {
		if _, err := q.ExecContext(ctx, deleteQuery, keyID); err != nil {
			return fmt.Errorf("delete key model access: %w", err)
		}

		for _, model := range models {
			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate id: %w", err)
			}
			if _, err := q.ExecContext(ctx, insertQuery, id.String(), keyID, model); err != nil {
				return fmt.Errorf("insert key model access %q: %w", model, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("set key model access %s: %w", keyID, err)
	}

	return nil
}

// CheckModelAccess reports whether modelName is accessible for the given key.
func (d *DB) CheckModelAccess(ctx context.Context, orgID, teamID, keyID, modelName string) (bool, error) {
	keyModels, err := d.GetKeyModelAccess(ctx, keyID)
	if err != nil {
		return false, fmt.Errorf("check model access: %w", err)
	}
	if len(keyModels) > 0 && !slices.Contains(keyModels, modelName) {
		return false, nil
	}

	return true, nil
}

// LoadAllModelAccess returns all model access entries grouped by scope.
func (d *DB) LoadAllModelAccess(ctx context.Context) (
	orgAccess map[string][]string,
	teamAccess map[string][]string,
	keyAccess map[string][]string,
	err error,
) {
	orgAccess = make(map[string][]string)
	teamAccess = make(map[string][]string)
	keyAccess, err = loadAccessMap(ctx, d, "SELECT key_id, model_name FROM key_model_access")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load all model access (keys): %w", err)
	}

	return orgAccess, teamAccess, keyAccess, nil
}

// loadAccessMap executes a two-column query (id, model_name) and builds a map.
func loadAccessMap(ctx context.Context, d *DB, query string) (map[string][]string, error) {
	rows, err := d.sql.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var id, modelName string
		if err := rows.Scan(&id, &modelName); err != nil {
			return nil, err
		}
		result[id] = append(result[id], modelName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
