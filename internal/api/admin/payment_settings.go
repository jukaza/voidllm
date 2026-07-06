package admin

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/payment"
)

func webhookURL(c fiber.Ctx) string {
	proto := c.Get("X-Forwarded-Proto")
	if proto == "" {
		if c.Protocol() == "https" {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := c.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Hostname()
	}
	return proto + "://" + host + "/api/v1/webhooks/sepay"
}

// GetAdminPaymentSettings handles GET /api/v1/admin/settings/payment.
func (h *Handler) GetAdminPaymentSettings(c fiber.Ctx) error {
	cfg, err := payment.LoadAdmin(c.Context(), h.DB, webhookURL(c))
	if err != nil {
		h.Log.ErrorContext(c.Context(), "admin payment: load", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load payment settings")
	}
	return c.JSON(cfg)
}

// UpdateAdminPaymentSettings handles PUT /api/v1/admin/settings/payment.
func (h *Handler) UpdateAdminPaymentSettings(c fiber.Ctx) error {
	var input payment.UpdateInput
	if err := c.Bind().JSON(&input); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if input.Sepay != nil && input.Sepay.Enabled && !input.Sepay.IsConfigured() {
		existing, err := payment.Load(c.Context(), h.DB)
		if err != nil {
			return apierror.InternalError(c, "failed to load payment settings")
		}
		merged := *input.Sepay
		if strings.TrimSpace(merged.BankCode) == "" {
			merged.BankCode = existing.Sepay.BankCode
		}
		if strings.TrimSpace(merged.AccountNumber) == "" {
			merged.AccountNumber = existing.Sepay.AccountNumber
		}
		if strings.TrimSpace(merged.AccountName) == "" {
			merged.AccountName = existing.Sepay.AccountName
		}
		if merged.MinAmount <= 0 {
			merged.MinAmount = existing.Sepay.MinAmount
		}
		if merged.MaxAmount <= 0 {
			merged.MaxAmount = existing.Sepay.MaxAmount
		}
		if merged.OrderTTLMinutes <= 0 {
			merged.OrderTTLMinutes = existing.Sepay.OrderTTLMinutes
		}
		if strings.TrimSpace(merged.WebhookToken) == "" {
			merged.WebhookToken = existing.Sepay.WebhookToken
		}
		if strings.TrimSpace(merged.WebhookSecret) == "" {
			merged.WebhookSecret = existing.Sepay.WebhookSecret
		}
		if strings.TrimSpace(merged.WebhookAuthMode) == "" {
			merged.WebhookAuthMode = existing.Sepay.WebhookAuthMode
		}
		if merged.WebhookAuthMode == "" {
			merged.WebhookAuthMode = payment.WebhookAuthAPIKey
		}
		input.Sepay = &merged
		if !merged.IsConfigured() {
			return apierror.BadRequest(c, "bank code, account number, account name, and webhook credentials are required")
		}
	}

	cfg, err := payment.Update(c.Context(), h.DB, input)
	if err != nil {
		if strings.Contains(err.Error(), "bonus_stack_mode") ||
			strings.Contains(err.Error(), "choose either bonus") {
			return apierror.BadRequest(c, err.Error())
		}
		h.Log.ErrorContext(c.Context(), "admin payment: update", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to save payment settings")
	}
	adminCfg, err := payment.LoadAdmin(c.Context(), h.DB, webhookURL(c))
	if err != nil {
		return apierror.InternalError(c, "failed to load payment settings")
	}
	adminCfg.Config = cfg
	return c.JSON(adminCfg)
}