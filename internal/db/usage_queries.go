package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func coalesceSelectCol(groupCol string) string {
	switch groupCol {
	case "key_id", "user_id":
		return "COALESCE(" + groupCol + ", '')"
	default:
		return groupCol
	}
}

// UsageAggregate holds aggregated usage metrics for a single group key.
type UsageAggregate struct {
	GroupKey         string
	GroupLabel       string
	TotalRequests    int64
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CachedTokens     int64
	Revenue          float64
	CostEstimate     float64
	AvgDurationMS    float64
}

// UsageFilter optionally scopes usage queries to a user and/or API key.
type UsageFilter struct {
	UserID string
	KeyID  string
}

// UsageScope identifies which column to filter on for token-usage lookups.
type UsageScope int

const (
	ScopeKey UsageScope = iota
	ScopeUser
)

// GetSystemUsageAggregates returns system-wide usage metrics within [from, to].
// groupBy accepts: "" (totals), "model", "key", "user", "day", "hour".
func (d *DB) GetSystemUsageAggregates(ctx context.Context, from, to time.Time, groupBy string) ([]UsageAggregate, error) {
	return d.queryUsageAggregates(ctx, nil, from, to, groupBy)
}

// GetScopedUsageAggregates returns usage metrics for events matching filter within [from, to].
func (d *DB) GetScopedUsageAggregates(ctx context.Context, filter UsageFilter, from, to time.Time, groupBy string) ([]UsageAggregate, error) {
	return d.queryUsageAggregates(ctx, &filter, from, to, groupBy)
}

func (d *DB) queryUsageAggregates(ctx context.Context, filter *UsageFilter, from, to time.Time, groupBy string) ([]UsageAggregate, error) {
	var groupCol string
	switch groupBy {
	case "":
	case "model":
		groupCol = "model_name"
	case "key":
		groupCol = "key_id"
	case "user":
		groupCol = "user_id"
	case "day":
		groupCol = "DATE(created_at)"
	case "hour":
		groupCol = d.dialect.HourTrunc()
	default:
		return nil, fmt.Errorf("queryUsageAggregates: invalid groupBy %q", groupBy)
	}

	fromStr := from.UTC().Format(time.RFC3339)
	toStr := to.UTC().Format(time.RFC3339)

	argN := 1
	p := d.dialect.Placeholder
	var conditions []string
	var args []any

	conditions = append(conditions, "created_at >= "+p(argN))
	args = append(args, fromStr)
	argN++

	conditions = append(conditions, "created_at <= "+p(argN))
	args = append(args, toStr)
	argN++

	if filter != nil {
		if filter.UserID != "" {
			conditions = append(conditions, "user_id = "+p(argN))
			args = append(args, filter.UserID)
			argN++
		}
		if filter.KeyID != "" {
			conditions = append(conditions, "key_id = "+p(argN))
			args = append(args, filter.KeyID)
			argN++
		}
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var query string
	if groupCol != "" {
		selectCol := coalesceSelectCol(groupCol)
		query = "SELECT " + selectCol + ", COUNT(*), " +
			"COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), " +
			"COALESCE(SUM(total_tokens), 0), COALESCE(SUM(cached_tokens), 0), COALESCE(SUM(revenue), 0), " +
			"COALESCE(SUM(cost_estimate), 0), COALESCE(AVG(request_duration_ms), 0) " +
			"FROM usage_events " + where +
			" GROUP BY " + groupCol +
			" ORDER BY " + groupCol
	} else {
		query = "SELECT '' AS group_key, COUNT(*), " +
			"COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), " +
			"COALESCE(SUM(total_tokens), 0), COALESCE(SUM(cached_tokens), 0), COALESCE(SUM(revenue), 0), " +
			"COALESCE(SUM(cost_estimate), 0), COALESCE(AVG(request_duration_ms), 0) " +
			"FROM usage_events " + where
	}

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("queryUsageAggregates: %w", err)
	}
	defer rows.Close()

	var results []UsageAggregate
	for rows.Next() {
		var a UsageAggregate
		if err := rows.Scan(
			&a.GroupKey,
			&a.TotalRequests,
			&a.PromptTokens,
			&a.CompletionTokens,
			&a.TotalTokens,
			&a.CachedTokens,
			&a.Revenue,
			&a.CostEstimate,
			&a.AvgDurationMS,
		); err != nil {
			return nil, fmt.Errorf("queryUsageAggregates scan: %w", err)
		}
		results = append(results, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queryUsageAggregates rows: %w", err)
	}
	return results, nil
}

// QueryUsageSeed implements ratelimit.UsageSeeder.
func (d *DB) QueryUsageSeed(ctx context.Context, since time.Time) (*sql.Rows, error) {
	query := "SELECT key_id, total_tokens FROM usage_events WHERE created_at >= " + d.dialect.Placeholder(1)
	rows, err := d.sql.QueryContext(ctx, query, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("QueryUsageSeed: %w", err)
	}
	return rows, nil
}

// GetTokenUsageSince returns total tokens for the given scope/id since the provided time.
func (d *DB) GetTokenUsageSince(ctx context.Context, scope UsageScope, id string, since time.Time) (int64, error) {
	var col string
	switch scope {
	case ScopeKey:
		col = "key_id"
	case ScopeUser:
		col = "user_id"
	default:
		return 0, fmt.Errorf("GetTokenUsageSince: invalid usage scope %d", scope)
	}

	query := "SELECT COALESCE(SUM(total_tokens), 0) FROM usage_events WHERE " +
		col + " = " + d.dialect.Placeholder(1) +
		" AND created_at > " + d.dialect.Placeholder(2)

	var total int64
	row := d.sql.QueryRowContext(ctx, query, id, since.UTC().Format(time.RFC3339))
	if err := row.Scan(&total); err != nil {
		return 0, fmt.Errorf("GetTokenUsageSince %s %s: %w", col, id, err)
	}
	return total, nil
}