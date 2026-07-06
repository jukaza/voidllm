package admin

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/features"
)

// GetPublicFeatures handles GET /api/v1/public/features.
func (h *Handler) GetPublicFeatures(c fiber.Ctx) error {
	cfg := h.currentFeatures(c)
	return c.JSON(features.PublicView(cfg))
}

// GetAdminFeaturesSettings handles GET /api/v1/admin/settings/features.
func (h *Handler) GetAdminFeaturesSettings(c fiber.Ctx) error {
	return c.JSON(h.currentFeatures(c))
}

// UpdateAdminFeaturesSettings handles PUT /api/v1/admin/settings/features.
func (h *Handler) UpdateAdminFeaturesSettings(c fiber.Ctx) error {
	var input features.UpdateInput
	if err := c.Bind().JSON(&input); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	cfg, err := features.Update(c.Context(), h.DB, input)
	if err != nil {
		if strings.Contains(err.Error(), "initial_balance_vnd") {
			return apierror.BadRequest(c, err.Error())
		}
		h.Log.ErrorContext(c.Context(), "admin features: update", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to save feature settings")
	}

	if h.FeaturesRuntime != nil {
		h.FeaturesRuntime.Set(cfg)
	}
	if h.ApplyFeatures != nil {
		if err := h.ApplyFeatures(c.Context(), cfg); err != nil {
			h.Log.ErrorContext(c.Context(), "admin features: apply", slog.String("error", err.Error()))
			return apierror.InternalError(c, "failed to apply feature settings")
		}
	}

	return c.JSON(cfg)
}

func (h *Handler) currentFeatures(c fiber.Ctx) features.Config {
	if h.FeaturesRuntime != nil {
		return h.FeaturesRuntime.Get()
	}
	cfg, err := features.Load(c.Context(), h.DB)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "features: load", slog.String("error", err.Error()))
		return features.DefaultConfig()
	}
	return cfg
}