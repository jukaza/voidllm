package admin

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
)

// usageResponse is the JSON envelope returned by MyUsage and SystemAdminUsage.
type usageResponse struct {
	From    string           `json:"from"`
	To      string           `json:"to"`
	GroupBy string           `json:"group_by,omitempty"`
	Data    []usageDataPoint `json:"data"`
}

// usageDataPoint holds aggregated metrics for one group within a usage response.
type usageDataPoint struct {
	GroupKey         string  `json:"group_key,omitempty"`
	GroupLabel       string  `json:"group_label,omitempty"`
	TotalRequests    int64   `json:"total_requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	CachedTokens     int64   `json:"cached_tokens"`
	Revenue          float64 `json:"revenue"`
	AvgDurationMS    float64 `json:"avg_duration_ms"`
}

// validGroupBy is the set of accepted group_by values for usage endpoints.
var validGroupBy = map[string]bool{
	"":           true,
	"model":      true,
	"key":        true,
	"user":       true,
	"day":        true,
	"hour":       true,
	"deployment": true,
	"provider":   true,
}

// maxUsageRangeDays is the maximum allowed time range for a usage query.
const maxUsageRangeDays = 90

// parseUsageRange parses and validates the from and to query parameters.
func parseUsageRange(c fiber.Ctx) (from, to time.Time, ok bool) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" {
		_ = apierror.BadRequest(c, "from is required")
		return
	}
	if toStr == "" {
		_ = apierror.BadRequest(c, "to is required")
		return
	}

	var err error
	from, err = time.Parse(time.RFC3339, fromStr)
	if err != nil {
		_ = apierror.BadRequest(c, "from must be a valid RFC3339 timestamp")
		return
	}
	to, err = time.Parse(time.RFC3339, toStr)
	if err != nil {
		_ = apierror.BadRequest(c, "to must be a valid RFC3339 timestamp")
		return
	}

	if !from.Before(to) {
		_ = apierror.BadRequest(c, "from must be before to")
		return
	}
	if to.Sub(from) > maxUsageRangeDays*24*time.Hour {
		_ = apierror.BadRequest(c, "time range must not exceed 90 days")
		return
	}

	ok = true
	return
}

// parseUsageParams parses and validates the from, to, and group_by query parameters.
func parseUsageParams(c fiber.Ctx) (from, to time.Time, groupBy string, ok bool) {
	from, to, ok = parseUsageRange(c)
	if !ok {
		return
	}

	groupBy = c.Query("group_by", "")
	if !validGroupBy[groupBy] {
		_ = apierror.BadRequest(c, "group_by must be one of: model, key, user, day, hour, deployment, provider")
		ok = false
		return
	}

	ok = true
	return
}

// aggregatesToDataPoints converts a slice of UsageAggregate to the JSON-serialisable slice.
func aggregatesToDataPoints(aggs []db.UsageAggregate) []usageDataPoint {
	data := make([]usageDataPoint, len(aggs))
	for i, a := range aggs {
		data[i] = usageDataPoint{
			GroupKey:         a.GroupKey,
			GroupLabel:       a.GroupLabel,
			TotalRequests:    a.TotalRequests,
			PromptTokens:     a.PromptTokens,
			CompletionTokens: a.CompletionTokens,
			TotalTokens:      a.TotalTokens,
			CachedTokens:     a.CachedTokens,
			Revenue:          a.Revenue,
			AvgDurationMS:    a.AvgDurationMS,
		}
	}
	return data
}

// parseMyUsageFilter builds the scoped usage filter for GET /usage/me.
// Optional query params key_id and model narrow results within the caller's scope.
func (h *Handler) parseMyUsageFilter(c fiber.Ctx, keyInfo *auth.KeyInfo) (db.UsageFilter, bool) {
	filter := db.UsageFilter{}
	if keyInfo.UserID != "" {
		filter.UserID = keyInfo.UserID
	} else {
		filter.KeyID = keyInfo.ID
	}

	if keyID := c.Query("key_id"); keyID != "" {
		key, err := h.DB.GetAPIKey(c.Context(), keyID)
		if err != nil {
			if err == db.ErrNotFound {
				_ = apierror.NotFound(c, "api key not found")
				return filter, false
			}
			h.Log.ErrorContext(c.Context(), "my usage: get api key", slog.String("error", err.Error()))
			_ = apierror.InternalError(c, "failed to validate api key")
			return filter, false
		}
		if keyInfo.UserID != "" {
			if key.UserID == nil || *key.UserID != keyInfo.UserID {
				_ = apierror.Send(c, fiber.StatusForbidden, "forbidden", "api key does not belong to you")
				return filter, false
			}
			filter.KeyID = keyID
		} else if keyID != keyInfo.ID {
			_ = apierror.Send(c, fiber.StatusForbidden, "forbidden", "api key does not belong to you")
			return filter, false
		}
	}

	filter.Model = c.Query("model")
	return filter, true
}

// enrichGroupLabels resolves entity IDs in the aggregates to human-readable labels.
func (h *Handler) enrichGroupLabels(ctx context.Context, groupBy string, aggs []db.UsageAggregate) {
	ids := make([]string, 0, len(aggs))
	for _, a := range aggs {
		ids = append(ids, a.GroupKey)
	}

	labels, err := h.DB.ResolveGroupLabels(ctx, groupBy, ids)
	if err != nil {
		h.Log.WarnContext(ctx, "resolve usage group labels",
			slog.String("group_by", groupBy),
			slog.String("error", err.Error()),
		)
		return
	}

	for i := range aggs {
		if label, ok := labels[aggs[i].GroupKey]; ok {
			aggs[i].GroupLabel = label
		}
	}
}

// SystemAdminUsage handles GET /api/v1/usage.
func (h *Handler) SystemAdminUsage(c fiber.Ctx) error {
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
		h.Log.ErrorContext(c.Context(), "system admin usage: get aggregates",
			slog.String("error", err.Error()),
		)
		return apierror.InternalError(c, "failed to retrieve usage data")
	}

	h.enrichGroupLabels(c.Context(), groupBy, aggregates)

	return c.JSON(usageResponse{
		From:    from.UTC().Format(time.RFC3339),
		To:      to.UTC().Format(time.RFC3339),
		GroupBy: groupBy,
		Data:    aggregatesToDataPoints(aggregates),
	})
}

// MyUsage handles GET /api/v1/usage/me.
func (h *Handler) MyUsage(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}

	from, to, groupBy, ok := parseUsageParams(c)
	if !ok {
		return nil
	}

	filter, ok := h.parseMyUsageFilter(c, keyInfo)
	if !ok {
		return nil
	}

	aggregates, err := h.DB.GetScopedUsageAggregates(c.Context(), filter, from, to, groupBy)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "my usage: get aggregates",
			slog.String("user_id", keyInfo.UserID),
			slog.String("error", err.Error()),
		)
		return apierror.InternalError(c, "failed to retrieve usage data")
	}

	h.enrichGroupLabels(c.Context(), groupBy, aggregates)

	return c.JSON(usageResponse{
		From:    from.UTC().Format(time.RFC3339),
		To:      to.UTC().Format(time.RFC3339),
		GroupBy: groupBy,
		Data:    aggregatesToDataPoints(aggregates),
	})
}
