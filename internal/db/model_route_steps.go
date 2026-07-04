package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const modelRouteStepSelectColumns = "id, model_id, position, provider_id, upstream_model, is_enabled, created_at"

// ModelRouteStep is one ordered hop in a product model combo chain.
type ModelRouteStep struct {
	ID             string
	ModelID        string
	Position       int
	ProviderID     string
	UpstreamModel  string
	IsEnabled      bool
	CreatedAt      string
}

// ModelRouteStepInput is one step when replacing the full route list.
type ModelRouteStepInput struct {
	ProviderID    string
	UpstreamModel string
	IsEnabled     bool
}

// ListModelRouteSteps returns ordered steps for a product model.
func (d *DB) ListModelRouteSteps(ctx context.Context, modelID string, enabledOnly bool) ([]ModelRouteStep, error) {
	p := d.dialect.Placeholder
	query := "SELECT " + modelRouteStepSelectColumns +
		" FROM model_route_steps WHERE model_id = " + p(1)
	args := []any{modelID}
	if enabledOnly {
		query += " AND is_enabled = 1"
	}
	query += " ORDER BY position ASC"

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListModelRouteSteps query: %w", err)
	}
	defer rows.Close()

	var out []ModelRouteStep
	for rows.Next() {
		step, scanErr := scanModelRouteStep(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("ListModelRouteSteps scan: %w", scanErr)
		}
		out = append(out, *step)
	}
	return out, rows.Err()
}

// ReplaceModelRouteSteps atomically replaces all steps for a model.
func (d *DB) ReplaceModelRouteSteps(ctx context.Context, modelID string, steps []ModelRouteStepInput) ([]ModelRouteStep, error) {
	var out []ModelRouteStep
	err := d.WithTx(ctx, func(q Querier) error {
		p := d.dialect.Placeholder
		if _, err := q.ExecContext(ctx, "DELETE FROM model_route_steps WHERE model_id = "+p(1), modelID); err != nil {
			return translateError(err)
		}

		for i, step := range steps {
			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("replace model route steps: generate id: %w", err)
			}
			isEnabled := 1
			if !step.IsEnabled {
				isEnabled = 0
			}
			insertQuery := "INSERT INTO model_route_steps " +
				"(id, model_id, position, provider_id, upstream_model, is_enabled, created_at) " +
				"VALUES (" + p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", " + p(5) + ", " + p(6) + ", CURRENT_TIMESTAMP)"
			if _, err := q.ExecContext(ctx, insertQuery,
				id.String(), modelID, i, step.ProviderID, step.UpstreamModel, isEnabled,
			); err != nil {
				return translateError(err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("replace model route steps: %w", err)
	}

	stepsOut, err := d.ListModelRouteSteps(ctx, modelID, false)
	if err != nil {
		return nil, err
	}
	out = stepsOut
	return out, nil
}

// CountModelRouteStepsByModelIDs returns enabled route step counts per model ID.
func (d *DB) CountModelRouteStepsByModelIDs(ctx context.Context, modelIDs []string) (map[string]int, error) {
	if len(modelIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(modelIDs))
	args := make([]any, len(modelIDs))
	for i, id := range modelIDs {
		placeholders[i] = d.dialect.Placeholder(i + 1)
		args[i] = id
	}

	query := "SELECT model_id, COUNT(*) FROM model_route_steps" +
		" WHERE model_id IN (" + strings.Join(placeholders, ",") + ")" +
		" AND is_enabled = 1 GROUP BY model_id"

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("CountModelRouteStepsByModelIDs: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int, len(modelIDs))
	for rows.Next() {
		var modelID string
		var count int
		if scanErr := rows.Scan(&modelID, &count); scanErr != nil {
			return nil, fmt.Errorf("CountModelRouteStepsByModelIDs scan: %w", scanErr)
		}
		result[modelID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("CountModelRouteStepsByModelIDs rows: %w", err)
	}
	return result, nil
}

// DeleteModelRouteSteps removes all route steps for a model.
func (d *DB) DeleteModelRouteSteps(ctx context.Context, modelID string) error {
	p := d.dialect.Placeholder
	_, err := d.sql.ExecContext(ctx, "DELETE FROM model_route_steps WHERE model_id = "+p(1), modelID)
	if err != nil {
		return fmt.Errorf("DeleteModelRouteSteps %s: %w", modelID, translateError(err))
	}
	return nil
}

func scanModelRouteStep(scanner interface{ Scan(...any) error }) (*ModelRouteStep, error) {
	var s ModelRouteStep
	var isEnabled int
	err := scanner.Scan(
		&s.ID, &s.ModelID, &s.Position, &s.ProviderID, &s.UpstreamModel, &isEnabled, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.IsEnabled = isEnabled == 1
	return &s, nil
}