package admin

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
)

type connectionItem struct {
	Linked bool   `json:"linked"`
	Label  string `json:"label,omitempty"`
}

type connectionsResponse struct {
	Google connectionItem `json:"google"`
	GitHub connectionItem `json:"github"`
}

// MeConnections handles GET /api/v1/me/connections.
func (h *Handler) MeConnections(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}
	links, err := h.DB.ListOAuthConnections(c.Context(), keyInfo.UserID)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "me connections", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load connections")
	}
	resp := connectionsResponse{}
	for _, link := range links {
		item := connectionItem{Linked: true, Label: link.Label}
		switch link.Provider {
		case "google":
			resp.Google = item
		case "github":
			resp.GitHub = item
		}
	}
	return c.JSON(resp)
}

// LinkMeConnection handles POST /api/v1/me/connections/:provider/link.
// Sets a short-lived bind cookie and redirects into the OAuth bind flow.
func (h *Handler) LinkMeConnection(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}
	provider, err := normalizeOAuthProvider(c.Params("provider"))
	if err != nil {
		return apierror.BadRequest(c, "unknown oauth provider")
	}
	c.Cookie(&fiber.Cookie{
		Name:     oauthBindCookie,
		Value:    keyInfo.UserID,
		HTTPOnly: true,
		Secure:   oauthCookieSecure(c),
		SameSite: "Strict",
		MaxAge:   600,
		Path:     "/",
	})
	redirectURL := h.apiBaseURL(c) + "/api/v1/auth/oauth/" + provider + "?mode=bind"
	return c.JSON(fiber.Map{"redirect_url": redirectURL})
}

// DeleteMeConnection handles DELETE /api/v1/me/connections/:provider.
func (h *Handler) DeleteMeConnection(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}
	provider, err := normalizeOAuthProvider(c.Params("provider"))
	if err != nil {
		return apierror.BadRequest(c, "unknown oauth provider")
	}
	ctx := c.Context()

	if _, err := h.DB.GetOAuthConnection(ctx, keyInfo.UserID, provider); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "connection not found")
		}
		return apierror.InternalError(c, "failed to unlink provider")
	}

	remaining, err := h.DB.CountOAuthConnections(ctx, keyInfo.UserID)
	if err != nil {
		return apierror.InternalError(c, "failed to unlink provider")
	}
	_, hash, err := h.DB.GetUserPasswordHashByID(ctx, keyInfo.UserID)
	hasPassword := err == nil && hash != ""
	if remaining <= 1 && !hasPassword {
		return apierror.BadRequest(c, "cannot remove your only sign-in method")
	}

	if err := h.DB.DeleteOAuthConnection(ctx, keyInfo.UserID, provider); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "connection not found")
		}
		h.Log.ErrorContext(ctx, "delete connection", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to unlink provider")
	}
	return c.SendStatus(fiber.StatusNoContent)
}