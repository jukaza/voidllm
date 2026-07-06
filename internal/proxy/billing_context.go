package proxy

import (
	"github.com/gofiber/fiber/v3"
	subsvc "github.com/jukaza/tavo/internal/subscription"
)

const (
	billingPathKey      = "billing_path"
	subscriptionIDKey   = "subscription_id"
)

func billingPathFromCtx(c fiber.Ctx) subsvc.BillingPath {
	if c == nil {
		return subsvc.BillingWallet
	}
	v := c.Locals(billingPathKey)
	if s, ok := v.(string); ok && s == string(subsvc.BillingSubscription) {
		return subsvc.BillingSubscription
	}
	return subsvc.BillingWallet
}

func subscriptionIDFromCtx(c fiber.Ctx) string {
	if c == nil {
		return ""
	}
	v := c.Locals(subscriptionIDKey)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}