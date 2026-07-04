package db

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ChannelModelStats holds usage metrics for one product model on a provider channel.
type ChannelModelStats struct {
	ModelName        string  `json:"model_name"`
	TotalRequests    int64   `json:"total_requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	CachedTokens     int64   `json:"cached_tokens"`
	Revenue          float64 `json:"revenue"`
}

// ChannelUsageStats holds rolled-up metrics for one provider (all keys combined).
type ChannelUsageStats struct {
	ChannelID        string              `json:"channel_id"`
	ChannelLabel     string              `json:"channel_label"`
	Provider         string              `json:"provider,omitempty"`
	ProviderLogo     string              `json:"provider_logo,omitempty"`
	ProviderSlug     string              `json:"provider_slug,omitempty"`
	ProviderProtocol string              `json:"provider_protocol,omitempty"`
	TotalRequests    int64               `json:"total_requests"`
	PromptTokens     int64               `json:"prompt_tokens"`
	CompletionTokens int64               `json:"completion_tokens"`
	TotalTokens      int64               `json:"total_tokens"`
	CachedTokens     int64               `json:"cached_tokens"`
	Revenue          float64             `json:"revenue"`
	Models           []ChannelModelStats `json:"models"`
}

// ChannelUsageTotals is the system-wide summary for a channel usage query.
type ChannelUsageTotals struct {
	TotalRequests    int64   `json:"total_requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	CachedTokens     int64   `json:"cached_tokens"`
	Revenue          float64 `json:"revenue"`
}

// GetChannelUsageStats returns per-provider usage with nested per-model breakdown.
func (d *DB) GetChannelUsageStats(ctx context.Context, from, to time.Time) ([]ChannelUsageStats, ChannelUsageTotals, error) {
	fromStr := from.UTC().Format(time.RFC3339)
	toStr := to.UTC().Format(time.RFC3339)
	p := d.dialect.Placeholder

	query := "SELECT provider_id, model_name, COUNT(*), " +
		"COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), " +
		"COALESCE(SUM(total_tokens), 0), COALESCE(SUM(cached_tokens), 0), " +
		"COALESCE(SUM(revenue), 0) " +
		"FROM (" +
		"SELECT COALESCE(pc.provider_id, md.provider_id) AS provider_id, ue.model_name, " +
		"ue.prompt_tokens, ue.completion_tokens, ue.total_tokens, ue.cached_tokens, ue.revenue " +
		"FROM usage_events ue " +
		"LEFT JOIN provider_connections pc ON pc.id = ue.deployment_id AND pc.deleted_at IS NULL " +
		"LEFT JOIN model_deployments md ON md.id = ue.deployment_id AND md.deleted_at IS NULL " +
		"WHERE " + d.dialect.TimestampGTE("ue.created_at", p(1)) +
		" AND " + d.dialect.TimestampLTE("ue.created_at", p(2)) +
		" AND ue.deployment_id IS NOT NULL AND ue.deployment_id != '' " +
		" AND (pc.provider_id IS NOT NULL OR md.provider_id IS NOT NULL)" +
		") sub GROUP BY provider_id, model_name " +
		"ORDER BY provider_id, COALESCE(SUM(revenue), 0) DESC"

	rows, err := d.sql.QueryContext(ctx, query, fromStr, toStr)
	if err != nil {
		return nil, ChannelUsageTotals{}, fmt.Errorf("GetChannelUsageStats: %w", err)
	}
	defer rows.Close()

	byProvider := make(map[string]*ChannelUsageStats)
	var totals ChannelUsageTotals

	for rows.Next() {
		var providerID, modelName string
		var m ChannelModelStats
		if err := rows.Scan(
			&providerID, &modelName,
			&m.TotalRequests, &m.PromptTokens, &m.CompletionTokens,
			&m.TotalTokens, &m.CachedTokens, &m.Revenue,
		); err != nil {
			return nil, ChannelUsageTotals{}, fmt.Errorf("GetChannelUsageStats scan: %w", err)
		}
		m.ModelName = modelName

		ch, ok := byProvider[providerID]
		if !ok {
			ch = &ChannelUsageStats{ChannelID: providerID, Models: []ChannelModelStats{}}
			byProvider[providerID] = ch
		}
		ch.Models = append(ch.Models, m)
		ch.TotalRequests += m.TotalRequests
		ch.PromptTokens += m.PromptTokens
		ch.CompletionTokens += m.CompletionTokens
		ch.TotalTokens += m.TotalTokens
		ch.CachedTokens += m.CachedTokens
		ch.Revenue += m.Revenue

		totals.TotalRequests += m.TotalRequests
		totals.PromptTokens += m.PromptTokens
		totals.CompletionTokens += m.CompletionTokens
		totals.TotalTokens += m.TotalTokens
		totals.CachedTokens += m.CachedTokens
		totals.Revenue += m.Revenue
	}
	if err := rows.Err(); err != nil {
		return nil, ChannelUsageTotals{}, fmt.Errorf("GetChannelUsageStats rows: %w", err)
	}

	providerIDs := make([]string, 0, len(byProvider))
	for id := range byProvider {
		providerIDs = append(providerIDs, id)
	}
	meta, metaErr := d.resolveProviderChannelMeta(ctx, providerIDs)
	if metaErr != nil {
		return nil, ChannelUsageTotals{}, metaErr
	}

	result := make([]ChannelUsageStats, 0, len(byProvider))
	for _, ch := range byProvider {
		if m, ok := meta[ch.ChannelID]; ok {
			ch.ChannelLabel = m.Name
			ch.Provider = m.Name
			ch.ProviderLogo = m.Logo
			ch.ProviderSlug = m.Slug
			ch.ProviderProtocol = m.Protocol
		} else {
			ch.ChannelLabel = ch.ChannelID
		}
		result = append(result, *ch)
	}

	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Revenue > result[i].Revenue {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, totals, nil
}

type providerChannelMeta struct {
	Name     string
	Logo     string
	Slug     string
	Protocol string
}

func (d *DB) resolveProviderChannelMeta(ctx context.Context, ids []string) (map[string]providerChannelMeta, error) {
	out := make(map[string]providerChannelMeta)
	filtered := deduplicateIDs(ids)
	if len(filtered) == 0 {
		return out, nil
	}

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

		query := "SELECT id, name, COALESCE(logo, ''), COALESCE(slug, ''), COALESCE(protocol, '') " +
			"FROM providers WHERE id IN (" + strings.Join(placeholders, ", ") + ")"
		rows, err := d.sql.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("resolveProviderChannelMeta: %w", err)
		}

		for rows.Next() {
			var id, name, logo, slug, protocol string
			if err := rows.Scan(&id, &name, &logo, &slug, &protocol); err != nil {
				rows.Close()
				return nil, fmt.Errorf("resolveProviderChannelMeta scan: %w", err)
			}
			out[id] = providerChannelMeta{Name: name, Logo: logo, Slug: slug, Protocol: protocol}
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// ResolveDeploymentLabels maps deployment/connection IDs to display labels and metadata.
func (d *DB) ResolveDeploymentLabels(ctx context.Context, ids []string) (labels, types, providers map[string]string, err error) {
	labels = make(map[string]string)
	types = make(map[string]string)
	providers = make(map[string]string)

	filtered := deduplicateIDs(ids)
	if len(filtered) == 0 {
		return labels, types, providers, nil
	}

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
		inList := strings.Join(placeholders, ", ")

		depQuery := "SELECT md.id, m.name || '/' || md.name, COALESCE(pr.name, md.provider, '') " +
			"FROM model_deployments md " +
			"JOIN models m ON m.id = md.model_id " +
			"LEFT JOIN providers pr ON pr.id = md.provider_id " +
			"WHERE md.id IN (" + inList + ")"
		if err := d.queryDeploymentLabels(ctx, depQuery, args, labels, types, providers, "deployment"); err != nil {
			return nil, nil, nil, err
		}

		connQuery := "SELECT pc.id, COALESCE(p.name, pc.provider_id), COALESCE(p.name, '') " +
			"FROM provider_connections pc " +
			"LEFT JOIN providers p ON p.id = pc.provider_id " +
			"WHERE pc.id IN (" + inList + ")"
		if err := d.queryDeploymentLabels(ctx, connQuery, args, labels, types, providers, "connection"); err != nil {
			return nil, nil, nil, err
		}
	}

	for _, id := range filtered {
		if _, ok := labels[id]; !ok {
			labels[id] = id
			types[id] = "unknown"
		}
	}

	return labels, types, providers, nil
}

func (d *DB) queryDeploymentLabels(
	ctx context.Context,
	query string,
	args []any,
	labels, types, providers map[string]string,
	channelType string,
) error {
	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("queryDeploymentLabels: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, label, provider string
		if err := rows.Scan(&id, &label, &provider); err != nil {
			return fmt.Errorf("queryDeploymentLabels scan: %w", err)
		}
		labels[id] = label
		types[id] = channelType
		if provider != "" {
			providers[id] = provider
		}
	}
	return rows.Err()
}