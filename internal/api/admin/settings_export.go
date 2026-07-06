package admin

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/features"
	"github.com/voidmind-io/voidllm/internal/payment"
	"github.com/voidmind-io/voidllm/internal/security"
	"github.com/voidmind-io/voidllm/internal/site"
)

type settingsExportPayload struct {
	ExportedAt string              `json:"exported_at"`
	Version    int                 `json:"version"`
	Site       site.Config         `json:"site"`
	Security   security.Config     `json:"security"`
	Payment    payment.AdminConfig `json:"payment"`
	Features   features.Config     `json:"features"`
}

type settingsImportBody struct {
	Payload settingsExportPayload `json:"payload"`
	Confirm bool                  `json:"confirm"`
}

type settingsImportPreview struct {
	Changed []string `json:"changed"`
}

func (h *Handler) ExportAdminSettings(c fiber.Ctx) error {
	ctx := c.Context()
	siteCfg, err := site.Load(ctx, h.DB)
	if err != nil {
		return apierror.InternalError(c, "failed to load site settings")
	}
	secCfg, err := security.Load(ctx, h.DB)
	if err != nil {
		return apierror.InternalError(c, "failed to load security settings")
	}
	payCfg, err := payment.LoadAdmin(ctx, h.DB, webhookURL(c))
	if err != nil {
		return apierror.InternalError(c, "failed to load payment settings")
	}
	featCfg := h.currentFeatures(c)
	return c.JSON(settingsExportPayload{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Version:    1,
		Site:       siteCfg,
		Security:   secCfg,
		Payment:    payCfg,
		Features:   featCfg,
	})
}

func (h *Handler) PreviewAdminSettingsImport(c fiber.Ctx) error {
	var body settingsImportBody
	if err := c.Bind().JSON(&body); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if body.Payload.Version != 1 {
		return apierror.BadRequest(c, "unsupported export version")
	}
	return c.JSON(settingsImportPreview{
		Changed: []string{"site", "security", "payment", "features"},
	})
}

func (h *Handler) ImportAdminSettings(c fiber.Ctx) error {
	var body settingsImportBody
	if err := c.Bind().JSON(&body); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if !body.Confirm {
		return apierror.BadRequest(c, "confirm must be true")
	}
	if body.Payload.Version != 1 {
		return apierror.BadRequest(c, "unsupported export version")
	}
	ctx := c.Context()

	if _, err := site.Update(ctx, h.DB, siteUpdateFromConfig(body.Payload.Site)); err != nil {
		h.Log.ErrorContext(ctx, "settings import: site", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to import site settings")
	}

	if _, err := security.Update(ctx, h.DB, securityUpdateFromConfig(body.Payload.Security)); err != nil {
		h.Log.ErrorContext(ctx, "settings import: security", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to import security settings")
	}

	if _, err := payment.Update(ctx, h.DB, paymentUpdateFromAdmin(body.Payload.Payment)); err != nil {
		h.Log.ErrorContext(ctx, "settings import: payment", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to import payment settings")
	}

	wallet := body.Payload.Features.Wallet
	modules := body.Payload.Features.Modules
	cfg, err := features.Update(ctx, h.DB, features.UpdateInput{
		Wallet:  &wallet,
		Modules: &modules,
	})
	if err != nil {
		return apierror.InternalError(c, "failed to import feature settings")
	}
	if h.FeaturesRuntime != nil {
		h.FeaturesRuntime.Set(cfg)
	}
	if h.ApplyFeatures != nil {
		if err := h.ApplyFeatures(ctx, cfg); err != nil {
			return apierror.InternalError(c, "failed to apply feature settings")
		}
	}

	return c.JSON(fiber.Map{"ok": true})
}

func siteUpdateFromConfig(cfg site.Config) site.UpdateInput {
	announcements := cfg.Announcements
	return site.UpdateInput{
		SystemName:        &cfg.SystemName,
		Logo:              &cfg.Logo,
		ServerAddress:     &cfg.ServerAddress,
		Footer:            &cfg.Footer,
		About:             &cfg.About,
		HomePageContent:   &cfg.HomePageContent,
		UserAgreement:     &cfg.UserAgreement,
		PrivacyPolicy:     &cfg.PrivacyPolicy,
		Announcements:     &announcements,
		NoticeEnabled:     &cfg.NoticeEnabled,
		RegisterEnabled:   &cfg.RegisterEnabled,
		SiteSubtitle:      &cfg.SiteSubtitle,
		SupportZalo:       &cfg.SupportZalo,
		SupportTelegram:   &cfg.SupportTelegram,
		DocURL:            &cfg.DocURL,
	}
}

func securityUpdateFromConfig(cfg security.Config) security.UpdateInput {
	return security.UpdateInput{
		Turnstile: &security.TurnstileUpdate{
			Enabled: &cfg.Turnstile.Enabled,
			SiteKey: &cfg.Turnstile.SiteKey,
		},
		OAuth: &security.OAuthUpdate{
			Google: oauthUpdateFromProvider(cfg.OAuth.Google),
			GitHub: oauthUpdateFromProvider(cfg.OAuth.GitHub),
		},
		Session: &security.SessionPolicyUpdate{
			TTLHours:      &cfg.Session.TTLHours,
			AllowMultiple: &cfg.Session.AllowMultiple,
			MaxConcurrent: &cfg.Session.MaxConcurrent,
		},
		Password: &security.PasswordPolicyUpdate{
			MinLength:             &cfg.Password.MinLength,
			AllowOAuthSetPassword: &cfg.Password.AllowOAuthSetPassword,
		},
	}
}

func oauthUpdateFromProvider(p security.OAuthProviderConfig) *security.OAuthProviderUpdate {
	return &security.OAuthProviderUpdate{
		Enabled:     &p.Enabled,
		AllowLogin:  &p.AllowLogin,
		AllowSignup: &p.AllowSignup,
		ClientID:    &p.ClientID,
	}
}

func paymentUpdateFromAdmin(cfg payment.AdminConfig) payment.UpdateInput {
	sepay := cfg.Sepay
	presets := cfg.AmountPresets
	tiers := cfg.TierBonuses
	campaigns := cfg.Campaigns
	first := cfg.FirstTopup
	stack := cfg.BonusStackMode
	return payment.UpdateInput{
		Sepay:          &sepay,
		AmountPresets:  &presets,
		TierBonuses:    &tiers,
		Campaigns:      &campaigns,
		FirstTopup:     &first,
		BonusStackMode: &stack,
	}
}