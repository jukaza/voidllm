package admin

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
)

// dashboardStatsResponse is the JSON envelope returned by DashboardStats.
type dashboardStatsResponse struct {
	Scope           string  `json:"scope"`
	ActiveKeys      int     `json:"active_keys"`
	Requests24h     int64 `json:"requests_24h"`
	Tokens24h       int64 `json:"tokens_24h"`
	ModelsHealthy   int   `json:"models_healthy"`
	ModelsUnhealthy int     `json:"models_unhealthy"`
	ModelsDegraded  int     `json:"models_degraded"`
}

// DashboardStats handles GET /api/v1/dashboard/stats.
func (h *Handler) DashboardStats(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return apierror.InternalError(c, "failed to load dashboard stats")
	}

	ctx := c.Context()
	from := time.Now().UTC().Add(-24 * time.Hour)
	filter := db.UsageFilter{}

	var resp dashboardStatsResponse

	if auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		resp.Scope = "system"

		keys, err := h.DB.CountActiveKeys(ctx)
		if err != nil {
			h.Log.LogAttrs(ctx, slog.LevelError, "dashboard: count active keys",
				slog.String("error", err.Error()),
			)
			return apierror.InternalError(c, "failed to load dashboard stats")
		}
		resp.ActiveKeys = keys
	} else {
		resp.Scope = "user"
		filter.UserID = keyInfo.UserID

		keys, err := h.DB.CountUserKeys(ctx, keyInfo.UserID)
		if err != nil {
			h.Log.LogAttrs(ctx, slog.LevelError, "dashboard: count user keys",
				slog.String("user_id", keyInfo.UserID),
				slog.String("error", err.Error()),
			)
			return apierror.InternalError(c, "failed to load dashboard stats")
		}
		resp.ActiveKeys = keys
	}

	agg, err := h.DB.GetHourlyUsageTotals(ctx, filter, from)
	if err != nil {
		h.Log.LogAttrs(ctx, slog.LevelError, "dashboard: get scoped usage",
			slog.String("scope", resp.Scope),
			slog.String("error", err.Error()),
		)
		return apierror.InternalError(c, "failed to load dashboard stats")
	}

	resp.Requests24h = agg.TotalRequests
	resp.Tokens24h = agg.TotalTokens

	if h.HealthChecker != nil {
		for _, mh := range h.HealthChecker.GetAllHealth() {
			switch mh.Status {
			case "healthy":
				resp.ModelsHealthy++
			case "unhealthy":
				resp.ModelsUnhealthy++
			case "degraded":
				resp.ModelsDegraded++
			}
		}
	}

	return c.JSON(resp)
}
