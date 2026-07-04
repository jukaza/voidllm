package upstream

import (
	"strconv"
	"strings"
	"time"
)

const (
	BackoffBaseMS           = 2000
	BackoffMaxMS            = 5 * 60 * 1000
	BackoffMaxLevel         = 15
	TransientCooldownMS     = 30 * 1000
	MaxRateLimitCooldownMS  = 30 * 60 * 1000
)

type errorRule struct {
	text     string
	status   int
	cooldown time.Duration
	backoff  bool
}

var errorRules = []errorRule{
	{text: "no credentials", cooldown: 2 * time.Minute},
	{text: "request not allowed", cooldown: 5 * time.Second},
	{text: "improperly formed request", cooldown: 2 * time.Minute},
	{text: "rate limit", backoff: true},
	{text: "too many requests", backoff: true},
	{text: "quota exceeded", backoff: true},
	{text: "capacity", backoff: true},
	{text: "overloaded", backoff: true},
	{status: 401, cooldown: 2 * time.Minute},
	{status: 402, cooldown: 2 * time.Minute},
	{status: 403, cooldown: 2 * time.Minute},
	{status: 404, cooldown: 2 * time.Minute},
	{status: 429, backoff: true},
}

// ClassifyResult describes how to handle an upstream failure for key rotation.
type ClassifyResult struct {
	ShouldRotate    bool
	Cooldown        time.Duration
	NewBackoffLevel int
}

func quotaCooldown(backoffLevel int) time.Duration {
	level := backoffLevel
	if level > 0 {
		level--
	}
	ms := BackoffBaseMS * (1 << level)
	if ms > BackoffMaxMS {
		ms = BackoffMaxMS
	}
	return time.Duration(ms) * time.Millisecond
}

// ClassifyError maps HTTP status + body text to rotation/cooldown behavior.
func ClassifyError(status int, errorText string, backoffLevel int) ClassifyResult {
	lower := strings.ToLower(errorText)
	for _, rule := range errorRules {
		if rule.text != "" && lower != "" && strings.Contains(lower, rule.text) {
			if rule.backoff {
				newLevel := backoffLevel + 1
				if newLevel > BackoffMaxLevel {
					newLevel = BackoffMaxLevel
				}
				return ClassifyResult{ShouldRotate: true, Cooldown: quotaCooldown(newLevel), NewBackoffLevel: newLevel}
			}
			return ClassifyResult{ShouldRotate: true, Cooldown: rule.cooldown}
		}
		if rule.status != 0 && rule.status == status {
			if rule.backoff {
				newLevel := backoffLevel + 1
				if newLevel > BackoffMaxLevel {
					newLevel = BackoffMaxLevel
				}
				return ClassifyResult{ShouldRotate: true, Cooldown: quotaCooldown(newLevel), NewBackoffLevel: newLevel}
			}
			return ClassifyResult{ShouldRotate: true, Cooldown: rule.cooldown}
		}
	}
	if status == 503 || status == 502 || status == 504 {
		return ClassifyResult{ShouldRotate: true, Cooldown: TransientCooldownMS * time.Millisecond}
	}
	if status >= 500 {
		return ClassifyResult{ShouldRotate: true, Cooldown: TransientCooldownMS * time.Millisecond}
	}
	return ClassifyResult{ShouldRotate: false}
}

// ParseRetryAfter converts Retry-After header (seconds or HTTP-date) to duration.
func ParseRetryAfter(header string) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}
	if sec, err := strconv.Atoi(header); err == nil && sec > 0 {
		d := time.Duration(sec) * time.Second
		if d > MaxRateLimitCooldownMS*time.Millisecond {
			return MaxRateLimitCooldownMS * time.Millisecond
		}
		return d
	}
	if t, err := time.Parse(time.RFC1123, header); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		if d > MaxRateLimitCooldownMS*time.Millisecond {
			return MaxRateLimitCooldownMS * time.Millisecond
		}
		return d
	}
	return 0
}

// IsModelLockActive reports whether upstreamModel is locked on locks map.
func IsModelLockActive(locks map[string]string, upstreamModel string, now time.Time) bool {
	if until, ok := locks[upstreamModel]; ok && until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil && t.After(now) {
			return true
		}
	}
	if until, ok := locks["__all__"]; ok && until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil && t.After(now) {
			return true
		}
	}
	return false
}

// EarliestLockUntil returns the soonest active lock expiry ISO string.
func EarliestLockUntil(locks map[string]string, accountUntil *string, now time.Time) *string {
	var earliest *time.Time
	check := func(raw string) {
		if raw == "" {
			return
		}
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil || !t.After(now) {
			return
		}
		if earliest == nil || t.Before(*earliest) {
			earliest = &t
		}
	}
	for _, v := range locks {
		check(v)
	}
	if accountUntil != nil {
		check(*accountUntil)
	}
	if earliest == nil {
		return nil
	}
	s := earliest.UTC().Format(time.RFC3339)
	return &s
}