package usage

import "encoding/json"

// EventMeta is the JSON shape stored in usage_events.meta.
type EventMeta struct {
	ProviderSlug  string              `json:"provider_slug,omitempty"`
	BillingMode   string              `json:"billing_mode,omitempty"`
	SellBreakdown *TokenCostBreakdown `json:"sell_breakdown,omitempty"`
	Estimated     bool                `json:"estimated,omitempty"`
}

// TokenCostBreakdown holds per-category sell charges in VND.
type TokenCostBreakdown struct {
	Input      float64 `json:"input,omitempty"`
	Cached     float64 `json:"cached,omitempty"`
	CacheWrite float64 `json:"cache_write,omitempty"`
	Output     float64 `json:"output,omitempty"`
	Flat       float64 `json:"flat,omitempty"`
}

// BuildEventMeta constructs the metadata blob for a usage event.
func BuildEventMeta(provider, billingMode string, revenue *float64, sellBD *TokenCostBreakdown, estimated bool) json.RawMessage {
	meta := EventMeta{
		ProviderSlug:  provider,
		BillingMode:   billingMode,
		SellBreakdown: sellBD,
		Estimated:     estimated,
	}
	_ = revenue
	b, err := json.Marshal(meta)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

// BillingModeFromFlags returns a stable billing mode label for storage.
func BillingModeFromFlags(billPerToken, billPerRequest, hasRevenue bool) string {
	switch {
	case billPerRequest && hasRevenue:
		return "bill_per_request"
	case billPerToken && hasRevenue:
		return "bill_per_token"
	case hasRevenue:
		return "billed"
	default:
		return "unbilled"
	}
}

// LogTypeFromStatus maps HTTP status to consume vs error log types.
func LogTypeFromStatus(statusCode int) string {
	if statusCode >= 400 {
		return "error"
	}
	return "consume"
}