package admin

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	voidredis "github.com/jukaza/tavo/internal/redis"
)

type batchCreateAPIKeysRequest struct {
	Keys []createAPIKeyRequest `json:"keys"`
}

type batchCreateAPIKeysResponse struct {
	Data []createAPIKeyResponse `json:"data"`
}

type batchDeleteAPIKeysRequest struct {
	IDs []string `json:"ids"`
}

type batchDeleteAPIKeysResponse struct {
	Deleted int `json:"deleted"`
}

// BatchCreateAPIKeys handles POST /api/v1/keys/batch.
func (h *Handler) BatchCreateAPIKeys(c fiber.Ctx) error {
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	var req batchCreateAPIKeysRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if len(req.Keys) == 0 {
		return apierror.BadRequest(c, "keys array is required")
	}
	if len(req.Keys) > 50 {
		return apierror.BadRequest(c, "batch size must not exceed 50")
	}

	ctx := c.Context()
	resp := batchCreateAPIKeysResponse{
		Data: make([]createAPIKeyResponse, 0, len(req.Keys)),
	}

	for i := range req.Keys {
		item, err := h.createAPIKeyFromRequest(ctx, keyInfo, &req.Keys[i])
		if err != nil {
			var apiErr *batchItemError
			if errors.As(err, &apiErr) {
				return apierror.BadRequest(c, apiErr.msg)
			}
			h.Log.ErrorContext(ctx, "batch create api key", slog.Int("index", i), slog.String("error", err.Error()))
			return apierror.InternalError(c, "failed to create api key")
		}
		resp.Data = append(resp.Data, item)
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// BatchDeleteAPIKeys handles POST /api/v1/keys/batch/delete.
func (h *Handler) BatchDeleteAPIKeys(c fiber.Ctx) error {
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	var req batchDeleteAPIKeysRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if len(req.IDs) == 0 {
		return apierror.BadRequest(c, "ids array is required")
	}
	if len(req.IDs) > 100 {
		return apierror.BadRequest(c, "batch size must not exceed 100")
	}

	ctx := c.Context()
	deleted := 0
	for _, id := range req.IDs {
		apiKey, err := h.DB.GetAPIKey(ctx, id)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				continue
			}
			h.Log.ErrorContext(ctx, "batch delete api key: get", slog.String("id", id), slog.String("error", err.Error()))
			return apierror.InternalError(c, "failed to delete api keys")
		}
		if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
			if apiKey.UserID == nil || *apiKey.UserID != keyInfo.UserID {
				continue
			}
		}
		if err := h.DB.DeleteAPIKey(ctx, apiKey.ID); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				continue
			}
			h.Log.ErrorContext(ctx, "batch delete api key", slog.String("id", id), slog.String("error", err.Error()))
			return apierror.InternalError(c, "failed to delete api keys")
		}
		h.KeyCache.Delete(apiKey.KeyHash)
		if h.Redis != nil {
			if err := h.Redis.PublishInvalidation(ctx, voidredis.ChannelKeys, apiKey.KeyHash); err != nil {
				h.Log.LogAttrs(ctx, slog.LevelWarn, "redis: publish key invalidation failed",
					slog.String("error", err.Error()),
				)
			}
		}
		deleted++
	}

	return c.JSON(batchDeleteAPIKeysResponse{Deleted: deleted})
}

type batchItemError struct {
	msg string
}

func (e *batchItemError) Error() string { return e.msg }