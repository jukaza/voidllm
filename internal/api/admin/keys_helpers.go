package admin

import (
	"time"

	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/internal/keys"
)

// apiKeyResponse is the JSON representation of an API key returned by GET and LIST operations.
// It never includes the plaintext key or the key_hash.
type apiKeyResponse struct {
	ID                 string   `json:"id"`
	KeyHint            string   `json:"key_hint"`
	KeyType            string   `json:"key_type"`
	Name               string   `json:"name"`
	UserID             *string  `json:"user_id,omitempty"`
	DailyTokenLimit    int64    `json:"daily_token_limit"`
	MonthlyTokenLimit  int64    `json:"monthly_token_limit"`
	RequestsPerMinute  int      `json:"requests_per_minute"`
	RequestsPerDay     int      `json:"requests_per_day"`
	Status             string   `json:"status"`
	SpendCap           float64  `json:"spend_cap"`
	SpendUsed          float64  `json:"spend_used"`
	IPWhitelist        []string `json:"ip_whitelist,omitempty"`
	IPBlacklist        []string `json:"ip_blacklist,omitempty"`
	ModelLimitsEnabled bool     `json:"model_limits_enabled"`
	ModelLimits        []string `json:"model_limits,omitempty"`
	ExpiresAt          *string  `json:"expires_at,omitempty"`
	LastUsedAt         *string  `json:"last_used_at,omitempty"`
	CreatedBy          string   `json:"created_by"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

// buildAPIKeyResponse converts a db.APIKey to its API wire representation.
func buildAPIKeyResponse(k *db.APIKey) apiKeyResponse {
	status := k.Status
	if status == "" {
		status = keys.StatusActive
	}
	return apiKeyResponse{
		ID:                 k.ID,
		KeyHint:            k.KeyHint,
		KeyType:            k.KeyType,
		Name:               k.Name,
		UserID:             k.UserID,
		DailyTokenLimit:    k.DailyTokenLimit,
		MonthlyTokenLimit:  k.MonthlyTokenLimit,
		RequestsPerMinute:  k.RequestsPerMinute,
		RequestsPerDay:     k.RequestsPerDay,
		Status:             status,
		SpendCap:           k.SpendCap,
		SpendUsed:          k.SpendUsed,
		IPWhitelist:        keys.ParseIPRules(k.IPWhitelist),
		IPBlacklist:        keys.ParseIPRules(k.IPBlacklist),
		ModelLimitsEnabled: k.ModelLimitsEnabled,
		ModelLimits:        keys.ParseModelLimits(k.ModelLimits),
		ExpiresAt:          k.ExpiresAt,
		LastUsedAt:         k.LastUsedAt,
		CreatedBy:          k.CreatedBy,
		CreatedAt:          k.CreatedAt,
		UpdatedAt:          k.UpdatedAt,
	}
}

// keyInfoFromAPIKey builds an auth.KeyInfo for cache population from a DB record.
func keyInfoFromAPIKey(k *db.APIKey, role string) auth.KeyInfo {
	status := k.Status
	if status == "" {
		status = keys.StatusActive
	}
	ki := auth.KeyInfo{
		ID:                 k.ID,
		KeyType:            k.KeyType,
		Role:               role,
		UserID:             derefStr(k.UserID),
		Name:               k.Name,
		DailyTokenLimit:    k.DailyTokenLimit,
		MonthlyTokenLimit:  k.MonthlyTokenLimit,
		RequestsPerMinute:  k.RequestsPerMinute,
		RequestsPerDay:     k.RequestsPerDay,
		Status:             status,
		SpendCap:           k.SpendCap,
		SpendUsed:          k.SpendUsed,
		IPWhitelist:        k.IPWhitelist,
		IPBlacklist:        k.IPBlacklist,
		ModelLimitsEnabled: k.ModelLimitsEnabled,
		ModelLimits:        k.ModelLimits,
	}
	if k.ExpiresAt != nil {
		if t, err := time.Parse(time.RFC3339, *k.ExpiresAt); err == nil {
			ki.ExpiresAt = &t
		}
	}
	return ki
}

// parseModelLimitsInput normalizes model limit slices from API input.
func parseModelLimitsInput(models []string) []string {
	if len(models) == 0 {
		return nil
	}
	return keys.ParseModelLimits(keys.FormatModelLimits(models))
}

// formatModelLimitsForDB encodes model names for persistence.
func formatModelLimitsForDB(models []string) string {
	return keys.FormatModelLimits(parseModelLimitsInput(models))
}