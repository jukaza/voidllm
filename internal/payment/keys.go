package payment

// Settings keys stored in the settings table.
const (
	KeySepayEnabled         = "payment.sepay.enabled"
	KeySepayBankCode        = "payment.sepay.bank_code"
	KeySepayAccountNumber   = "payment.sepay.account_number"
	KeySepayAccountName     = "payment.sepay.account_name"
	KeySepayWebhookToken    = "payment.sepay.webhook_token"
	KeySepayWebhookAuthMode = "payment.sepay.webhook_auth_mode"
	KeySepayWebhookSecret   = "payment.sepay.webhook_secret"
	KeySepayWebhookIPCheck  = "payment.sepay.webhook_ip_check"
	KeySepayMinAmount       = "payment.sepay.min_amount"
	KeySepayMaxAmount       = "payment.sepay.max_amount"
	KeySepayOrderTTLMinutes = "payment.sepay.order_ttl_minutes"
	KeyAmountPresets        = "payment.amount_presets"
	KeyTierBonuses          = "payment.tier_bonuses"
	KeyCampaigns            = "payment.campaigns"
	KeyFirstTopupBonus      = "payment.first_topup_bonus"
	KeyBonusStackMode       = "payment.bonus_stack_mode"
)