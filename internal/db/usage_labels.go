package db

import (
	"context"
	"fmt"
	"strings"
)

// labelResolveChunkSize is the maximum number of IDs processed per IN-list query.
// Keeping this below the SQLite bind-parameter limit (~999) avoids driver errors at
// high cardinality while still requiring very few round-trips in practice.
const labelResolveChunkSize = 500

// ResolveGroupLabels returns a map from entity ID to a human-readable display
// label for the given usage groupBy dimension. The resolvable dimensions are
// "key" and "user"; for any other dimension
// (model/day/hour/server/tool/status/"") it returns an empty, non-nil map.
// Soft-deleted rows are intentionally included so historical usage still resolves
// to a name. IDs not found are simply absent from the returned map.
//
// This is an UNSCOPED global lookup by id. Callers MUST pass only IDs already
// constrained to the caller's authorized scope — this function does not enforce
// tenant boundaries itself.
func (d *DB) ResolveGroupLabels(ctx context.Context, groupBy string, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}

	var table, labelExpr string
	switch groupBy {
	case "key":
		table = "api_keys"
		labelExpr = "CASE WHEN name != '' THEN name ELSE key_hint END"
	case "user":
		table = "users"
		labelExpr = "display_name"
	case "deployment":
		labels, _, _, err := d.ResolveDeploymentLabels(ctx, ids)
		if err != nil {
			return nil, err
		}
		return labels, nil
	default:
		// Non-resolvable dimension (model, day, hour, provider, "").
		return map[string]string{}, nil
	}

	filtered := deduplicateIDs(ids)

	if len(filtered) == 0 {
		return map[string]string{}, nil
	}

	result := make(map[string]string, len(filtered))

	for start := 0; start < len(filtered); start += labelResolveChunkSize {
		end := start + labelResolveChunkSize
		if end > len(filtered) {
			end = len(filtered)
		}
		chunk := filtered[start:end]

		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for i, id := range chunk {
			placeholders[i] = d.dialect.Placeholder(i + 1)
			args[i] = id
		}

		query := "SELECT id, " + labelExpr +
			" FROM " + table +
			" WHERE id IN (" + strings.Join(placeholders, ", ") + ")"

		if err := d.queryLabelsInto(ctx, result, "ResolveGroupLabels "+groupBy, query, args...); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// queryLabelsInto runs query with args, scanning each (id, label) row into dst.
// It closes its rows before returning so chunked callers do not accumulate open
// result sets across iterations. errCtx is prepended to any error message.
func (d *DB) queryLabelsInto(ctx context.Context, dst map[string]string, errCtx, query string, args ...any) error {
	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return fmt.Errorf("%s: %w", errCtx, err)
		}
		dst[id] = label
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}
	return nil
}

// deduplicateIDs returns a new slice with empty strings removed and duplicates
// eliminated, preserving first-occurrence order.
func deduplicateIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
