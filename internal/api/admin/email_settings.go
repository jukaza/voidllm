package admin

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/email"
)

// GetAdminEmailSettings handles GET /api/v1/admin/settings/email.
func (h *Handler) GetAdminEmailSettings(c fiber.Ctx) error {
	cfg, err := email.LoadAdmin(c.Context(), h.DB)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "admin email: load", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load email settings")
	}
	return c.JSON(cfg)
}

// UpdateAdminEmailSettings handles PUT /api/v1/admin/settings/email.
func (h *Handler) UpdateAdminEmailSettings(c fiber.Ctx) error {
	var input email.UpdateInput
	if err := c.Bind().JSON(&input); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	if input.SMTP != nil && input.SMTP.Enabled {
		existing, err := email.Load(c.Context(), h.DB)
		if err != nil {
			return apierror.InternalError(c, "failed to load email settings")
		}
		merged := *input.SMTP
		if strings.TrimSpace(merged.Host) == "" {
			merged.Host = existing.Host
		}
		if merged.Port <= 0 {
			merged.Port = existing.Port
		}
		if strings.TrimSpace(merged.Username) == "" {
			merged.Username = existing.Username
		}
		if strings.TrimSpace(merged.From) == "" {
			merged.From = existing.From
		}
		if strings.TrimSpace(merged.Password) == "" {
			merged.Password = existing.Password
		}
		input.SMTP = &merged
		if !merged.IsConfigured() || strings.TrimSpace(merged.Password) == "" {
			return apierror.BadRequest(c, "smtp host, port, username, from address, and app password are required")
		}
	}

	cfg, err := email.Update(c.Context(), h.DB, input)
	if err != nil {
		if strings.Contains(err.Error(), "smtp port") {
			return apierror.BadRequest(c, err.Error())
		}
		h.Log.ErrorContext(c.Context(), "admin email: update", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to save email settings")
	}

	adminCfg, err := email.LoadAdmin(c.Context(), h.DB)
	if err != nil {
		return apierror.InternalError(c, "failed to load email settings")
	}
	adminCfg.SMTPConfig = email.SMTPConfig{
		Enabled:    cfg.Enabled,
		Host:       cfg.Host,
		Port:       cfg.Port,
		Username:   cfg.Username,
		From:       cfg.From,
		SSLEnabled: cfg.SSLEnabled,
	}
	return c.JSON(adminCfg)
}