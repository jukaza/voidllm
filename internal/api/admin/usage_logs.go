package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
)

type usageLogResponse struct {
	ID                 string   `json:"id"`
	RequestID          string   `json:"request_id"`
	CreatedAt          string   `json:"created_at"`
	ModelName          string   `json:"model_name"`
	RequestedModelName string   `json:"requested_model_name,omitempty"`
	UserID             string   `json:"user_id,omitempty"`
	UserDisplayName    string   `json:"user_display_name,omitempty"`
	KeyID              string   `json:"key_id,omitempty"`
	KeyHint            string   `json:"key_hint,omitempty"`
	DeploymentID       string   `json:"deployment_id,omitempty"`
	ChannelLabel       string   `json:"channel_label,omitempty"`
	ProviderName       string   `json:"provider_name,omitempty"`
	PromptTokens       int      `json:"prompt_tokens"`
	CompletionTokens   int      `json:"completion_tokens"`
	TotalTokens        int      `json:"total_tokens"`
	CachedTokens       int      `json:"cached_tokens"`
	CacheWriteTokens   int      `json:"cache_write_tokens,omitempty"`
	Revenue            *float64 `json:"revenue,omitempty"`
	StatusCode         int      `json:"status_code"`
	DurationMS         int      `json:"duration_ms"`
	TTFTMS             *int     `json:"ttft_ms,omitempty"`
	LogType            string   `json:"log_type"`
	IsStream           bool     `json:"is_stream"`
	Meta               any      `json:"meta,omitempty"`
}

type usageLogsListResponse struct {
	From       string             `json:"from"`
	To         string             `json:"to"`
	Data       []usageLogResponse `json:"data"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type profitTotals struct {
	TotalRequests    int64   `json:"total_requests"`
	Revenue          float64 `json:"revenue"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	CachedTokens     int64   `json:"cached_tokens"`
}

type profitResponse struct {
	From    string           `json:"from"`
	To      string           `json:"to"`
	GroupBy string           `json:"group_by,omitempty"`
	Totals  profitTotals     `json:"totals"`
	Data    []usageDataPoint `json:"data"`
}

func usageLogFromRow(row db.UsageLogRow, adminView bool) usageLogResponse {
	resp := usageLogResponse{
		ID:                 row.ID,
		RequestID:          row.RequestID,
		CreatedAt:          row.CreatedAt,
		ModelName:          row.ModelName,
		RequestedModelName: row.RequestedModelName,
		PromptTokens:       row.PromptTokens,
		CompletionTokens:   row.CompletionTokens,
		TotalTokens:        row.TotalTokens,
		CachedTokens:       row.CachedTokens,
		CacheWriteTokens:   row.CacheWriteTokens,
		Revenue:            row.Revenue,
		StatusCode:         row.StatusCode,
		DurationMS:         row.DurationMS,
		TTFTMS:             row.TTFTMS,
		LogType:            row.LogType,
		IsStream:           row.IsStream,
	}
	if adminView {
		resp.UserID = row.UserID
		resp.UserDisplayName = row.UserDisplayName
		resp.KeyID = row.KeyID
		resp.KeyHint = row.KeyHint
		resp.DeploymentID = row.DeploymentID
		if row.Meta != "" && row.Meta != "{}" {
			var meta any
			if err := json.Unmarshal([]byte(row.Meta), &meta); err == nil {
				resp.Meta = meta
			}
		}
	}
	return resp
}

func (h *Handler) enrichUsageLogChannels(ctx context.Context, data []usageLogResponse) {
	ids := make([]string, 0, len(data))
	for _, row := range data {
		if row.DeploymentID != "" {
			ids = append(ids, row.DeploymentID)
		}
	}
	if len(ids) == 0 {
		return
	}

	labels, _, providers, err := h.DB.ResolveDeploymentLabels(ctx, ids)
	if err != nil {
		h.Log.WarnContext(ctx, "enrich usage log channels", slog.String("error", err.Error()))
		return
	}

	for i := range data {
		id := data[i].DeploymentID
		if id == "" {
			continue
		}
		if label, ok := labels[id]; ok {
			data[i].ChannelLabel = label
		}
		if provider, ok := providers[id]; ok {
			data[i].ProviderName = provider
		}
	}
}

func parseUsageLogsFilter(c fiber.Ctx, from, to time.Time) db.UsageLogFilter {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	status := 0
	if v := c.Query("status"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			status = n
		}
	}
	return db.UsageLogFilter{
		Model:        c.Query("model"),
		UserID:       c.Query("user_id"),
		KeyID:        c.Query("key_id"),
		DeploymentID: c.Query("deployment_id"),
		RequestID:    c.Query("request_id"),
		LogType:      c.Query("log_type"),
		StatusCode:   status,
		From:         from,
		To:           to,
		Cursor:       c.Query("cursor"),
		Limit:        limit,
	}
}

// MyUsageLogs handles GET /api/v1/usage/me/logs.
func (h *Handler) MyUsageLogs(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}

	from, to, ok := parseUsageRange(c)
	if !ok {
		return nil
	}

	filter := parseUsageLogsFilter(c, from, to)
	if keyInfo.UserID != "" {
		filter.UserID = keyInfo.UserID
	} else {
		filter.KeyID = keyInfo.ID
	}

	rows, nextCursor, err := h.DB.ListUsageLogs(c.Context(), filter)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "my usage logs", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to retrieve usage logs")
	}

	data := make([]usageLogResponse, len(rows))
	for i, row := range rows {
		data[i] = usageLogFromRow(row, false)
	}

	return c.JSON(usageLogsListResponse{
		From:       from.UTC().Format(time.RFC3339),
		To:         to.UTC().Format(time.RFC3339),
		Data:       data,
		NextCursor: nextCursor,
	})
}

// SystemUsageLogs handles GET /api/v1/usage/logs (system_admin).
func (h *Handler) SystemUsageLogs(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "system admin access required")
	}

	from, to, ok := parseUsageRange(c)
	if !ok {
		return nil
	}

	filter := parseUsageLogsFilter(c, from, to)
	rows, nextCursor, err := h.DB.ListUsageLogs(c.Context(), filter)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "system usage logs", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to retrieve usage logs")
	}

	data := make([]usageLogResponse, len(rows))
	for i, row := range rows {
		data[i] = usageLogFromRow(row, true)
	}
	h.enrichUsageLogChannels(c.Context(), data)

	return c.JSON(usageLogsListResponse{
		From:       from.UTC().Format(time.RFC3339),
		To:         to.UTC().Format(time.RFC3339),
		Data:       data,
		NextCursor: nextCursor,
	})
}

// MyUsageLogDetail handles GET /api/v1/usage/me/logs/:request_id.
func (h *Handler) MyUsageLogDetail(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}

	scope := db.UsageFilter{}
	if keyInfo.UserID != "" {
		scope.UserID = keyInfo.UserID
	} else {
		scope.KeyID = keyInfo.ID
	}

	row, err := h.DB.GetUsageLogByRequestID(c.Context(), c.Params("request_id"), scope)
	if err != nil {
		if err == db.ErrNotFound {
			return apierror.NotFound(c, "usage log not found")
		}
		h.Log.ErrorContext(c.Context(), "my usage log detail", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load usage log")
	}
	return c.JSON(usageLogFromRow(*row, false))
}

// SystemUsageLogDetail handles GET /api/v1/usage/logs/:request_id (system_admin).
func (h *Handler) SystemUsageLogDetail(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "system admin access required")
	}

	row, err := h.DB.GetUsageLogByRequestID(c.Context(), c.Params("request_id"), db.UsageFilter{})
	if err != nil {
		if err == db.ErrNotFound {
			return apierror.NotFound(c, "usage log not found")
		}
		h.Log.ErrorContext(c.Context(), "system usage log detail", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load usage log")
	}
	resp := usageLogFromRow(*row, true)
	h.enrichUsageLogChannels(c.Context(), []usageLogResponse{resp})
	return c.JSON(resp)
}

// UsageProfit handles GET /api/v1/usage/profit (system_admin).
func (h *Handler) UsageProfit(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "system admin access required")
	}

	from, to, groupBy, ok := parseUsageParams(c)
	if !ok {
		return nil
	}

	aggregates, err := h.DB.GetSystemUsageAggregates(c.Context(), from, to, groupBy)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "usage profit", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to retrieve profit data")
	}
	h.enrichGroupLabels(c.Context(), groupBy, aggregates)

	var totals profitTotals
	data := aggregatesToDataPoints(aggregates)
	for _, a := range aggregates {
		totals.TotalRequests += a.TotalRequests
		totals.PromptTokens += a.PromptTokens
		totals.CompletionTokens += a.CompletionTokens
		totals.CachedTokens += a.CachedTokens
		totals.Revenue += a.Revenue
	}

	return c.JSON(profitResponse{
		From:    from.UTC().Format(time.RFC3339),
		To:      to.UTC().Format(time.RFC3339),
		GroupBy: groupBy,
		Totals:  totals,
		Data:    data,
	})
}