package admin

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/db"
	voidredis "github.com/jukaza/tavo/internal/redis"
)

type revokeAllUserKeysResponse struct {
	Revoked int64 `json:"revoked"`
}

// ListUserAPIKeys handles GET /api/v1/admin/users/:user_id/keys.
func (h *Handler) ListUserAPIKeys(c fiber.Ctx) error {
	userID := c.Params("user_id")

	p, err := parsePagination(c)
	if err != nil {
		return apierror.BadRequest(c, err.Error())
	}

	if _, err := h.DB.GetUser(c.Context(), userID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user not found")
		}
		h.Log.ErrorContext(c.Context(), "list user keys: get user", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to list user keys")
	}

	keys, err := h.DB.ListAPIKeys(c.Context(), userID, p.Cursor, p.Limit+1, false)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "list user keys", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to list user keys")
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
		resp.Data[i] = buildAPIKeyResponse(&keys[i])
	}
	if hasMore && len(keys) > 0 {
		resp.Cursor = keys[len(keys)-1].ID
	}
	return c.JSON(resp)
}

// RevokeAllUserAPIKeys handles POST /api/v1/admin/users/:user_id/keys/revoke-all.
func (h *Handler) RevokeAllUserAPIKeys(c fiber.Ctx) error {
	userID := c.Params("user_id")
	ctx := c.Context()

	if _, err := h.DB.GetUser(ctx, userID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user not found")
		}
		h.Log.ErrorContext(ctx, "revoke all user keys: get user", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to revoke user keys")
	}

	keys, err := h.DB.ListAPIKeys(ctx, userID, "", 1000, false)
	if err != nil {
		h.Log.ErrorContext(ctx, "revoke all user keys: list", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to revoke user keys")
	}

	revoked, err := h.DB.RevokeAllUserAPIKeys(ctx, userID)
	if err != nil {
		h.Log.ErrorContext(ctx, "revoke all user keys", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to revoke user keys")
	}

	for i := range keys {
		h.KeyCache.Delete(keys[i].KeyHash)
		if h.Redis != nil {
			if err := h.Redis.PublishInvalidation(ctx, voidredis.ChannelKeys, keys[i].KeyHash); err != nil {
				h.Log.LogAttrs(ctx, slog.LevelWarn, "redis: publish key invalidation failed",
					slog.String("error", err.Error()),
				)
			}
		}
	}

	return c.JSON(revokeAllUserKeysResponse{Revoked: revoked})
}