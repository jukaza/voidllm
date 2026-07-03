package admin

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
	voidredis "github.com/voidmind-io/voidllm/internal/redis"
)

// modelAccessRequest is the JSON body accepted by the Set*ModelAccess handlers.
type modelAccessRequest struct {
	Models []string `json:"models"`
}

// modelAccessResponse is the JSON body returned by all model access handlers.
type modelAccessResponse struct {
	Models []string `json:"models"`
}

// GetKeyModelAccess handles GET /api/v1/keys/:key_id/model-access.
// Returns the list of model names allowed for the API key.
// An empty list means "not configured" — all models are allowed.
func (h *Handler) GetKeyModelAccess(c fiber.Ctx) error {
	keyID := c.Params("key_id")

	if _, ok := requireOrgAccess(c, ""); !ok {
		return nil
	}

	if err := h.requireKeyBelongsToUser(c, keyID); err != nil {
		return err
	}

	models, err := h.DB.GetKeyModelAccess(c.Context(), keyID)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "get key model access", slog.String("key_id", keyID), slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to get key model access")
	}

	if models == nil {
		models = []string{}
	}
	return c.JSON(modelAccessResponse{Models: models})
}

// SetKeyModelAccess handles PUT /api/v1/keys/:key_id/model-access.
// Replaces the API key's model allowlist atomically.
func (h *Handler) SetKeyModelAccess(c fiber.Ctx) error {
	keyID := c.Params("key_id")

	if _, ok := requireOrgAccess(c, ""); !ok {
		return nil
	}

	if err := h.requireKeyBelongsToUser(c, keyID); err != nil {
		return err
	}

	var req modelAccessRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	if err := h.validateModelNames(c, req.Models); err != nil {
		return err
	}

	if err := h.DB.SetKeyModelAccess(c.Context(), keyID, req.Models); err != nil {
		h.Log.ErrorContext(c.Context(), "set key model access", slog.String("key_id", keyID), slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to set key model access")
	}

	h.refreshAccessCache(c.Context())
	h.publishAccessInvalidation(c.Context())

	return c.JSON(modelAccessResponse{Models: req.Models})
}

// validateModelNames checks that each name in models references a known model
// in the registry and that there are no duplicates. It writes a 400 response
// and returns a non-nil error on the first violation found.
func (h *Handler) validateModelNames(c fiber.Ctx, models []string) error {
	seen := make(map[string]struct{}, len(models))
	for _, name := range models {
		if _, dup := seen[name]; dup {
			return apierror.BadRequest(c, "duplicate model: "+name)
		}
		seen[name] = struct{}{}
		if _, err := h.Registry.Resolve(name); err != nil {
			return apierror.BadRequest(c, "unknown model: "+name)
		}
	}
	return nil
}

// requireKeyBelongsToUser fetches the API key and confirms it belongs to the authenticated user.
func (h *Handler) requireKeyBelongsToUser(c fiber.Ctx, keyID string) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	key, err := h.DB.GetAPIKey(c.Context(), keyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "api key not found")
		}
		h.Log.ErrorContext(c.Context(), "validate api key ownership", slog.String("key_id", keyID), slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to validate api key")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		if key.UserID == nil || *key.UserID != keyInfo.UserID {
			return apierror.NotFound(c, "api key not found")
		}
	}
	return nil
}

// publishAccessInvalidation sends a cache invalidation message on the access
// channel. It is a no-op when Redis is not configured.
func (h *Handler) publishAccessInvalidation(ctx context.Context) {
	if h.Redis == nil {
		return
	}
	if err := h.Redis.PublishInvalidation(ctx, voidredis.ChannelAccess, "reload"); err != nil {
		h.Log.LogAttrs(ctx, slog.LevelWarn, "redis: publish access invalidation failed",
			slog.String("error", err.Error()),
		)
	}
}
