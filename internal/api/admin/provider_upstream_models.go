package admin

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/db"
)

type upstreamModelImportSpec struct {
	UpstreamID string `json:"upstream_id"`
}

type importUpstreamModelsRequest struct {
	Models []upstreamModelImportSpec `json:"models"`
}

type updateUpstreamModelRequest struct {
	DisplayName *string `json:"display_name"`
	IsEnabled   *bool   `json:"is_enabled"`
}

func upstreamModelToJSON(m *db.ProviderUpstreamModel) fiber.Map {
	return fiber.Map{
		"id":                m.ID,
		"provider_id":       m.ProviderID,
		"upstream_id":       m.UpstreamID,
		"display_name":      m.DisplayName,
		"is_enabled":   m.IsEnabled,
		"metadata":     m.Metadata,
		"created_at":        m.CreatedAt,
		"updated_at":        m.UpdatedAt,
	}
}

// ListProviderUpstreamModels handles GET /api/v1/providers/:provider_id/upstream-models
func (h *Handler) ListProviderUpstreamModels(c fiber.Ctx) error {
	ctx := c.Context()
	providerID := c.Params("provider_id")
	if _, err := h.ensureProvider(ctx, providerID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "provider not found")
		}
		return apierror.InternalError(c, "failed to load provider")
	}
	enabledOnly := c.Query("enabled_only") == "1"
	models, err := h.DB.ListProviderUpstreamModels(ctx, providerID, enabledOnly)
	if err != nil {
		return apierror.InternalError(c, "failed to list upstream models")
	}
	out := make([]fiber.Map, 0, len(models))
	for i := range models {
		out = append(out, upstreamModelToJSON(&models[i]))
	}
	return c.JSON(fiber.Map{"data": out})
}

// ImportProviderUpstreamModels handles POST /api/v1/providers/:provider_id/upstream-models/import
func (h *Handler) ImportProviderUpstreamModels(c fiber.Ctx) error {
	ctx := c.Context()
	providerID := c.Params("provider_id")
	if _, err := h.ensureProvider(ctx, providerID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "provider not found")
		}
		return apierror.InternalError(c, "failed to load provider")
	}

	var req importUpstreamModelsRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if len(req.Models) == 0 {
		return apierror.BadRequest(c, "models is required")
	}

	type rowResult struct {
		UpstreamID string `json:"upstream_id"`
		ID         string `json:"id,omitempty"`
		Error      string `json:"error,omitempty"`
	}
	results := make([]rowResult, 0, len(req.Models))
	for _, spec := range req.Models {
		res := rowResult{UpstreamID: spec.UpstreamID}
		if spec.UpstreamID == "" {
			res.Error = "upstream_id is required"
			results = append(results, res)
			continue
		}
		m, err := h.DB.UpsertProviderUpstreamModel(ctx, db.UpsertProviderUpstreamModelParams{
			ProviderID: providerID, UpstreamID: spec.UpstreamID,
			IsEnabled: true,
		})
		if err != nil {
			res.Error = "import failed"
			results = append(results, res)
			continue
		}
		res.ID = m.ID
		results = append(results, res)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"results": results})
}

// UpdateProviderUpstreamModel handles PATCH /api/v1/providers/:provider_id/upstream-models/:model_id
func (h *Handler) UpdateProviderUpstreamModel(c fiber.Ctx) error {
	ctx := c.Context()
	providerID := c.Params("provider_id")
	modelID := c.Params("model_id")

	m, err := h.DB.GetProviderUpstreamModel(ctx, modelID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "upstream model not found")
		}
		return apierror.InternalError(c, "failed to load upstream model")
	}
	if m.ProviderID != providerID {
		return apierror.NotFound(c, "upstream model not found")
	}

	var req updateUpstreamModelRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	updated, err := h.DB.UpdateProviderUpstreamModel(ctx, modelID, db.UpdateProviderUpstreamModelParams{
		DisplayName: req.DisplayName,
		IsEnabled:   req.IsEnabled,
	})
	if err != nil {
		return apierror.InternalError(c, "failed to update upstream model")
	}
	return c.JSON(upstreamModelToJSON(updated))
}

// DeleteProviderUpstreamModel handles DELETE /api/v1/providers/:provider_id/upstream-models/:model_id
func (h *Handler) DeleteProviderUpstreamModel(c fiber.Ctx) error {
	ctx := c.Context()
	providerID := c.Params("provider_id")
	modelID := c.Params("model_id")

	m, err := h.DB.GetProviderUpstreamModel(ctx, modelID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "upstream model not found")
		}
		return apierror.InternalError(c, "failed to load upstream model")
	}
	if m.ProviderID != providerID {
		return apierror.NotFound(c, "upstream model not found")
	}
	if err := h.DB.DeleteProviderUpstreamModel(ctx, modelID); err != nil {
		return apierror.InternalError(c, "failed to delete upstream model")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ListAllUpstreamModels handles GET /api/v1/upstream-models (combo picker).
func (h *Handler) ListAllUpstreamModels(c fiber.Ctx) error {
	ctx := c.Context()
	enabledOnly := c.Query("enabled_only") != "0"
	models, err := h.DB.ListAllProviderUpstreamModels(ctx, enabledOnly)
	if err != nil {
		h.Log.ErrorContext(ctx, "list all upstream models", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to list upstream models")
	}
	provs, _ := h.DB.ListProviders(ctx, "", 500)
	provMap := make(map[string]db.Provider, len(provs))
	for _, p := range provs {
		provMap[p.ID] = p
	}
	out := make([]fiber.Map, 0, len(models))
	for i := range models {
		m := models[i]
		entry := upstreamModelToJSON(&m)
		if p, ok := provMap[m.ProviderID]; ok {
			slug := ""
			if p.Slug != nil {
				slug = *p.Slug
			}
			entry["provider_name"] = p.Name
			entry["provider_slug"] = slug
			entry["provider_protocol"] = p.Protocol
		}
		out = append(out, entry)
	}
	return c.JSON(fiber.Map{"data": out})
}