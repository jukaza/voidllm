package admin

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/site"
)

// UploadSiteLogo handles POST /api/v1/admin/settings/site/logo (multipart field "logo").
func (h *Handler) UploadSiteLogo(c fiber.Ctx) error {
	file, err := c.FormFile("logo")
	if err != nil {
		return apierror.BadRequest(c, "logo file is required")
	}

	publicPath, err := site.SaveUploadedLogo(h.DataDir, file)
	if err != nil {
		if strings.Contains(err.Error(), "exceeds") ||
			strings.Contains(err.Error(), "unsupported") ||
			strings.Contains(err.Error(), "must be") ||
			strings.Contains(err.Error(), "required") ||
			strings.Contains(err.Error(), "empty") {
			return apierror.BadRequest(c, err.Error())
		}
		h.Log.ErrorContext(c.Context(), "site logo upload", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to save logo")
	}

	cfg, err := site.Update(c.Context(), h.DB, site.UpdateInput{Logo: &publicPath})
	if err != nil {
		h.Log.ErrorContext(c.Context(), "site logo update", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to update site settings")
	}
	return c.JSON(cfg)
}

// ResetSiteLogo handles DELETE /api/v1/admin/settings/site/logo.
func (h *Handler) ResetSiteLogo(c fiber.Ctx) error {
	if err := site.RemoveUploadedLogo(h.DataDir); err != nil {
		h.Log.ErrorContext(c.Context(), "site logo reset", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to remove logo file")
	}

	defaultLogo := site.DefaultLogo
	cfg, err := site.Update(c.Context(), h.DB, site.UpdateInput{Logo: &defaultLogo})
	if err != nil {
		h.Log.ErrorContext(c.Context(), "site logo reset update", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to update site settings")
	}
	return c.JSON(cfg)
}