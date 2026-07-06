package admin

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/keys"
)

// GetAdminKeysPolicy handles GET /api/v1/admin/settings/keys.
func (h *Handler) GetAdminKeysPolicy(c fiber.Ctx) error {
	if h.KeysRuntime != nil {
		return c.JSON(h.KeysRuntime.Get())
	}
	policy, err := keys.LoadPolicy(c.Context(), h.DB)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "admin keys policy: load", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load keys policy")
	}
	return c.JSON(policy)
}

// UpdateAdminKeysPolicy handles PUT /api/v1/admin/settings/keys.
func (h *Handler) UpdateAdminKeysPolicy(c fiber.Ctx) error {
	var input keys.Policy
	if err := c.Bind().JSON(&input); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if err := keys.SavePolicy(c.Context(), h.DB, input); err != nil {
		h.Log.ErrorContext(c.Context(), "admin keys policy: save", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to save keys policy")
	}
	policy, err := keys.LoadPolicy(c.Context(), h.DB)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "admin keys policy: reload", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load keys policy")
	}
	if h.KeysRuntime != nil {
		h.KeysRuntime.Set(policy)
	}
	return c.JSON(policy)
}