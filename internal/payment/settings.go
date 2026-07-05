package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SettingsStore reads and writes payment settings.
type SettingsStore interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	SetSettingIfNotExists(ctx context.Context, key, value string) error
}

// SepayConfig holds the single bank account used for VietQR transfers.
type SepayConfig struct {
	Enabled          bool   `json:"enabled"`
	BankCode         string `json:"bank_code"`
	AccountNumber    string `json:"account_number"`
	AccountName      string `json:"account_name"`
	WebhookAuthMode  string `json:"webhook_auth_mode"` // api_key | hmac
	WebhookToken     string `json:"webhook_token,omitempty"`
	WebhookSecret    string `json:"webhook_secret,omitempty"`
	WebhookIPCheck   bool   `json:"webhook_ip_check"`
	MinAmount        float64 `json:"min_amount"`
	MaxAmount        float64 `json:"max_amount"`
	OrderTTLMinutes  int    `json:"order_ttl_minutes"`
}

// TierBonus applies when pay_amount reaches min_amount.
// BonusType is "percent" or "fixed" — only one reward mode per tier.
type TierBonus struct {
	MinAmount    float64 `json:"min_amount"`
	BonusType    string  `json:"bonus_type"`
	BonusPercent float64 `json:"bonus_percent"`
	BonusFixed   float64 `json:"bonus_fixed"`
	Label        string  `json:"label,omitempty"`
}

// Campaign is a time-bounded promotion.
// BonusType is "percent" or "fixed" — only one reward mode per campaign.
type Campaign struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Enabled        bool    `json:"enabled"`
	StartAt        string  `json:"start_at"`
	EndAt          string  `json:"end_at"`
	BonusType      string  `json:"bonus_type"`
	BonusPercent   float64 `json:"bonus_percent"`
	BonusFixed     float64 `json:"bonus_fixed"`
	MinAmount      float64 `json:"min_amount"`
	MaxBonus       float64 `json:"max_bonus"`
	FirstTopupOnly bool    `json:"first_topup_only"`
}

// FirstTopupBonus rewards a user's first successful top-up.
// BonusType is "percent" or "fixed" — only one reward mode.
type FirstTopupBonus struct {
	Enabled      bool    `json:"enabled"`
	BonusType    string  `json:"bonus_type"`
	BonusPercent float64 `json:"bonus_percent"`
	BonusFixed   float64 `json:"bonus_fixed"`
}

// Config is the full payment configuration.
type Config struct {
	Sepay                 SepayConfig       `json:"sepay"`
	AmountPresets         []float64         `json:"amount_presets"`
	TierBonuses           []TierBonus       `json:"tier_bonuses"`
	Campaigns             []Campaign        `json:"campaigns"`
	FirstTopup            FirstTopupBonus   `json:"first_topup"`
	BonusStackMode        string            `json:"bonus_stack_mode"`
	WebhookTokenConfigured  bool   `json:"webhook_token_configured"`
	WebhookSecretConfigured bool   `json:"webhook_secret_configured"`
	WebhookAuthMode         string `json:"webhook_auth_mode"`
}

// PublicConfig is exposed to authenticated customers (no secrets).
type PublicConfig struct {
	Enabled        bool              `json:"enabled"`
	MinAmount      float64           `json:"min_amount"`
	MaxAmount      float64           `json:"max_amount"`
	AmountPresets  []float64         `json:"amount_presets"`
	TierBonuses    []TierBonus       `json:"tier_bonuses"`
	Campaigns      []ActiveCampaign  `json:"active_campaigns"`
	FirstTopup     FirstTopupBonus   `json:"first_topup"`
	BonusStackMode string            `json:"bonus_stack_mode"`
	Banks          []BankOption      `json:"banks"`
}

// ActiveCampaign is a campaign currently in effect for display.
type ActiveCampaign struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	BonusType    string  `json:"bonus_type"`
	BonusPercent float64 `json:"bonus_percent"`
	BonusFixed   float64 `json:"bonus_fixed"`
	MinAmount    float64 `json:"min_amount"`
	MaxBonus     float64 `json:"max_bonus"`
	EndAt        string  `json:"end_at"`
}

// AdminConfig includes webhook metadata for the settings UI.
type AdminConfig struct {
	Config
	WebhookURL string `json:"webhook_url"`
}

// UpdateInput is the admin payload for PUT /admin/settings/payment.
type UpdateInput struct {
	Sepay          *SepayConfig     `json:"sepay"`
	AmountPresets  *[]float64       `json:"amount_presets"`
	TierBonuses    *[]TierBonus     `json:"tier_bonuses"`
	Campaigns      *[]Campaign      `json:"campaigns"`
	FirstTopup     *FirstTopupBonus `json:"first_topup"`
	BonusStackMode *string          `json:"bonus_stack_mode"`
}

// EnsureDefaults seeds first-run payment settings.
func EnsureDefaults(ctx context.Context, store SettingsStore) error {
	defaults := map[string]string{
		KeySepayEnabled:         "false",
		KeySepayBankCode:        "MB",
		KeySepayAccountNumber:   "",
		KeySepayAccountName:     "",
		KeySepayWebhookToken:    "",
		KeySepayWebhookAuthMode: WebhookAuthAPIKey,
		KeySepayWebhookSecret:   "",
		KeySepayWebhookIPCheck:  "false",
		KeySepayMinAmount:       "50000",
		KeySepayMaxAmount:       "50000000",
		KeySepayOrderTTLMinutes: "15",
		KeyAmountPresets:        `[50000,100000,200000,500000,1000000]`,
		KeyTierBonuses:          `[]`,
		KeyCampaigns:            `[]`,
		KeyFirstTopupBonus:      `{"enabled":false,"bonus_percent":0,"bonus_fixed":0}`,
		KeyBonusStackMode:       "stack",
	}
	for key, value := range defaults {
		if err := store.SetSettingIfNotExists(ctx, key, value); err != nil {
			return fmt.Errorf("seed %s: %w", key, err)
		}
	}
	return nil
}

// Load reads payment settings from the store.
func Load(ctx context.Context, store SettingsStore) (Config, error) {
	if err := EnsureDefaults(ctx, store); err != nil {
		return Config{}, err
	}

	enabled, err := loadBool(ctx, store, KeySepayEnabled, false)
	if err != nil {
		return Config{}, err
	}
	minAmount, err := loadFloat(ctx, store, KeySepayMinAmount, 50000)
	if err != nil {
		return Config{}, err
	}
	maxAmount, err := loadFloat(ctx, store, KeySepayMaxAmount, 50000000)
	if err != nil {
		return Config{}, err
	}
	ttl, err := loadInt(ctx, store, KeySepayOrderTTLMinutes, 15)
	if err != nil {
		return Config{}, err
	}
	bankCode, err := store.GetSetting(ctx, KeySepayBankCode)
	if err != nil {
		return Config{}, err
	}
	accountNumber, err := store.GetSetting(ctx, KeySepayAccountNumber)
	if err != nil {
		return Config{}, err
	}
	accountName, err := store.GetSetting(ctx, KeySepayAccountName)
	if err != nil {
		return Config{}, err
	}
	webhookToken, err := store.GetSetting(ctx, KeySepayWebhookToken)
	if err != nil {
		return Config{}, err
	}
	webhookAuthMode, err := store.GetSetting(ctx, KeySepayWebhookAuthMode)
	if err != nil {
		return Config{}, err
	}
	webhookSecret, err := store.GetSetting(ctx, KeySepayWebhookSecret)
	if err != nil {
		return Config{}, err
	}
	webhookIPCheck, err := loadBool(ctx, store, KeySepayWebhookIPCheck, false)
	if err != nil {
		return Config{}, err
	}
	if webhookAuthMode != WebhookAuthHMAC {
		webhookAuthMode = WebhookAuthAPIKey
	}
	presets, err := loadFloatSlice(ctx, store, KeyAmountPresets, []float64{50000, 100000, 200000, 500000, 1000000})
	if err != nil {
		return Config{}, err
	}
	tiers, err := loadTierBonuses(ctx, store)
	if err != nil {
		return Config{}, err
	}
	campaigns, err := loadCampaigns(ctx, store)
	if err != nil {
		return Config{}, err
	}
	firstTopup, err := loadFirstTopup(ctx, store)
	if err != nil {
		return Config{}, err
	}
	stackMode, err := store.GetSetting(ctx, KeyBonusStackMode)
	if err != nil {
		return Config{}, err
	}
	if stackMode != "max" {
		stackMode = "stack"
	}

	return Config{
		Sepay: SepayConfig{
			Enabled:          enabled,
			BankCode:         strings.TrimSpace(bankCode),
			AccountNumber:    strings.TrimSpace(accountNumber),
			AccountName:      strings.TrimSpace(accountName),
			WebhookAuthMode:  webhookAuthMode,
			WebhookToken:     webhookToken,
			WebhookSecret:    webhookSecret,
			WebhookIPCheck:   webhookIPCheck,
			MinAmount:        minAmount,
			MaxAmount:        maxAmount,
			OrderTTLMinutes:  ttl,
		},
		AmountPresets:           presets,
		TierBonuses:             tiers,
		Campaigns:               campaigns,
		FirstTopup:              firstTopup,
		BonusStackMode:          stackMode,
		WebhookTokenConfigured:  strings.TrimSpace(webhookToken) != "",
		WebhookSecretConfigured: strings.TrimSpace(webhookSecret) != "",
		WebhookAuthMode:         webhookAuthMode,
	}, nil
}

// LoadAdmin returns settings for the admin UI without exposing the raw token.
func LoadAdmin(ctx context.Context, store SettingsStore, webhookURL string) (AdminConfig, error) {
	cfg, err := Load(ctx, store)
	if err != nil {
		return AdminConfig{}, err
	}
	cfg.Sepay.WebhookToken = ""
	cfg.Sepay.WebhookSecret = ""
	return AdminConfig{Config: cfg, WebhookURL: webhookURL}, nil
}

// PublicFromConfig builds the customer-facing config snapshot.
func PublicFromConfig(cfg Config, now time.Time, hasCompletedTopup bool) PublicConfig {
	active := make([]ActiveCampaign, 0, len(cfg.Campaigns))
	for _, c := range cfg.Campaigns {
		if !campaignActive(c, now, hasCompletedTopup) {
			continue
		}
		normalized := NormalizeCampaign(c)
		active = append(active, ActiveCampaign{
			ID:           normalized.ID,
			Name:         normalized.Name,
			BonusType:    normalized.BonusType,
			BonusPercent: normalized.BonusPercent,
			BonusFixed:   normalized.BonusFixed,
			MinAmount:    normalized.MinAmount,
			MaxBonus:     normalized.MaxBonus,
			EndAt:        normalized.EndAt,
		})
	}
	firstTopup := cfg.FirstTopup
	if hasCompletedTopup {
		firstTopup = FirstTopupBonus{}
	}
	return PublicConfig{
		Enabled:        cfg.Sepay.Enabled && cfg.Sepay.IsConfigured(),
		MinAmount:      cfg.Sepay.MinAmount,
		MaxAmount:      cfg.Sepay.MaxAmount,
		AmountPresets:  cfg.AmountPresets,
		TierBonuses:    cfg.TierBonuses,
		Campaigns:      active,
		FirstTopup:     firstTopup,
		BonusStackMode: cfg.BonusStackMode,
		Banks:          SupportedBanks,
	}
}

// Update applies partial admin changes.
func Update(ctx context.Context, store SettingsStore, input UpdateInput) (Config, error) {
	if err := EnsureDefaults(ctx, store); err != nil {
		return Config{}, err
	}

	if input.Sepay != nil {
		s := *input.Sepay
		if err := store.SetSetting(ctx, KeySepayEnabled, strconv.FormatBool(s.Enabled)); err != nil {
			return Config{}, err
		}
		if err := store.SetSetting(ctx, KeySepayBankCode, strings.TrimSpace(s.BankCode)); err != nil {
			return Config{}, err
		}
		if err := store.SetSetting(ctx, KeySepayAccountNumber, strings.TrimSpace(s.AccountNumber)); err != nil {
			return Config{}, err
		}
		if err := store.SetSetting(ctx, KeySepayAccountName, strings.TrimSpace(s.AccountName)); err != nil {
			return Config{}, err
		}
		authMode := strings.TrimSpace(s.WebhookAuthMode)
		if authMode == WebhookAuthHMAC || authMode == WebhookAuthAPIKey {
			if err := store.SetSetting(ctx, KeySepayWebhookAuthMode, authMode); err != nil {
				return Config{}, err
			}
		}
		if token := strings.TrimSpace(s.WebhookToken); token != "" {
			if err := store.SetSetting(ctx, KeySepayWebhookToken, token); err != nil {
				return Config{}, err
			}
		}
		if secret := strings.TrimSpace(s.WebhookSecret); secret != "" {
			if err := store.SetSetting(ctx, KeySepayWebhookSecret, secret); err != nil {
				return Config{}, err
			}
		}
		if err := store.SetSetting(ctx, KeySepayWebhookIPCheck, strconv.FormatBool(s.WebhookIPCheck)); err != nil {
			return Config{}, err
		}
		if s.MinAmount > 0 {
			if err := store.SetSetting(ctx, KeySepayMinAmount, strconv.FormatFloat(s.MinAmount, 'f', -1, 64)); err != nil {
				return Config{}, err
			}
		}
		if s.MaxAmount > 0 {
			if err := store.SetSetting(ctx, KeySepayMaxAmount, strconv.FormatFloat(s.MaxAmount, 'f', -1, 64)); err != nil {
				return Config{}, err
			}
		}
		if s.OrderTTLMinutes > 0 {
			if err := store.SetSetting(ctx, KeySepayOrderTTLMinutes, strconv.Itoa(s.OrderTTLMinutes)); err != nil {
				return Config{}, err
			}
		}
	}
	if input.AmountPresets != nil {
		if err := saveJSON(ctx, store, KeyAmountPresets, *input.AmountPresets); err != nil {
			return Config{}, err
		}
	}
	if input.TierBonuses != nil {
		tiers, err := normalizeTierBonuses(*input.TierBonuses)
		if err != nil {
			return Config{}, err
		}
		if err := saveJSON(ctx, store, KeyTierBonuses, tiers); err != nil {
			return Config{}, err
		}
	}
	if input.Campaigns != nil {
		campaigns, err := normalizeCampaigns(*input.Campaigns)
		if err != nil {
			return Config{}, err
		}
		if err := saveJSON(ctx, store, KeyCampaigns, campaigns); err != nil {
			return Config{}, err
		}
	}
	if input.FirstTopup != nil {
		firstTopup, err := normalizeFirstTopupBonus(*input.FirstTopup)
		if err != nil {
			return Config{}, err
		}
		if err := saveJSON(ctx, store, KeyFirstTopupBonus, firstTopup); err != nil {
			return Config{}, err
		}
	}
	if input.BonusStackMode != nil {
		mode := strings.TrimSpace(*input.BonusStackMode)
		if mode != "max" && mode != "stack" {
			return Config{}, fmt.Errorf("bonus_stack_mode must be stack or max")
		}
		if err := store.SetSetting(ctx, KeyBonusStackMode, mode); err != nil {
			return Config{}, err
		}
	}
	return Load(ctx, store)
}

func loadBool(ctx context.Context, store SettingsStore, key string, fallback bool) (bool, error) {
	raw, err := store.GetSetting(ctx, key)
	if err != nil {
		return false, err
	}
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback, nil
	}
	return parsed, nil
}

func loadFloat(ctx context.Context, store SettingsStore, key string, fallback float64) (float64, error) {
	raw, err := store.GetSetting(ctx, key)
	if err != nil {
		return 0, err
	}
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback, nil
	}
	return parsed, nil
}

func loadInt(ctx context.Context, store SettingsStore, key string, fallback int) (int, error) {
	raw, err := store.GetSetting(ctx, key)
	if err != nil {
		return 0, err
	}
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback, nil
	}
	return parsed, nil
}

func loadFloatSlice(ctx context.Context, store SettingsStore, key string, fallback []float64) ([]float64, error) {
	raw, err := store.GetSetting(ctx, key)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	var values []float64
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return fallback, nil
	}
	return values, nil
}

func loadTierBonuses(ctx context.Context, store SettingsStore) ([]TierBonus, error) {
	raw, err := store.GetSetting(ctx, KeyTierBonuses)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		return []TierBonus{}, nil
	}
	var tiers []TierBonus
	if err := json.Unmarshal([]byte(raw), &tiers); err != nil {
		return []TierBonus{}, nil
	}
	for i := range tiers {
		tiers[i] = NormalizeTierBonus(tiers[i])
	}
	return tiers, nil
}

func loadCampaigns(ctx context.Context, store SettingsStore) ([]Campaign, error) {
	raw, err := store.GetSetting(ctx, KeyCampaigns)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		return []Campaign{}, nil
	}
	var campaigns []Campaign
	if err := json.Unmarshal([]byte(raw), &campaigns); err != nil {
		return []Campaign{}, nil
	}
	for i := range campaigns {
		campaigns[i] = NormalizeCampaign(campaigns[i])
	}
	return campaigns, nil
}

func loadFirstTopup(ctx context.Context, store SettingsStore) (FirstTopupBonus, error) {
	raw, err := store.GetSetting(ctx, KeyFirstTopupBonus)
	if err != nil {
		return FirstTopupBonus{}, err
	}
	if strings.TrimSpace(raw) == "" {
		return FirstTopupBonus{}, nil
	}
	var bonus FirstTopupBonus
	if err := json.Unmarshal([]byte(raw), &bonus); err != nil {
		return FirstTopupBonus{}, nil
	}
	return NormalizeFirstTopup(bonus), nil
}

func normalizeTierBonuses(tiers []TierBonus) ([]TierBonus, error) {
	out := make([]TierBonus, len(tiers))
	for i, tier := range tiers {
		normalized := NormalizeTierBonus(tier)
		if normalized.MinAmount > 0 && !hasPositiveBonus(normalized.BonusType, normalized.BonusPercent, normalized.BonusFixed) {
			return nil, fmt.Errorf("tier %d: choose either bonus percent or fixed amount", i+1)
		}
		out[i] = normalized
	}
	return out, nil
}

func normalizeCampaigns(campaigns []Campaign) ([]Campaign, error) {
	out := make([]Campaign, len(campaigns))
	for i, campaign := range campaigns {
		normalized := NormalizeCampaign(campaign)
		if normalized.Enabled && normalized.MinAmount > 0 && !hasPositiveBonus(normalized.BonusType, normalized.BonusPercent, normalized.BonusFixed) {
			return nil, fmt.Errorf("campaign %d: choose either bonus percent or fixed amount", i+1)
		}
		out[i] = normalized
	}
	return out, nil
}

func normalizeFirstTopupBonus(bonus FirstTopupBonus) (FirstTopupBonus, error) {
	normalized := NormalizeFirstTopup(bonus)
	if normalized.Enabled && !hasPositiveBonus(normalized.BonusType, normalized.BonusPercent, normalized.BonusFixed) {
		return FirstTopupBonus{}, fmt.Errorf("first top-up bonus: choose either bonus percent or fixed amount")
	}
	return normalized, nil
}

func saveJSON(ctx context.Context, store SettingsStore, key string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", key, err)
	}
	return store.SetSetting(ctx, key, string(encoded))
}

func campaignActive(c Campaign, now time.Time, hasCompletedTopup bool) bool {
	if !c.Enabled {
		return false
	}
	if c.FirstTopupOnly && hasCompletedTopup {
		return false
	}
	start, err1 := time.Parse(time.RFC3339, c.StartAt)
	end, err2 := time.Parse(time.RFC3339, c.EndAt)
	if err1 != nil || err2 != nil {
		return false
	}
	return !now.Before(start) && !now.After(end)
}