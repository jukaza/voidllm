package subscription

import (
	"strings"
	"time"

	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/keys"
	"github.com/jukaza/tavo/internal/ratelimit"
)

// BillingPath indicates how a request should be charged.
type BillingPath string

const (
	BillingSubscription BillingPath = "subscription"
	BillingWallet       BillingPath = "wallet"
)

// ResolveBillingPath picks subscription vs wallet billing for a model request.
func ResolveBillingPath(keyInfo *auth.KeyInfo, model string) (BillingPath, *auth.SubscriptionBinding) {
	if keyInfo == nil || keyInfo.Subscription == nil {
		return BillingWallet, nil
	}
	sub := keyInfo.Subscription
	if time.Now().After(sub.ExpiresAt) {
		return BillingWallet, nil
	}
	if model == "" {
		return BillingWallet, nil
	}
	allowed := keys.ParseModelLimits(sub.AllowedModels)
	if len(allowed) == 0 {
		return BillingWallet, nil
	}
	if !keys.ModelAllowed(model, true, allowed) {
		return BillingWallet, nil
	}
	return BillingSubscription, sub
}

// EffectiveRateLimits returns RPM/RPD limits using the stricter of key vs subscription.
func EffectiveRateLimits(keyInfo *auth.KeyInfo, sub *auth.SubscriptionBinding, path BillingPath) ratelimit.Limits {
	limits := ratelimit.Limits{
		RequestsPerMinute: keyInfo.RequestsPerMinute,
		RequestsPerDay:    keyInfo.RequestsPerDay,
	}
	if path != BillingSubscription || sub == nil {
		return limits
	}
	if sub.RequestsPerMinute > 0 {
		if limits.RequestsPerMinute == 0 || sub.RequestsPerMinute < limits.RequestsPerMinute {
			limits.RequestsPerMinute = sub.RequestsPerMinute
		}
	}
	if sub.RequestsPerDay > 0 {
		if limits.RequestsPerDay == 0 || sub.RequestsPerDay < limits.RequestsPerDay {
			limits.RequestsPerDay = sub.RequestsPerDay
		}
	}
	return limits
}

// SubscriptionTokenLimits returns token limits for subscription scope checks.
func SubscriptionTokenLimits(sub *auth.SubscriptionBinding) ratelimit.Limits {
	if sub == nil {
		return ratelimit.Limits{}
	}
	return ratelimit.Limits{
		DailyTokenLimit:   sub.DailyTokenLimit,
		MonthlyTokenLimit: sub.MonthlyTokenLimit,
	}
}

// SubscriptionRequestLimits returns daily/monthly request limits on subscription scope.
func SubscriptionRequestLimits(sub *auth.SubscriptionBinding) ratelimit.Limits {
	if sub == nil {
		return ratelimit.Limits{}
	}
	return ratelimit.Limits{
		RequestsPerDay: sub.DailyRequestLimit,
		// Reuse MonthlyTokenLimit field slot via custom struct — use RequestsPerDay only for daily,
		// monthly requests handled separately in proxy via MonthlyRequestLimit on binding.
	}
}

// SubscriptionScope returns the in-memory counter scope for a subscription instance.
func SubscriptionScope(userSubscriptionID string) string {
	return "sub:" + strings.TrimSpace(userSubscriptionID)
}