package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// UsageLogRow is one per-request usage_events record for the logs API.
type UsageLogRow struct {
	ID                 string
	RequestID          string
	CreatedAt          string
	KeyID              string
	KeyHint            string
	UserID             string
	UserDisplayName    string
	ModelName          string
	RequestedModelName string
	DeploymentID       string
	PromptTokens       int
	CompletionTokens   int
	TotalTokens        int
	CachedTokens       int
	CacheWriteTokens   int
	Revenue            *float64
	StatusCode         int
	DurationMS         int
	TTFTMS             *int
	LogType            string
	IsStream           bool
	Meta               string
}

// UsageLogFilter scopes and filters per-request log queries.
type UsageLogFilter struct {
	UserID       string
	KeyID        string
	Model        string
	DeploymentID string
	RequestID    string
	LogType      string
	StatusCode   int // 0 = any
	From         time.Time
	To           time.Time
	Cursor       string
	Limit        int
}

const usageLogSelectColumns = `ue.id, ue.request_id, ue.created_at, ue.key_id, COALESCE(ak.key_hint, ''),
ue.user_id, COALESCE(u.display_name, ''), ue.model_name, ue.requested_model_name, ue.deployment_id,
ue.prompt_tokens, ue.completion_tokens, ue.total_tokens, ue.cached_tokens, ue.cache_write_tokens,
ue.revenue, ue.status_code, ue.request_duration_ms, ue.ttft_ms,
ue.log_type, ue.is_stream, ue.meta`

// ListUsageLogs returns a page of usage events newest-first with keyset pagination.
func (d *DB) ListUsageLogs(ctx context.Context, filter UsageLogFilter) ([]UsageLogRow, string, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	p := d.dialect.Placeholder
	argN := 1
	var conditions []string
	var args []any

	fromStr := filter.From.UTC().Format(time.RFC3339)
	toStr := filter.To.UTC().Format(time.RFC3339)
	conditions = append(conditions, d.dialect.TimestampGTE("ue.created_at", p(argN)))
	args = append(args, fromStr)
	argN++
	conditions = append(conditions, d.dialect.TimestampLTE("ue.created_at", p(argN)))
	args = append(args, toStr)
	argN++

	if filter.UserID != "" {
		conditions = append(conditions, "ue.user_id = "+p(argN))
		args = append(args, filter.UserID)
		argN++
	}
	if filter.KeyID != "" {
		conditions = append(conditions, "ue.key_id = "+p(argN))
		args = append(args, filter.KeyID)
		argN++
	}
	if filter.Model != "" {
		conditions = append(conditions, "ue.model_name = "+p(argN))
		args = append(args, filter.Model)
		argN++
	}
	if filter.DeploymentID != "" {
		conditions = append(conditions, "ue.deployment_id = "+p(argN))
		args = append(args, filter.DeploymentID)
		argN++
	}
	if filter.RequestID != "" {
		conditions = append(conditions, "ue.request_id = "+p(argN))
		args = append(args, filter.RequestID)
		argN++
	}
	if filter.LogType != "" {
		conditions = append(conditions, "ue.log_type = "+p(argN))
		args = append(args, filter.LogType)
		argN++
	}
	if filter.StatusCode > 0 {
		conditions = append(conditions, "ue.status_code = "+p(argN))
		args = append(args, filter.StatusCode)
		argN++
	}
	if filter.Cursor != "" {
		conditions = append(conditions, "ue.id < "+p(argN))
		args = append(args, filter.Cursor)
		argN++
	}

	query := "SELECT " + usageLogSelectColumns +
		" FROM usage_events ue" +
		" LEFT JOIN api_keys ak ON ak.id = ue.key_id" +
		" LEFT JOIN users u ON u.id = ue.user_id" +
		" WHERE " + strings.Join(conditions, " AND ") +
		" ORDER BY ue.id DESC LIMIT " + p(argN)
	args = append(args, limit)

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("ListUsageLogs query: %w", err)
	}
	defer rows.Close()

	var out []UsageLogRow
	for rows.Next() {
		row, scanErr := scanUsageLogRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("ListUsageLogs scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("ListUsageLogs rows: %w", err)
	}

	nextCursor := ""
	if len(out) == limit {
		nextCursor = out[len(out)-1].ID
	}
	return out, nextCursor, nil
}

// GetUsageLogByRequestID returns one log row for drawer/detail views.
func (d *DB) GetUsageLogByRequestID(ctx context.Context, requestID string, filter UsageFilter) (*UsageLogRow, error) {
	p := d.dialect.Placeholder
	argN := 1
	var conditions []string
	var args []any

	conditions = append(conditions, "ue.request_id = "+p(argN))
	args = append(args, requestID)
	argN++

	if filter.UserID != "" {
		conditions = append(conditions, "ue.user_id = "+p(argN))
		args = append(args, filter.UserID)
		argN++
	}
	if filter.KeyID != "" {
		conditions = append(conditions, "ue.key_id = "+p(argN))
		args = append(args, filter.KeyID)
		argN++
	}

	query := "SELECT " + usageLogSelectColumns +
		" FROM usage_events ue" +
		" LEFT JOIN api_keys ak ON ak.id = ue.key_id" +
		" LEFT JOIN users u ON u.id = ue.user_id" +
		" WHERE " + strings.Join(conditions, " AND ") +
		" ORDER BY ue.id DESC LIMIT 1"

	row := d.sql.QueryRowContext(ctx, query, args...)
	logRow, err := scanUsageLogRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetUsageLogByRequestID %s: %w", requestID, err)
	}
	return &logRow, nil
}

type usageLogScanner interface {
	Scan(dest ...any) error
}

func scanUsageLogRow(row usageLogScanner) (UsageLogRow, error) {
	var r UsageLogRow
	var userID, userName, reqModel, deploymentID sql.NullString
	var revenue sql.NullFloat64
	var ttft sql.NullInt64
	var isStream int
	var meta sql.NullString

	err := row.Scan(
		&r.ID, &r.RequestID, &r.CreatedAt, &r.KeyID, &r.KeyHint,
		&userID, &userName, &r.ModelName, &reqModel, &deploymentID,
		&r.PromptTokens, &r.CompletionTokens, &r.TotalTokens, &r.CachedTokens, &r.CacheWriteTokens,
		&revenue, &r.StatusCode, &r.DurationMS, &ttft,
		&r.LogType, &isStream, &meta,
	)
	if err != nil {
		return r, err
	}
	if userID.Valid {
		r.UserID = userID.String
	}
	if userName.Valid {
		r.UserDisplayName = userName.String
	}
	if reqModel.Valid {
		r.RequestedModelName = reqModel.String
	}
	if deploymentID.Valid {
		r.DeploymentID = deploymentID.String
	}
	if revenue.Valid {
		v := revenue.Float64
		r.Revenue = &v
	}
	if ttft.Valid {
		v := int(ttft.Int64)
		r.TTFTMS = &v
	}
	r.IsStream = isStream != 0
	if meta.Valid {
		r.Meta = meta.String
	} else {
		r.Meta = "{}"
	}
	return r, nil
}