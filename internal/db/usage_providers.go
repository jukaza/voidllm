package db

import (
	"context"
	"fmt"
	"time"
)

// ProviderUsageTotals is the system-wide summary for provider usage queries.
type ProviderUsageTotals struct {
	TotalRequests    int64   `json:"total_requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	CachedTokens     int64   `json:"cached_tokens"`
	Revenue          float64 `json:"revenue"`
}

// ProviderUsageStats holds usage metrics attributed to one business provider.
type ProviderUsageStats struct {
	ProviderID       string  `json:"provider_id"`
	TotalRequests    int64   `json:"total_requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	CachedTokens     int64   `json:"cached_tokens"`
	Revenue          float64 `json:"revenue"`
}

// GetProviderUsageStats aggregates usage_events by providers table ID, mapping
// deployment_id to provider_connections or model_deployments.
func (d *DB) GetProviderUsageStats(ctx context.Context, from, to time.Time) ([]ProviderUsageStats, ProviderUsageTotals, error) {
	fromStr := from.UTC().Format(time.RFC3339)
	toStr := to.UTC().Format(time.RFC3339)
	p := d.dialect.Placeholder

	query := "SELECT provider_id, COUNT(*), " +
		"COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), " +
		"COALESCE(SUM(total_tokens), 0), COALESCE(SUM(cached_tokens), 0), " +
		"COALESCE(SUM(revenue), 0) " +
		"FROM (" +
		"SELECT COALESCE(pc.provider_id, md.provider_id) AS provider_id, " +
		"ue.prompt_tokens, ue.completion_tokens, ue.total_tokens, ue.cached_tokens, ue.revenue " +
		"FROM usage_events ue " +
		"LEFT JOIN provider_connections pc ON pc.id = ue.deployment_id AND pc.deleted_at IS NULL " +
		"LEFT JOIN model_deployments md ON md.id = ue.deployment_id AND md.deleted_at IS NULL " +
		"WHERE " + d.dialect.TimestampGTE("ue.created_at", p(1)) +
		" AND " + d.dialect.TimestampLTE("ue.created_at", p(2)) +
		" AND ue.deployment_id IS NOT NULL AND ue.deployment_id != '' " +
		" AND (pc.provider_id IS NOT NULL OR md.provider_id IS NOT NULL)" +
		") sub GROUP BY provider_id ORDER BY COALESCE(SUM(revenue), 0) DESC"

	rows, err := d.sql.QueryContext(ctx, query, fromStr, toStr)
	if err != nil {
		return nil, ProviderUsageTotals{}, fmt.Errorf("GetProviderUsageStats: %w", err)
	}
	defer rows.Close()

	var result []ProviderUsageStats
	var totals ProviderUsageTotals

	for rows.Next() {
		var s ProviderUsageStats
		if err := rows.Scan(
			&s.ProviderID, &s.TotalRequests,
			&s.PromptTokens, &s.CompletionTokens, &s.TotalTokens, &s.CachedTokens,
			&s.Revenue,
		); err != nil {
			return nil, ProviderUsageTotals{}, fmt.Errorf("GetProviderUsageStats scan: %w", err)
		}
		result = append(result, s)
		totals.TotalRequests += s.TotalRequests
		totals.PromptTokens += s.PromptTokens
		totals.CompletionTokens += s.CompletionTokens
		totals.TotalTokens += s.TotalTokens
		totals.CachedTokens += s.CachedTokens
		totals.Revenue += s.Revenue
	}
	if err := rows.Err(); err != nil {
		return nil, ProviderUsageTotals{}, fmt.Errorf("GetProviderUsageStats rows: %w", err)
	}
	return result, totals, nil
}