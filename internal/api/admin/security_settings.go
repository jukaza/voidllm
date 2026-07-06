package admin

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/security"
)

type adminSecuritySettingsResponse struct {
	security.Config
	OAuthCallbackURLs map[string]string `json:"oauth_callback_urls"`
}

func (h *Handler) adminSecuritySettingsResponse(c fiber.Ctx) (adminSecuritySettingsResponse, error) {
	cfg, err := security.Load(c.Context(), h.DB)
	if err != nil {
		return adminSecuritySettingsResponse{}, err
	}
	return adminSecuritySettingsResponse{
		Config: cfg,
		OAuthCallbackURLs: map[string]string{
			"google": h.oauthCallbackURL(c, "google"),
			"github": h.oauthCallbackURL(c, "github"),
		},
	}, nil
}

// GetAdminSecuritySettings handles GET /api/v1/admin/settings/security.
func (h *Handler) GetAdminSecuritySettings(c fiber.Ctx) error {
	resp, err := h.adminSecuritySettingsResponse(c)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "admin security: load", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load security settings")
	}
	return c.JSON(resp)
}

// UpdateAdminSecuritySettings handles PUT /api/v1/admin/settings/security.
func (h *Handler) UpdateAdminSecuritySettings(c fiber.Ctx) error {
	var input security.UpdateInput
	if err := c.Bind().JSON(&input); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if _, err := security.Update(c.Context(), h.DB, input); err != nil {
		if errors.Is(err, security.ErrTurnstileNotConfigured) ||
			errors.Is(err, security.ErrSessionTTLOutOfRange) ||
			errors.Is(err, security.ErrSessionMaxConcurrentOutOfRange) ||
			errors.Is(err, security.ErrPasswordMinLengthOutOfRange) {
			return apierror.BadRequest(c, err.Error())
		}
		h.Log.ErrorContext(c.Context(), "admin security: update", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to save security settings")
	}
	resp, err := h.adminSecuritySettingsResponse(c)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "admin security: reload", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load security settings")
	}
	return c.JSON(resp)
}