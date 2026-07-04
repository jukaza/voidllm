package admin

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
)

func providerUsageMap(stats []db.ProviderUsageStats) map[string]interface{} {
	out := make(map[string]interface{}, len(stats))
	for _, s := range stats {
		out[s.ProviderID] = fiber.Map{
			"revenue":        s.Revenue,
			"total_requests": s.TotalRequests,
		}
	}
	return out
}

// ProviderUsage handles GET /api/v1/providers/usage — today + all-time revenue per provider.
func (h *Handler) ProviderUsage(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "system admin access required")
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	allTimeStart := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	ctx := c.Context()

	todayStats, todayTotals, err := h.DB.GetProviderUsageStats(ctx, todayStart, now)
	if err != nil {
		h.Log.ErrorContext(ctx, "provider usage today", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to retrieve provider usage")
	}

	allStats, allTotals, err := h.DB.GetProviderUsageStats(ctx, allTimeStart, now)
	if err != nil {
		h.Log.ErrorContext(ctx, "provider usage all time", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to retrieve provider usage")
	}

	return c.JSON(fiber.Map{
		"today": fiber.Map{
			"from":        todayStart.Format(time.RFC3339),
			"to":          now.Format(time.RFC3339),
			"totals":      todayTotals,
			"by_provider": providerUsageMap(todayStats),
		},
		"all_time": fiber.Map{
			"from":        allTimeStart.Format(time.RFC3339),
			"to":          now.Format(time.RFC3339),
			"totals":      allTotals,
			"by_provider": providerUsageMap(allStats),
		},
	})
}