package admin

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/site"
)

// GetPublicSite handles GET /api/v1/public/site — unauthenticated branding and
// legal content for the storefront and console UI.
func (h *Handler) GetPublicSite(c fiber.Ctx) error {
	cfg, err := site.Load(c.Context(), h.DB)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "public site: load", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load site settings")
	}
	return c.JSON(cfg)
}

// GetAdminSite handles GET /api/v1/admin/settings/site.
func (h *Handler) GetAdminSite(c fiber.Ctx) error {
	cfg, err := site.Load(c.Context(), h.DB)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "admin site: load", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load site settings")
	}
	return c.JSON(cfg)
}

// UpdateAdminSite handles PUT /api/v1/admin/settings/site.
func (h *Handler) UpdateAdminSite(c fiber.Ctx) error {
	var input site.UpdateInput
	if err := c.Bind().JSON(&input); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	cfg, err := site.Update(c.Context(), h.DB, input)
	if err != nil {
		if strings.Contains(err.Error(), "system_name is required") ||
			strings.Contains(err.Error(), "announcement") ||
			strings.Contains(err.Error(), "announcements") {
			return apierror.BadRequest(c, err.Error())
		}
		h.Log.ErrorContext(c.Context(), "admin site: update", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to save site settings")
	}
	return c.JSON(cfg)
}