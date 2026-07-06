package proxy

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/keys"
	"github.com/jukaza/tavo/internal/ratelimit"
)

type proxyUsageLimits struct {
	RequestsPerMinute int     `json:"requests_per_minute"`
	RequestsPerDay    int     `json:"requests_per_day"`
	DailyTokenLimit   int64   `json:"daily_token_limit"`
	MonthlyTokenLimit int64   `json:"monthly_token_limit"`
	SpendCap          float64 `json:"spend_cap"`
}

type proxyUsageCounters struct {
	RequestsPerMinute int64   `json:"requests_per_minute"`
	RequestsPerDay    int64   `json:"requests_per_day"`
	DailyTokens       int64   `json:"daily_tokens"`
	MonthlyTokens     int64   `json:"monthly_tokens"`
	SpendUsed         float64 `json:"spend_used"`
}

type proxyUsageResponse struct {
	KeyID   string             `json:"key_id"`
	Status  string             `json:"status"`
	Limits  proxyUsageLimits   `json:"limits"`
	Usage   proxyUsageCounters `json:"usage"`
}

// UsageHandler handles GET /v1/usage — read-only usage for the authenticated API key.
func (p *ProxyHandler) UsageHandler(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return apierror.Unauthorized(c, "missing authentication")
	}

	status := keyInfo.Status
	if status == "" {
		status = keys.StatusActive
	}

	resp := proxyUsageResponse{
		KeyID:  keyInfo.ID,
		Status: status,
		Limits: proxyUsageLimits{
			RequestsPerMinute: keyInfo.RequestsPerMinute,
			RequestsPerDay:    keyInfo.RequestsPerDay,
			DailyTokenLimit:   keyInfo.DailyTokenLimit,
			MonthlyTokenLimit: keyInfo.MonthlyTokenLimit,
			SpendCap:          keyInfo.SpendCap,
		},
		Usage: proxyUsageCounters{
			SpendUsed: keyInfo.SpendUsed,
		},
	}

	if rl, ok := p.rateLimiter(); ok {
		snap := rl.Snapshot(keyInfo.ID)
		resp.Usage.RequestsPerMinute = snap.RequestsPerMinute
		resp.Usage.RequestsPerDay = snap.RequestsPerDay
	}
	if p.TokenCounter != nil {
		snap := p.TokenCounter.Snapshot(keyInfo.ID)
		resp.Usage.DailyTokens = snap.DailyTokens
		resp.Usage.MonthlyTokens = snap.MonthlyTokens
	}

	return c.JSON(resp)
}

func (p *ProxyHandler) rateLimiter() (*ratelimit.RateLimiter, bool) {
	if p.RateLimiter == nil {
		return nil, false
	}
	rl, ok := p.RateLimiter.(*ratelimit.RateLimiter)
	return rl, ok
}