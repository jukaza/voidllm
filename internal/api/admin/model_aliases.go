package admin

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/db"
	voidredis "github.com/voidmind-io/voidllm/internal/redis"
)

// createModelAliasRequest is the JSON body accepted by the Create*Alias handlers.
type createModelAliasRequest struct {
	Alias     string `json:"alias"`
	ModelName string `json:"model_name"`
}

// modelAliasResponse is the JSON body returned by all model alias handlers.
type modelAliasResponse struct {
	ID        string `json:"id"`
	Alias     string `json:"alias"`
	ModelName string `json:"model_name"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

// modelAliasToResponse converts a db.ModelAlias to its API wire representation.
func modelAliasToResponse(a *db.ModelAlias) modelAliasResponse {
	return modelAliasResponse{
		ID:        a.ID,
		Alias:     a.Alias,
		ModelName: a.ModelName,
		CreatedBy: a.CreatedBy,
		CreatedAt: a.CreatedAt,
	}
}

// validateAliasName checks that name is non-empty, at most 128 characters, and
// contains only letters, digits, hyphens, and underscores.
func validateAliasName(name string) string {
	if name == "" {
		return "alias is required"
	}
	if len(name) > 128 {
		return "alias must be at most 128 characters"
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return "alias may only contain letters, numbers, hyphens, and underscores"
		}
	}
	return ""
}

// CreateModelAlias handles POST /api/v1/model-aliases.
func (h *Handler) CreateModelAlias(c fiber.Ctx) error {
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	var req createModelAliasRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	aliasName := strings.TrimSpace(req.Alias)
	if msg := validateAliasName(aliasName); msg != "" {
		return apierror.BadRequest(c, msg)
	}
	if req.ModelName == "" {
		return apierror.BadRequest(c, "model_name is required")
	}

	if _, err := h.Registry.Resolve(req.ModelName); err != nil {
		return apierror.BadRequest(c, "unknown model: "+req.ModelName)
	}

	if _, err := h.Registry.Resolve(aliasName); err == nil {
		return apierror.BadRequest(c, "alias conflicts with model name")
	}

	params := db.CreateModelAliasParams{
		Alias:     aliasName,
		ModelName: req.ModelName,
		CreatedBy: keyInfo.ID,
	}
	alias, err := h.DB.CreateModelAlias(c.Context(), params)
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			return apierror.Conflict(c, "alias already exists")
		}
		h.Log.ErrorContext(c.Context(), "create model alias",
			slog.String("error", err.Error()),
		)
		return apierror.InternalError(c, "failed to create model alias")
	}

	h.refreshAliasCache(c.Context())
	h.publishAliasInvalidation(c.Context())

	return c.Status(fiber.StatusCreated).JSON(modelAliasToResponse(alias))
}

// ListModelAliases handles GET /api/v1/model-aliases.
func (h *Handler) ListModelAliases(c fiber.Ctx) error {
	if _, ok := requireAuth(c); !ok {
		return nil
	}

	aliases, err := h.DB.ListModelAliases(c.Context(), "", "")
	if err != nil {
		h.Log.ErrorContext(c.Context(), "list model aliases",
			slog.String("error", err.Error()),
		)
		return apierror.InternalError(c, "failed to list model aliases")
	}

	resp := make([]modelAliasResponse, len(aliases))
	for i := range aliases {
		resp[i] = modelAliasToResponse(&aliases[i])
	}
	return c.JSON(resp)
}

// DeleteModelAlias handles DELETE /api/v1/model-aliases/:alias_id.
func (h *Handler) DeleteModelAlias(c fiber.Ctx) error {
	aliasID := c.Params("alias_id")

	if _, ok := requireAuth(c); !ok {
		return nil
	}

	if err := h.DB.DeleteModelAlias(c.Context(), aliasID, "", ""); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "model alias not found")
		}
		h.Log.ErrorContext(c.Context(), "delete model alias",
			slog.String("alias_id", aliasID),
			slog.String("error", err.Error()),
		)
		return apierror.InternalError(c, "failed to delete model alias")
	}

	h.refreshAliasCache(c.Context())
	h.publishAliasInvalidation(c.Context())

	return c.SendStatus(fiber.StatusNoContent)
}

// refreshAliasCache reloads all model aliases.
func (h *Handler) refreshAliasCache(ctx context.Context) {
	if h.AliasCache == nil {
		return
	}
	aliases, err := h.DB.LoadAllModelAliases(ctx)
	if err != nil {
		h.Log.ErrorContext(ctx, "refresh alias cache", slog.String("error", err.Error()))
		return
	}
	h.AliasCache.Load(aliases)
}

// publishAliasInvalidation sends a cache invalidation message on the aliases channel.
func (h *Handler) publishAliasInvalidation(ctx context.Context) {
	if h.Redis == nil {
		return
	}
	if err := h.Redis.PublishInvalidation(ctx, voidredis.ChannelAliases, "reload"); err != nil {
		h.Log.LogAttrs(ctx, slog.LevelWarn, "redis: publish alias invalidation failed",
			slog.String("error", err.Error()),
		)
	}
}
