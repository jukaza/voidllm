package ratelimit

// Checker evaluates request rate limits. Implementations must be safe for
// concurrent use. A nil Checker disables rate limiting entirely.
type Checker interface {
	CheckRate(keyID string, keyLimits Limits) error
}