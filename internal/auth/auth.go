// Package auth provides Bearer token authentication middleware and RBAC
// enforcement for Tavo's proxy and admin APIs.
package auth

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/cache"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/pkg/keygen"
)

// KeyInfo holds the authenticated identity and limits associated with an API key.
type KeyInfo struct {
	// ID is the unique identifier of the API key record.
	ID string
	// KeyType is the category of the key: keygen.KeyTypeUser or KeyTypeSession.
	KeyType string
	// Role is the RBAC role assigned to this key: RoleSystemAdmin or RoleMember.
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
	// Status is the key lifecycle state: active, disabled, expired, quota_exhausted.
	Status string
	// SpendCap is the maximum spend allowed on this key (0 = unlimited).
	SpendCap float64
	// SpendUsed is the cumulative spend charged to this key.
	SpendUsed float64
	// IPWhitelist and IPBlacklist are newline-separated IP/CIDR rules.
	IPWhitelist string
	IPBlacklist string
	// ModelLimitsEnabled gates requests to ModelLimits when true.
	ModelLimitsEnabled bool
	// ModelLimits is a JSON array of allowed model names.
	ModelLimits string
}

// Middleware returns a Fiber handler that authenticates requests via Bearer token.
// It extracts the token, computes HMAC-SHA256 with hmacSecret, looks up the hash
// in keyCache, checks expiration, and stores the KeyInfo in the request context.
func Middleware(keyCache *cache.Cache[string, KeyInfo], hmacSecret []byte, database ...*db.DB) fiber.Handler {
	var lookupDB *db.DB
	if len(database) > 0 {
		lookupDB = database[0]
	}
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
		if !ok && lookupDB != nil {
			record, err := lookupDB.LookupActiveKeyByHash(c.Context(), hash)
			if err == nil {
				keyInfo = keyInfoFromRecord(record)
				keyCache.Set(hash, keyInfo)
				ok = true
			}
		}
		if !ok {
			return apierror.Unauthorized(c, "invalid API key")
		}

		if keyInfo.ExpiresAt != nil && time.Now().After(*keyInfo.ExpiresAt) {
			keyCache.Delete(hash)
			return apierror.Unauthorized(c, "invalid API key")
		}
		if keyInfo.KeyType != keygen.KeyTypeSession {
			switch keyInfo.Status {
			case "disabled", "expired", "quota_exhausted":
				return apierror.Send(c, fiber.StatusForbidden, "key_disabled", "api key is disabled")
			case "", "active":
			default:
				return apierror.Send(c, fiber.StatusForbidden, "key_disabled", "api key is disabled")
			}
		}

		c.Locals(keyInfoKey, &keyInfo)
		return c.Next()
	}
}

func keyInfoFromRecord(r db.KeyRecord) KeyInfo {
	status := r.Status
	if status == "" {
		status = "active"
	}
	ki := KeyInfo{
		ID:                 r.ID,
		KeyType:            r.KeyType,
		Name:               r.Name,
		DailyTokenLimit:    r.DailyTokenLimit,
		MonthlyTokenLimit:  r.MonthlyTokenLimit,
		RequestsPerMinute:  r.RequestsPerMinute,
		RequestsPerDay:     r.RequestsPerDay,
		ExpiresAt:          r.ExpiresAt,
		Status:             status,
		SpendCap:           r.SpendCap,
		SpendUsed:          r.SpendUsed,
		IPWhitelist:        r.IPWhitelist,
		IPBlacklist:        r.IPBlacklist,
		ModelLimitsEnabled: r.ModelLimitsEnabled,
		ModelLimits:        r.ModelLimits,
	}
	if r.UserID != nil {
		ki.UserID = *r.UserID
	}
	if r.IsSystemAdmin == 1 {
		ki.Role = RoleSystemAdmin
	} else {
		ki.Role = RoleMember
	}
	return ki
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
