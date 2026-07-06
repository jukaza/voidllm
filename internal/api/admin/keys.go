package admin

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	voidredis "github.com/jukaza/tavo/internal/redis"
	"github.com/jukaza/tavo/pkg/keygen"
)

// createAPIKeyRequest is the JSON body accepted by CreateAPIKey.
type createAPIKeyRequest struct {
	Name              string  `json:"name"`
	KeyType           string  `json:"key_type"`
	UserID            *string `json:"user_id"`
	DailyTokenLimit   int64   `json:"daily_token_limit"`
	MonthlyTokenLimit int64   `json:"monthly_token_limit"`
	RequestsPerMinute int     `json:"requests_per_minute"`
	RequestsPerDay    int     `json:"requests_per_day"`
	ExpiresAt         *string `json:"expires_at"`
}

// updateAPIKeyRequest is the JSON body accepted by UpdateAPIKey.
// All fields are optional; a nil pointer means the field is left unchanged.
type updateAPIKeyRequest struct {
	Name              *string `json:"name"`
	DailyTokenLimit   *int64  `json:"daily_token_limit"`
	MonthlyTokenLimit *int64  `json:"monthly_token_limit"`
	RequestsPerMinute *int    `json:"requests_per_minute"`
	RequestsPerDay    *int    `json:"requests_per_day"`
	ExpiresAt         *string `json:"expires_at"`
}

// createAPIKeyResponse is the JSON representation returned by CreateAPIKey.
// It includes the plaintext key, which is shown exactly once at creation time.
type createAPIKeyResponse struct {
	ID                string  `json:"id"`
	Key               string  `json:"key"`
	KeyHint           string  `json:"key_hint"`
	KeyType           string  `json:"key_type"`
	Name              string  `json:"name"`
	UserID            *string `json:"user_id,omitempty"`
	DailyTokenLimit   int64   `json:"daily_token_limit"`
	MonthlyTokenLimit int64   `json:"monthly_token_limit"`
	RequestsPerMinute int     `json:"requests_per_minute"`
	RequestsPerDay    int     `json:"requests_per_day"`
	ExpiresAt         *string `json:"expires_at,omitempty"`
	CreatedBy         string  `json:"created_by"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

// apiKeyResponse is the JSON representation of an API key returned by GET and LIST operations.
// It never includes the plaintext key or the key_hash.
type apiKeyResponse struct {
	ID                string  `json:"id"`
	KeyHint           string  `json:"key_hint"`
	KeyType           string  `json:"key_type"`
	Name              string  `json:"name"`
	UserID            *string `json:"user_id,omitempty"`
	DailyTokenLimit   int64   `json:"daily_token_limit"`
	MonthlyTokenLimit int64   `json:"monthly_token_limit"`
	RequestsPerMinute int     `json:"requests_per_minute"`
	RequestsPerDay    int     `json:"requests_per_day"`
	ExpiresAt         *string `json:"expires_at,omitempty"`
	LastUsedAt        *string `json:"last_used_at,omitempty"`
	CreatedBy         string  `json:"created_by"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
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

// apiKeyToResponse converts a db.APIKey to its API wire representation.
// The key_hash field is intentionally excluded.
func apiKeyToResponse(k *db.APIKey) apiKeyResponse {
	return apiKeyResponse{
		ID:                k.ID,
		KeyHint:           k.KeyHint,
		KeyType:           k.KeyType,
		Name:              k.Name,
		UserID:            k.UserID,
		DailyTokenLimit:   k.DailyTokenLimit,
		MonthlyTokenLimit: k.MonthlyTokenLimit,
		RequestsPerMinute: k.RequestsPerMinute,
		RequestsPerDay:    k.RequestsPerDay,
		ExpiresAt:         k.ExpiresAt,
		LastUsedAt:        k.LastUsedAt,
		CreatedBy:         k.CreatedBy,
		CreatedAt:         k.CreatedAt,
		UpdatedAt:         k.UpdatedAt,
	}
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

// CreateAPIKey handles POST /api/v1/keys.
// Members may only create user_key for themselves (user_id is forced to their own ID).
// The plaintext key is returned exactly once in the response body.
//
// @Summary      Create an API key
// @Description  Creates a new API key. Members may only create user keys for themselves. The plaintext key is returned exactly once.
// @Tags         keys
// @Accept       json
// @Produce      json
// @Param        body    body      createAPIKeyRequest  true  "API key parameters"
// @Success      201     {object}  createAPIKeyResponse
// @Failure      400     {object}  swaggerErrorResponse
// @Failure      401     {object}  swaggerErrorResponse
// @Failure      403     {object}  swaggerErrorResponse
// @Failure      500     {object}  swaggerErrorResponse
// @Security     BearerAuth
// @Router       /keys [post]
func (h *Handler) CreateAPIKey(c fiber.Ctx) error {
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	if keyInfo.UserID == "" {
		return apierror.BadRequest(c, "keys can only be created by user keys")
	}

	var req createAPIKeyRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	if req.Name == "" {
		return apierror.BadRequest(c, "name is required")
	}
	req.KeyType = keygen.KeyTypeUser

	// Members may only create user_key for themselves.
	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		req.UserID = &keyInfo.UserID
	}

	ctx := c.Context()

	// Verify referenced user exists.
	if req.UserID != nil && *req.UserID != "" {
		if _, err := h.DB.GetUser(ctx, *req.UserID); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				return apierror.BadRequest(c, "user not found")
			}
			h.Log.ErrorContext(ctx, "create api key: get user", slog.String("error", err.Error()))
			return apierror.InternalError(c, "failed to validate user")
		}
	} else {
		req.UserID = &keyInfo.UserID
	}

	plaintextKey, err := keygen.Generate(req.KeyType)
	if err != nil {
		h.Log.ErrorContext(ctx, "create api key: generate", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to generate key")
	}

	keyHash := keygen.Hash(plaintextKey, h.HMACSecret)
	keyHint := keygen.Hint(plaintextKey)

	apiKey, err := h.DB.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		KeyHash:           keyHash,
		KeyHint:           keyHint,
		KeyType:           req.KeyType,
		Name:              req.Name,
		UserID:            req.UserID,
		DailyTokenLimit:   req.DailyTokenLimit,
		MonthlyTokenLimit: req.MonthlyTokenLimit,
		RequestsPerMinute: req.RequestsPerMinute,
		RequestsPerDay:    req.RequestsPerDay,
		ExpiresAt:         req.ExpiresAt,
		CreatedBy:         keyInfo.UserID,
	})
	if err != nil {
		h.Log.ErrorContext(ctx, "create api key: db insert", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to create api key")
	}

	// Resolve RBAC role for the new key.
	resolvedRole := auth.RoleMember
	if req.UserID != nil {
		role, err := h.DB.GetUserRole(ctx, *req.UserID)
		if err == nil {
			resolvedRole = role
		}
	}

	var expiresAt *time.Time
	if apiKey.ExpiresAt != nil {
		t, parseErr := time.Parse(time.RFC3339, *apiKey.ExpiresAt)
		if parseErr == nil {
			expiresAt = &t
		}
	}

	h.KeyCache.Set(apiKey.KeyHash, auth.KeyInfo{
		ID:                apiKey.ID,
		KeyType:           apiKey.KeyType,
		Role:              resolvedRole,
		UserID:            derefStr(apiKey.UserID),
		Name:              apiKey.Name,
		DailyTokenLimit:   apiKey.DailyTokenLimit,
		MonthlyTokenLimit: apiKey.MonthlyTokenLimit,
		RequestsPerMinute: apiKey.RequestsPerMinute,
		RequestsPerDay:    apiKey.RequestsPerDay,
		ExpiresAt:         expiresAt,
	})

	resp := createAPIKeyResponse{
		ID:                apiKey.ID,
		Key:               plaintextKey,
		KeyHint:           apiKey.KeyHint,
		KeyType:           apiKey.KeyType,
		Name:              apiKey.Name,
		UserID:            apiKey.UserID,
		DailyTokenLimit:   apiKey.DailyTokenLimit,
		MonthlyTokenLimit: apiKey.MonthlyTokenLimit,
		RequestsPerMinute: apiKey.RequestsPerMinute,
		RequestsPerDay:    apiKey.RequestsPerDay,
		ExpiresAt:         apiKey.ExpiresAt,
		CreatedBy:         apiKey.CreatedBy,
		CreatedAt:         apiKey.CreatedAt,
		UpdatedAt:         apiKey.UpdatedAt,
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// GetAPIKey handles GET /api/v1/keys/:key_id.
// System admins see any key. Members see only their own keys.
//
// @Summary      Get an API key
// @Description  Returns a single API key scoped by role.
// @Tags         keys
// @Produce      json
// @Param        key_id  path      string  true  "API key ID"
// @Success      200     {object}  apiKeyResponse
// @Failure      401     {object}  swaggerErrorResponse
// @Failure      403     {object}  swaggerErrorResponse
// @Failure      404     {object}  swaggerErrorResponse
// @Failure      500     {object}  swaggerErrorResponse
// @Security     BearerAuth
// @Router       /keys/{key_id} [get]
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

	return c.JSON(apiKeyToResponse(apiKey))
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

	// Determine scope filters based on role.
	var filterUserID string
	if auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		filterUserID = c.Query("user_id")
	} else {
		filterUserID = keyInfo.UserID
	}

	keys, err := h.DB.ListAPIKeys(c.Context(), filterUserID, p.Cursor, p.Limit+1, includeDeleted)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "list api keys", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to list api keys")
	}

	hasMore := len(keys) > p.Limit
	if hasMore {
		keys = keys[:p.Limit]
	}

	resp := paginatedAPIKeysResponse{
		Data:    make([]apiKeyResponse, len(keys)),
		HasMore: hasMore,
	}
	for i := range keys {
		resp.Data[i] = apiKeyToResponse(&keys[i])
	}
	if hasMore && len(keys) > 0 {
		resp.Cursor = keys[len(keys)-1].ID
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

	apiKey, err := h.DB.UpdateAPIKey(c.Context(), existing.ID, db.UpdateAPIKeyParams{
		Name:              req.Name,
		DailyTokenLimit:   req.DailyTokenLimit,
		MonthlyTokenLimit: req.MonthlyTokenLimit,
		RequestsPerMinute: req.RequestsPerMinute,
		RequestsPerDay:    req.RequestsPerDay,
		ExpiresAt:         req.ExpiresAt,
	})
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "api key not found")
		}
		h.Log.ErrorContext(c.Context(), "update api key", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to update api key")
	}

	if cached, ok := h.KeyCache.Get(existing.KeyHash); ok {
		cached.Name = apiKey.Name
		cached.DailyTokenLimit = apiKey.DailyTokenLimit
		cached.MonthlyTokenLimit = apiKey.MonthlyTokenLimit
		cached.RequestsPerMinute = apiKey.RequestsPerMinute
		cached.RequestsPerDay = apiKey.RequestsPerDay
		if apiKey.ExpiresAt != nil {
			t, parseErr := time.Parse(time.RFC3339, *apiKey.ExpiresAt)
			if parseErr == nil {
				cached.ExpiresAt = &t
			}
		} else {
			cached.ExpiresAt = nil
		}
		h.KeyCache.Set(existing.KeyHash, cached)
	}

	return c.JSON(apiKeyToResponse(apiKey))
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
	rotatedName := strings.TrimSuffix(existing.Name, " (rotated)") + " (rotated)"

	graceDeadline := time.Now().UTC().Add(rotateKeyGracePeriod)
	oldExpiresAt := graceDeadline.Format(time.RFC3339)
	if existing.ExpiresAt != nil {
		if t, parseErr := time.Parse(time.RFC3339, *existing.ExpiresAt); parseErr == nil && t.Before(graceDeadline) {
			oldExpiresAt = *existing.ExpiresAt
		}
	}

	rotated, err := h.DB.RotateKeyTx(ctx, existing.ID, oldExpiresAt, db.CreateAPIKeyParams{
		KeyHash:           keyHash,
		KeyHint:           keyHint,
		KeyType:           existing.KeyType,
		Name:              rotatedName,
		UserID:            existing.UserID,
		DailyTokenLimit:   existing.DailyTokenLimit,
		MonthlyTokenLimit: existing.MonthlyTokenLimit,
		RequestsPerMinute: existing.RequestsPerMinute,
		RequestsPerDay:    existing.RequestsPerDay,
		ExpiresAt:         existing.ExpiresAt,
		CreatedBy:         keyInfo.UserID,
	})
	if err != nil {
		h.Log.ErrorContext(ctx, "rotate api key: rotate tx", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to rotate key")
	}
	newKey := rotated.NewKey
	updatedOld := rotated.OldKey

	resolvedRole := auth.RoleMember
	if existing.UserID != nil {
		role, err := h.DB.GetUserRole(ctx, *existing.UserID)
		if err == nil {
			resolvedRole = role
		}
	}

	var newExpiresAt *time.Time
	if newKey.ExpiresAt != nil {
		if t, parseErr := time.Parse(time.RFC3339, *newKey.ExpiresAt); parseErr == nil {
			newExpiresAt = &t
		}
	}
	h.KeyCache.Set(newKey.KeyHash, auth.KeyInfo{
		ID:                newKey.ID,
		KeyType:           newKey.KeyType,
		Role:              resolvedRole,
		UserID:            derefStr(newKey.UserID),
		Name:              newKey.Name,
		DailyTokenLimit:   newKey.DailyTokenLimit,
		MonthlyTokenLimit: newKey.MonthlyTokenLimit,
		RequestsPerMinute: newKey.RequestsPerMinute,
		RequestsPerDay:    newKey.RequestsPerDay,
		ExpiresAt:         newExpiresAt,
	})

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
