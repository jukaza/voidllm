package admin

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/db"
	voidredis "github.com/jukaza/tavo/internal/redis"
)

type modelRouteStepInput struct {
	ProviderID    string `json:"provider_id"`
	UpstreamModel string `json:"upstream_model"`
	IsEnabled     *bool  `json:"is_enabled"`
}

type replaceModelRoutesRequest struct {
	Steps []modelRouteStepInput `json:"steps"`
}

func modelRouteStepToJSON(s *db.ModelRouteStep, prov *db.Provider) fiber.Map {
	out := fiber.Map{
		"id":             s.ID,
		"model_id":       s.ModelID,
		"position":       s.Position,
		"provider_id":    s.ProviderID,
		"upstream_model": s.UpstreamModel,
		"is_enabled":     s.IsEnabled,
		"created_at":     s.CreatedAt,
	}
	if prov != nil {
		out["provider_name"] = prov.Name
		out["provider_protocol"] = prov.Protocol
		if prov.Slug != nil {
			out["provider_slug"] = *prov.Slug
		}
	}
	return out
}

// ListModelRoutes handles GET /api/v1/models/:model_id/routes
func (h *Handler) ListModelRoutes(c fiber.Ctx) error {
	ctx := c.Context()
	modelID := c.Params("model_id")
	if _, err := h.DB.GetModel(ctx, modelID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "model not found")
		}
		return apierror.InternalError(c, "failed to load model")
	}
	steps, err := h.DB.ListModelRouteSteps(ctx, modelID, false)
	if err != nil {
		return apierror.InternalError(c, "failed to list route steps")
	}
	provs, _ := h.DB.ListProviders(ctx, "", 500)
	provMap := make(map[string]db.Provider, len(provs))
	for _, p := range provs {
		provMap[p.ID] = p
	}
	out := make([]fiber.Map, 0, len(steps))
	for i := range steps {
		var prov *db.Provider
		if p, ok := provMap[steps[i].ProviderID]; ok {
			pp := p
			prov = &pp
		}
		out = append(out, modelRouteStepToJSON(&steps[i], prov))
	}
	return c.JSON(fiber.Map{"data": out})
}

// ReplaceModelRoutes handles PUT /api/v1/models/:model_id/routes
func (h *Handler) ReplaceModelRoutes(c fiber.Ctx) error {
	ctx := c.Context()
	modelID := c.Params("model_id")
	if _, err := h.DB.GetModel(ctx, modelID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "model not found")
		}
		return apierror.InternalError(c, "failed to load model")
	}

	var req replaceModelRoutesRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	inputs := make([]db.ModelRouteStepInput, 0, len(req.Steps))
	for _, s := range req.Steps {
		if s.ProviderID == "" || s.UpstreamModel == "" {
			return apierror.BadRequest(c, "each step requires provider_id and upstream_model")
		}
		prov, err := h.DB.GetProvider(ctx, s.ProviderID)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				return apierror.BadRequest(c, "provider not found: "+s.ProviderID)
			}
			return apierror.InternalError(c, "failed to validate provider")
		}
		if prov.Status == "paused" {
			return apierror.BadRequest(c, "provider is paused: "+prov.Name)
		}
		um, umErr := h.DB.GetProviderUpstreamModelByUpstreamID(ctx, s.ProviderID, s.UpstreamModel)
		if umErr != nil {
			if errors.Is(umErr, db.ErrNotFound) {
				return apierror.BadRequest(c, "upstream model not found in provider inventory: "+s.UpstreamModel)
			}
			return apierror.InternalError(c, "failed to validate upstream model")
		}
		if !um.IsEnabled {
			return apierror.BadRequest(c, "upstream model is disabled: "+s.UpstreamModel)
		}
		enabled := true
		if s.IsEnabled != nil {
			enabled = *s.IsEnabled
		}
		inputs = append(inputs, db.ModelRouteStepInput{
			ProviderID: s.ProviderID, UpstreamModel: s.UpstreamModel, IsEnabled: enabled,
		})
	}

	steps, err := h.DB.ReplaceModelRouteSteps(ctx, modelID, inputs)
	if err != nil {
		h.Log.ErrorContext(ctx, "replace model routes", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to save route steps")
	}

	if h.ReloadModels != nil {
		if reloadErr := h.ReloadModels(ctx); reloadErr != nil {
			h.Log.ErrorContext(ctx, "reload registry after route replace", slog.String("error", reloadErr.Error()))
		}
	}
	if h.Redis != nil {
		if pubErr := h.Redis.PublishInvalidation(ctx, voidredis.ChannelModels, "reload"); pubErr != nil {
			h.Log.ErrorContext(ctx, "replace model routes: publish invalidation", slog.String("error", pubErr.Error()))
		}
	}

	provs, _ := h.DB.ListProviders(ctx, "", 500)
	provMap := make(map[string]db.Provider, len(provs))
	for _, p := range provs {
		provMap[p.ID] = p
	}
	out := make([]fiber.Map, 0, len(steps))
	for i := range steps {
		var prov *db.Provider
		if p, ok := provMap[steps[i].ProviderID]; ok {
			pp := p
			prov = &pp
		}
		out = append(out, modelRouteStepToJSON(&steps[i], prov))
	}
	return c.JSON(fiber.Map{"data": out})
}