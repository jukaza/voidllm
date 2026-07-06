package db

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// CatalogModelStats holds aggregated usage metrics for the model catalog card.
type CatalogModelStats struct {
	RequestCount int64
	SuccessRate  float64
	AvgLatencyMs float64
	AvgTps       float64
}

// GetCatalogModelStats returns system-wide per-model usage aggregates since the
// given time. Keys are customer-facing product names (requested_model_name when
// set, otherwise model_name).
func (d *DB) GetCatalogModelStats(ctx context.Context, since time.Time, modelNames []string) (map[string]CatalogModelStats, error) {
	p := d.dialect.Placeholder
	sinceStr := since.UTC().Format(time.RFC3339)

	productCol := "COALESCE(NULLIF(requested_model_name, ''), model_name)"

	argN := 1
	conditions := []string{d.dialect.TimestampGTE("created_at", p(argN))}
	args := []any{sinceStr}
	argN++

	if len(modelNames) > 0 {
		placeholders := make([]string, len(modelNames))
		for i, name := range modelNames {
			placeholders[i] = p(argN)
			args = append(args, name)
			argN++
		}
		conditions = append(conditions, productCol+" IN ("+strings.Join(placeholders, ", ")+")")
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	query := "SELECT " + productCol + ", COUNT(*), " +
		"CASE WHEN COUNT(*) > 0 THEN " +
		"SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END) * 100.0 / COUNT(*) " +
		"ELSE 0 END, " +
		"COALESCE(AVG(CASE WHEN status_code >= 200 AND status_code < 300 THEN request_duration_ms END), 0), " +
		"COALESCE(AVG(CASE WHEN status_code >= 200 AND status_code < 300 THEN tokens_per_second END), 0) " +
		"FROM usage_events " + where +
		" GROUP BY " + productCol

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get catalog model stats: %w", err)
	}
	defer rows.Close()

	out := make(map[string]CatalogModelStats)
	for rows.Next() {
		var name string
		var stats CatalogModelStats
		if err := rows.Scan(&name, &stats.RequestCount, &stats.SuccessRate, &stats.AvgLatencyMs, &stats.AvgTps); err != nil {
			return nil, fmt.Errorf("get catalog model stats scan: %w", err)
		}
		out[name] = stats
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get catalog model stats rows: %w", err)
	}
	return out, nil
}