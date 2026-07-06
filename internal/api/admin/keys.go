package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/internal/keys"
	voidredis "github.com/jukaza/tavo/internal/redis"
	"github.com/jukaza/tavo/pkg/keygen"
)

// createAPIKeyRequest is the JSON body accepted by CreateAPIKey.
type createAPIKeyRequest struct {
	Name               string   `json:"name"`
	KeyType            string   `json:"key_type"`
	UserID             *string  `json:"user_id"`
	DailyTokenLimit    int64    `json:"daily_token_limit"`
	MonthlyTokenLimit  int64    `json:"monthly_token_limit"`
	RequestsPerMinute  int      `json:"requests_per_minute"`
	RequestsPerDay     int      `json:"requests_per_day"`
	Status             string   `json:"status"`
	SpendCap           float64  `json:"spend_cap"`
	IPWhitelist        []string `json:"ip_whitelist"`
	IPBlacklist        []string `json:"ip_blacklist"`
	ModelLimitsEnabled bool     `json:"model_limits_enabled"`
	ModelLimits        []string `json:"model_limits"`
	ExpiresAt          *string  `json:"expires_at"`
}

// updateAPIKeyRequest is the JSON body accepted by UpdateAPIKey.
// All fields are optional; a nil pointer means the field is left unchanged.
type updateAPIKeyRequest struct {
	Name               *string   `json:"name"`
	DailyTokenLimit    *int64    `json:"daily_token_limit"`
	MonthlyTokenLimit  *int64    `json:"monthly_token_limit"`
	RequestsPerMinute  *int      `json:"requests_per_minute"`
	RequestsPerDay     *int      `json:"requests_per_day"`
	Status             *string   `json:"status"`
	SpendCap           *float64  `json:"spend_cap"`
	SpendUsed          *float64  `json:"spend_used"`
	IPWhitelist        *[]string `json:"ip_whitelist"`
	IPBlacklist        *[]string `json:"ip_blacklist"`
	ModelLimitsEnabled *bool     `json:"model_limits_enabled"`
	ModelLimits        *[]string `json:"model_limits"`
	ExpiresAt          *string   `json:"expires_at"`
}

// createAPIKeyResponse is the JSON representation returned by CreateAPIKey.
// It includes the plaintext key, which is shown exactly once at creation time.
type createAPIKeyResponse struct {
	ID                 string   `json:"id"`
	Key                string   `json:"key"`
	KeyHint            string   `json:"key_hint"`
	KeyType            string   `json:"key_type"`
	Name               string   `json:"name"`
	UserID             *string  `json:"user_id,omitempty"`
	DailyTokenLimit    int64    `json:"daily_token_limit"`
	MonthlyTokenLimit  int64    `json:"monthly_token_limit"`
	RequestsPerMinute  int      `json:"requests_per_minute"`
	RequestsPerDay       int      `json:"requests_per_day"`
	Status             string   `json:"status"`
	SpendCap           float64  `json:"spend_cap"`
	SpendUsed          float64  `json:"spend_used"`
	IPWhitelist        []string `json:"ip_whitelist,omitempty"`
	IPBlacklist        []string `json:"ip_blacklist,omitempty"`
	ModelLimitsEnabled bool     `json:"model_limits_enabled"`
	ModelLimits        []string `json:"model_limits,omitempty"`
	ExpiresAt          *string  `json:"expires_at,omitempty"`
	CreatedBy          string   `json:"created_by"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

// paginatedAPIKeysResponse wraps a page of API keys with pagination metadata.
type paginatedAPIKeysResponse struct {
	Data    []apiKeyResponse `json:"data"`
	HasMore bool             `json:"has_more"`
	Cursor  string           `json:"next_cursor,omitempty"`
}

// validKeyTypes is the set of accepted key_type values.
var validKeyTypes = map[string]bool{
	keygen.KeyTypeUser: true,
}

// derefStr returns the string value pointed to by s, or "" if s is nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// apiKeyVisibleToCallerKey reports whether the given API key is within the visible
// scope of the caller.
func apiKeyVisibleToCallerKey(key *db.APIKey, caller *auth.KeyInfo) bool {
	if auth.HasRole(caller.Role, auth.RoleSystemAdmin) {
		return true
	}
	return key.UserID != nil && *key.UserID == caller.UserID
}

func buildCreateAPIKeyResponse(apiKey *db.APIKey, plaintextKey string) createAPIKeyResponse {
	base := buildAPIKeyResponse(apiKey)
	return createAPIKeyResponse{
		ID:                 base.ID,
		Key:                plaintextKey,
		KeyHint:            base.KeyHint,
		KeyType:            base.KeyType,
		Name:               base.Name,
		UserID:             base.UserID,
		DailyTokenLimit:    base.DailyTokenLimit,
		MonthlyTokenLimit:  base.MonthlyTokenLimit,
		RequestsPerMinute:  base.RequestsPerMinute,
		RequestsPerDay:     base.RequestsPerDay,
		Status:             base.Status,
		SpendCap:           base.SpendCap,
		SpendUsed:          base.SpendUsed,
		IPWhitelist:        base.IPWhitelist,
		IPBlacklist:        base.IPBlacklist,
		ModelLimitsEnabled: base.ModelLimitsEnabled,
		ModelLimits:        base.ModelLimits,
		ExpiresAt:          base.ExpiresAt,
		CreatedBy:          base.CreatedBy,
		CreatedAt:          base.CreatedAt,
		UpdatedAt:          base.UpdatedAt,
	}
}

func (h *Handler) currentKeysPolicy(ctx context.Context) keys.Policy {
	if h.KeysRuntime != nil {
		return h.KeysRuntime.Get()
	}
	policy, err := keys.LoadPolicy(ctx, h.DB)
	if err != nil {
		h.Log.ErrorContext(ctx, "keys policy: load", slog.String("error", err.Error()))
		return keys.DefaultPolicy()
	}
	return policy
}

func (h *Handler) createAPIKeyFromRequest(ctx context.Context, keyInfo *auth.KeyInfo, req *createAPIKeyRequest) (createAPIKeyResponse, error) {
	if keyInfo.UserID == "" {
		return createAPIKeyResponse{}, &batchItemError{msg: "keys can only be created by user keys"}
	}
	if req.Name == "" {
		return createAPIKeyResponse{}, &batchItemError{msg: "name is required"}
	}
	req.KeyType = keygen.KeyTypeUser

	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		req.UserID = &keyInfo.UserID
	}

	if req.UserID != nil && *req.UserID != "" {
		if _, err := h.DB.GetUser(ctx, *req.UserID); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				return createAPIKeyResponse{}, &batchItemError{msg: "user not found"}
			}
			return createAPIKeyResponse{}, fmt.Errorf("get user: %w", err)
		}
	} else {
		req.UserID = &keyInfo.UserID
	}

	policy := h.currentKeysPolicy(ctx)
	count, err := h.DB.CountUserAPIKeys(ctx, *req.UserID)
	if err != nil {
		return createAPIKeyResponse{}, fmt.Errorf("count user api keys: %w", err)
	}
	if count >= policy.MaxPerUser {
		return createAPIKeyResponse{}, &batchItemError{msg: fmt.Sprintf("maximum of %d api keys per user reached", policy.MaxPerUser)}
	}

	status := req.Status
	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		if status != "" && status != keys.StatusActive {
			return createAPIKeyResponse{}, &batchItemError{msg: "members may only create active keys"}
		}
		status = keys.StatusActive
	} else if status == "" {
		status = keys.StatusActive
	} else if !keys.ValidStatus(status) {
		return createAPIKeyResponse{}, &batchItemError{msg: "invalid status"}
	}

	plaintextKey, err := keygen.Generate(req.KeyType)
	if err != nil {
		return createAPIKeyResponse{}, fmt.Errorf("generate key: %w", err)
	}

	keyHash := keygen.Hash(plaintextKey, h.HMACSecret)
	keyHint := keygen.Hint(plaintextKey)

	keyID, err := uuid.NewV7()
	if err != nil {
		return createAPIKeyResponse{}, fmt.Errorf("generate key id: %w", err)
	}
	keyEnc, err := h.encryptStoredAPIKey(keyID.String(), plaintextKey)
	if err != nil {
		return createAPIKeyResponse{}, err
	}

	apiKey, err := h.DB.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		ID:                 keyID.String(),
		KeyEncrypted:       keyEnc,
		KeyHash:            keyHash,
		KeyHint:            keyHint,
		KeyType:            req.KeyType,
		Name:               req.Name,
		UserID:             req.UserID,
		DailyTokenLimit:    req.DailyTokenLimit,
		MonthlyTokenLimit:  req.MonthlyTokenLimit,
		RequestsPerMinute:  req.RequestsPerMinute,
		RequestsPerDay:     req.RequestsPerDay,
		Status:             status,
		SpendCap:           req.SpendCap,
		IPWhitelist:        keys.FormatIPRules(req.IPWhitelist),
		IPBlacklist:        keys.FormatIPRules(req.IPBlacklist),
		ModelLimitsEnabled: req.ModelLimitsEnabled,
		ModelLimits:        formatModelLimitsForDB(req.ModelLimits),
		ExpiresAt:          req.ExpiresAt,
		CreatedBy:          keyInfo.UserID,
	})
	if err != nil {
		return createAPIKeyResponse{}, fmt.Errorf("db insert: %w", err)
	}

	resolvedRole := auth.RoleMember
	if req.UserID != nil {
		role, roleErr := h.DB.GetUserRole(ctx, *req.UserID)
		if roleErr == nil {
			resolvedRole = role
		}
	}

	h.KeyCache.Set(apiKey.KeyHash, keyInfoFromAPIKey(apiKey, resolvedRole))
	return buildCreateAPIKeyResponse(apiKey, plaintextKey), nil
}

// CreateAPIKey handles POST /api/v1/keys.
func (h *Handler) CreateAPIKey(c fiber.Ctx) error {
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	var req createAPIKeyRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	resp, err := h.createAPIKeyFromRequest(c.Context(), keyInfo, &req)
	if err != nil {
		var apiErr *batchItemError
		if errors.As(err, &apiErr) {
			return apierror.BadRequest(c, apiErr.msg)
		}
		h.Log.ErrorContext(c.Context(), "create api key", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to create api key")
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// GetAPIKey handles GET /api/v1/keys/:key_id.
func (h *Handler) GetAPIKey(c fiber.Ctx) error {
	keyID := c.Params("key_id")

	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	apiKey, err := h.DB.GetAPIKey(c.Context(), keyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "api key not found")
		}
		h.Log.ErrorContext(c.Context(), "get api key", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to get api key")
	}

	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		if !apiKeyVisibleToCallerKey(apiKey, keyInfo) {
			return apierror.NotFound(c, "api key not found")
		}
	}

	return c.JSON(buildAPIKeyResponse(apiKey))
}

func (h *Handler) ListAPIKeys(c fiber.Ctx) error {
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	p, err := parsePagination(c)
	if err != nil {
		return apierror.BadRequest(c, err.Error())
	}
	includeDeleted := c.Query("include_deleted") == "true" && auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin)

	var filterUserID string
	if auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		filterUserID = c.Query("user_id")
	} else {
		filterUserID = keyInfo.UserID
	}

	keysList, err := h.DB.ListAPIKeys(c.Context(), filterUserID, p.Cursor, p.Limit+1, includeDeleted)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "list api keys", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to list api keys")
	}

	hasMore := len(keysList) > p.Limit
	if hasMore {
		keysList = keysList[:p.Limit]
	}

	resp := paginatedAPIKeysResponse{
		Data:    make([]apiKeyResponse, len(keysList)),
		HasMore: hasMore,
	}
	for i := range keysList {
		resp.Data[i] = buildAPIKeyResponse(&keysList[i])
	}
	if hasMore && len(keysList) > 0 {
		resp.Cursor = keysList[len(keysList)-1].ID
	}
	return c.JSON(resp)
}

func (h *Handler) UpdateAPIKey(c fiber.Ctx) error {
	keyID := c.Params("key_id")

	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	existing, err := h.DB.GetAPIKey(c.Context(), keyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "api key not found")
		}
		h.Log.ErrorContext(c.Context(), "update api key: get", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to get api key")
	}

	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		if !apiKeyVisibleToCallerKey(existing, keyInfo) {
			return apierror.NotFound(c, "api key not found")
		}
	}

	var req updateAPIKeyRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	if req.Status != nil && !keys.ValidStatus(*req.Status) {
		return apierror.BadRequest(c, "invalid status")
	}

	params := db.UpdateAPIKeyParams{
		Name:               req.Name,
		DailyTokenLimit:    req.DailyTokenLimit,
		MonthlyTokenLimit:  req.MonthlyTokenLimit,
		RequestsPerMinute:  req.RequestsPerMinute,
		RequestsPerDay:     req.RequestsPerDay,
		Status:             req.Status,
		SpendCap:           req.SpendCap,
		SpendUsed:          req.SpendUsed,
		ExpiresAt:          req.ExpiresAt,
	}
	if req.IPWhitelist != nil {
		formatted := keys.FormatIPRules(*req.IPWhitelist)
		params.IPWhitelist = &formatted
	}
	if req.IPBlacklist != nil {
		formatted := keys.FormatIPRules(*req.IPBlacklist)
		params.IPBlacklist = &formatted
	}
	if req.ModelLimitsEnabled != nil {
		params.ModelLimitsEnabled = req.ModelLimitsEnabled
	}
	if req.ModelLimits != nil {
		formatted := formatModelLimitsForDB(*req.ModelLimits)
		params.ModelLimits = &formatted
	}

	apiKey, err := h.DB.UpdateAPIKey(c.Context(), existing.ID, params)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "api key not found")
		}
		h.Log.ErrorContext(c.Context(), "update api key", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to update api key")
	}

	if cached, ok := h.KeyCache.Get(existing.KeyHash); ok {
		updated := keyInfoFromAPIKey(apiKey, cached.Role)
		h.KeyCache.Set(existing.KeyHash, updated)
	}

	return c.JSON(buildAPIKeyResponse(apiKey))
}

// rotateKeyGracePeriod is the duration an old key remains valid after rotation.
const rotateKeyGracePeriod = 24 * time.Hour

type rotateKeyResponse struct {
	NewKey rotatedKeyInfo `json:"new_key"`
	OldKey rotatedKeyInfo `json:"old_key"`
}

type rotatedKeyInfo struct {
	ID        string  `json:"id"`
	Key       string  `json:"key,omitempty"`
	Hint      string  `json:"hint"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

func (h *Handler) RotateAPIKey(c fiber.Ctx) error {
	keyID := c.Params("key_id")

	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	ctx := c.Context()

	existing, err := h.DB.GetAPIKey(ctx, keyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "api key not found")
		}
		h.Log.ErrorContext(ctx, "rotate api key: get", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to get api key")
	}

	if existing.KeyType != keygen.KeyTypeUser {
		return apierror.BadRequest(c, "only user keys can be rotated")
	}

	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		if !apiKeyVisibleToCallerKey(existing, keyInfo) {
			return apierror.Send(c, fiber.StatusForbidden, "forbidden", "you can only rotate keys within your scope")
		}
	}

	plaintextKey, err := keygen.Generate(existing.KeyType)
	if err != nil {
		h.Log.ErrorContext(ctx, "rotate api key: generate", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to generate key")
	}

	keyHash := keygen.Hash(plaintextKey, h.HMACSecret)
	keyHint := keygen.Hint(plaintextKey)

	newKeyID, err := uuid.NewV7()
	if err != nil {
		h.Log.ErrorContext(ctx, "rotate api key: generate id", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to rotate key")
	}
	keyEnc, err := h.encryptStoredAPIKey(newKeyID.String(), plaintextKey)
	if err != nil {
		h.Log.ErrorContext(ctx, "rotate api key: encrypt", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to rotate key")
	}

	rotatedName := strings.TrimSuffix(existing.Name, " (rotated)") + " (rotated)"

	graceDeadline := time.Now().UTC().Add(rotateKeyGracePeriod)
	oldExpiresAt := graceDeadline.Format(time.RFC3339)
	if existing.ExpiresAt != nil {
		if t, parseErr := time.Parse(time.RFC3339, *existing.ExpiresAt); parseErr == nil && t.Before(graceDeadline) {
			oldExpiresAt = *existing.ExpiresAt
		}
	}

	status := existing.Status
	if status == "" {
		status = keys.StatusActive
	}

	rotated, err := h.DB.RotateKeyTx(ctx, existing.ID, oldExpiresAt, db.CreateAPIKeyParams{
		ID:                 newKeyID.String(),
		KeyEncrypted:       keyEnc,
		KeyHash:            keyHash,
		KeyHint:            keyHint,
		KeyType:            existing.KeyType,
		Name:               rotatedName,
		UserID:             existing.UserID,
		DailyTokenLimit:    existing.DailyTokenLimit,
		MonthlyTokenLimit:  existing.MonthlyTokenLimit,
		RequestsPerMinute:  existing.RequestsPerMinute,
		RequestsPerDay:     existing.RequestsPerDay,
		Status:             status,
		SpendCap:           existing.SpendCap,
		IPWhitelist:        existing.IPWhitelist,
		IPBlacklist:        existing.IPBlacklist,
		ModelLimitsEnabled: existing.ModelLimitsEnabled,
		ModelLimits:        existing.ModelLimits,
		ExpiresAt:          existing.ExpiresAt,
		CreatedBy:          keyInfo.UserID,
	})
	if err != nil {
		h.Log.ErrorContext(ctx, "rotate api key: rotate tx", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to rotate key")
	}
	newKey := rotated.NewKey
	updatedOld := rotated.OldKey

	resolvedRole := auth.RoleMember
	if existing.UserID != nil {
		role, roleErr := h.DB.GetUserRole(ctx, *existing.UserID)
		if roleErr == nil {
			resolvedRole = role
		}
	}

	h.KeyCache.Set(newKey.KeyHash, keyInfoFromAPIKey(newKey, resolvedRole))

	if cached, hit := h.KeyCache.Get(existing.KeyHash); hit {
		if t, parseErr := time.Parse(time.RFC3339, oldExpiresAt); parseErr == nil {
			cached.ExpiresAt = &t
			h.KeyCache.Set(existing.KeyHash, cached)
		}
	}

	if h.Redis != nil {
		if err := h.Redis.PublishInvalidation(ctx, voidredis.ChannelKeys, existing.KeyHash); err != nil {
			h.Log.LogAttrs(ctx, slog.LevelWarn, "redis: publish key invalidation failed",
				slog.String("error", err.Error()),
			)
		}
	}

	resp := rotateKeyResponse{
		NewKey: rotatedKeyInfo{
			ID:        newKey.ID,
			Key:       plaintextKey,
			Hint:      newKey.KeyHint,
			ExpiresAt: newKey.ExpiresAt,
		},
		OldKey: rotatedKeyInfo{
			ID:        updatedOld.ID,
			Hint:      updatedOld.KeyHint,
			ExpiresAt: updatedOld.ExpiresAt,
		},
	}
	return c.JSON(resp)
}

func (h *Handler) DeleteAPIKey(c fiber.Ctx) error {
	keyID := c.Params("key_id")

	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	apiKey, err := h.DB.GetAPIKey(c.Context(), keyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "api key not found")
		}
		h.Log.ErrorContext(c.Context(), "delete api key: get", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to get api key")
	}

	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		if apiKey.UserID == nil || *apiKey.UserID != keyInfo.UserID {
			return apierror.Send(c, fiber.StatusForbidden, "forbidden", "you can only delete your own keys")
		}
	}

	if err := h.DB.DeleteAPIKey(c.Context(), apiKey.ID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "api key not found")
		}
		h.Log.ErrorContext(c.Context(), "delete api key", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to delete api key")
	}

	h.KeyCache.Delete(apiKey.KeyHash)

	if h.Redis != nil {
		if err := h.Redis.PublishInvalidation(c.Context(), voidredis.ChannelKeys, apiKey.KeyHash); err != nil {
			h.Log.LogAttrs(c.Context(), slog.LevelWarn, "redis: publish key invalidation failed",
				slog.String("error", err.Error()),
			)
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}