// Package auth provides Bearer token authentication middleware and RBAC
// enforcement for VoidLLM's proxy and admin APIs.
package auth

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/cache"
	"github.com/voidmind-io/voidllm/pkg/keygen"
)

// KeyInfo holds the authenticated identity and limits associated with an API key.
type KeyInfo struct {
	// ID is the unique identifier of the API key record.
	ID string
	// KeyType is the category of the key: keygen.KeyTypeUser, KeyTypeTeam, or KeyTypeSA.
	KeyType string
	// Role is the RBAC role assigned to this key: RoleSystemAdmin, RoleOrgAdmin, RoleTeamAdmin, or RoleMember.
	Role string
	// UserID is the user this key belongs to.
	UserID string
	// Name is the human-readable label for the key.
	Name string
	// DailyTokenLimit is the maximum number of tokens allowed per day. Zero means unlimited.
	DailyTokenLimit int64
	// MonthlyTokenLimit is the maximum number of tokens allowed per month. Zero means unlimited.
	MonthlyTokenLimit int64
	// RequestsPerMinute is the maximum number of requests allowed per minute. Zero means unlimited.
	RequestsPerMinute int
	// RequestsPerDay is the maximum number of requests allowed per day. Zero means unlimited.
	RequestsPerDay int
	// ExpiresAt is the expiration time of the key. Nil means no expiration.
	ExpiresAt *time.Time
}

// Middleware returns a Fiber handler that authenticates requests via Bearer token.
// It extracts the token, computes HMAC-SHA256 with hmacSecret, looks up the hash
// in keyCache, checks expiration, and stores the KeyInfo in the request context.
func Middleware(keyCache *cache.Cache[string, KeyInfo], hmacSecret []byte) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth := c.Get("Authorization")
		token := extractBearerToken(auth)
		if token == "" {
			return apierror.Unauthorized(c, "missing authorization header")
		}

		if _, err := keygen.ValidatePrefix(token); err != nil {
			return apierror.Unauthorized(c, "invalid API key format")
		}

		// Hash is used as a cache map key. Map lookup is not constant-time, but
		// the hash is HMAC-SHA256 with a server-side secret, so an attacker cannot
		// control the hash value to exploit timing differences.
		hash := keygen.Hash(token, hmacSecret)
		keyInfo, ok := keyCache.Get(hash)
		if !ok {
			return apierror.Unauthorized(c, "invalid API key")
		}

		if keyInfo.ExpiresAt != nil && time.Now().After(*keyInfo.ExpiresAt) {
			keyCache.Delete(hash)
			return apierror.Unauthorized(c, "invalid API key")
		}

		c.Locals(keyInfoKey, &keyInfo)
		return c.Next()
	}
}

// extractBearerToken parses a Bearer token from an Authorization header value.
// Returns the token string, or empty string if the header is absent or malformed.
func extractBearerToken(header string) string {
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		return ""
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	return token
}
