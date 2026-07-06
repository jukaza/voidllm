package admin

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/security"
	"github.com/jukaza/tavo/internal/site"
)

// GetPublicAuthConfig handles GET /api/v1/public/auth-config.
func (h *Handler) GetPublicAuthConfig(c fiber.Ctx) error {
	ctx := c.Context()
	sec, err := security.Load(ctx, h.DB)
	if err != nil {
		h.Log.ErrorContext(ctx, "public auth-config: security", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load auth configuration")
	}
	siteCfg, err := site.Load(ctx, h.DB)
	if err != nil {
		h.Log.ErrorContext(ctx, "public auth-config: site", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load auth configuration")
	}
	return c.JSON(security.PublicFrom(sec, siteCfg.RegisterEnabled))
}

// AuthProviders handles GET /api/v1/auth/providers (legacy alias).
func (h *Handler) AuthProviders(c fiber.Ctx) error {
	return h.GetPublicAuthConfig(c)
}