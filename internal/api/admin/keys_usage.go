package admin

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/internal/ratelimit"
)

type keyLimitsResponse struct {
	RequestsPerMinute int     `json:"requests_per_minute"`
	RequestsPerDay    int     `json:"requests_per_day"`
	DailyTokenLimit   int64   `json:"daily_token_limit"`
	MonthlyTokenLimit int64   `json:"monthly_token_limit"`
	SpendCap          float64 `json:"spend_cap"`
	SpendUsed         float64 `json:"spend_used"`
	Status            string  `json:"status"`
}

type keyLiveUsageResponse struct {
	RequestsPerMinute int64   `json:"requests_per_minute"`
	RequestsPerDay    int64   `json:"requests_per_day"`
	DailyTokens       int64   `json:"daily_tokens"`
	MonthlyTokens     int64   `json:"monthly_tokens"`
	SpendUsed         float64 `json:"spend_used"`
}

type keyUsageSnapshotResponse struct {
	KeyID string           `json:"key_id"`
	From  string           `json:"from"`
	To    string           `json:"to"`
	Data  []usageDataPoint `json:"data"`
}

type keyLimitsSnapshotResponse struct {
	KeyID  string               `json:"key_id"`
	Limits keyLimitsResponse    `json:"limits"`
	Usage  keyLiveUsageResponse `json:"usage"`
}

// GetAPIKeyUsage handles GET /api/v1/keys/:key_id/usage.
func (h *Handler) GetAPIKeyUsage(c fiber.Ctx) error {
	keyID := c.Params("key_id")
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	apiKey, err := h.loadVisibleAPIKey(c, keyID, keyInfo)
	if err != nil {
		return err
	}

	from, to, ok := parseUsageRange(c)
	if !ok {
		return nil
	}

	filter := db.UsageFilter{KeyID: apiKey.ID}
	aggregates, err := h.DB.GetScopedUsageAggregates(c.Context(), filter, from, to, "")
	if err != nil {
		h.Log.ErrorContext(c.Context(), "key usage: get aggregates",
			slog.String("key_id", apiKey.ID),
			slog.String("error", err.Error()),
		)
		return apierror.InternalError(c, "failed to retrieve usage data")
	}

	return c.JSON(keyUsageSnapshotResponse{
		KeyID: apiKey.ID,
		From:  from.UTC().Format(time.RFC3339),
		To:    to.UTC().Format(time.RFC3339),
		Data:  aggregatesToDataPoints(aggregates),
	})
}

// GetAPIKeyLimits handles GET /api/v1/keys/:key_id/limits.
func (h *Handler) GetAPIKeyLimits(c fiber.Ctx) error {
	keyID := c.Params("key_id")
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	apiKey, err := h.loadVisibleAPIKey(c, keyID, keyInfo)
	if err != nil {
		return err
	}

	usage := keyLiveUsageResponse{SpendUsed: apiKey.SpendUsed}
	if rl, ok := h.rateLimiter(); ok {
		snap := rl.Snapshot(apiKey.ID)
		usage.RequestsPerMinute = snap.RequestsPerMinute
		usage.RequestsPerDay = snap.RequestsPerDay
	}
	if h.TokenCounter != nil {
		snap := h.TokenCounter.Snapshot(apiKey.ID)
		usage.DailyTokens = snap.DailyTokens
		usage.MonthlyTokens = snap.MonthlyTokens
	}

	status := apiKey.Status
	if status == "" {
		status = "active"
	}

	return c.JSON(keyLimitsSnapshotResponse{
		KeyID: apiKey.ID,
		Limits: keyLimitsResponse{
			RequestsPerMinute: apiKey.RequestsPerMinute,
			RequestsPerDay:    apiKey.RequestsPerDay,
			DailyTokenLimit:   apiKey.DailyTokenLimit,
			MonthlyTokenLimit: apiKey.MonthlyTokenLimit,
			SpendCap:          apiKey.SpendCap,
			SpendUsed:         apiKey.SpendUsed,
			Status:            status,
		},
		Usage: usage,
	})
}

func (h *Handler) loadVisibleAPIKey(c fiber.Ctx, keyID string, caller *auth.KeyInfo) (*db.APIKey, error) {
	apiKey, err := h.DB.GetAPIKey(c.Context(), keyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, apierror.NotFound(c, "api key not found")
		}
		h.Log.ErrorContext(c.Context(), "get api key", slog.String("error", err.Error()))
		return nil, apierror.InternalError(c, "failed to get api key")
	}
	if !auth.HasRole(caller.Role, auth.RoleSystemAdmin) {
		if !apiKeyVisibleToCallerKey(apiKey, caller) {
			return nil, apierror.NotFound(c, "api key not found")
		}
	}
	return apiKey, nil
}

func (h *Handler) rateLimiter() (*ratelimit.RateLimiter, bool) {
	if h.RateLimiter == nil {
		return nil, false
	}
	rl, ok := h.RateLimiter.(*ratelimit.RateLimiter)
	return rl, ok
}