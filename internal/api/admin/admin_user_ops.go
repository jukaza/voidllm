package admin

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
)

type adminConnectionItem struct {
	Provider string `json:"provider"`
	Label    string `json:"label,omitempty"`
	Linked   bool   `json:"linked"`
}

type adminConnectionsResponse struct {
	Connections []adminConnectionItem `json:"connections"`
}

// AdminUserConnections handles GET /api/v1/admin/users/:user_id/connections.
func (h *Handler) AdminUserConnections(c fiber.Ctx) error {
	keyInfo, err := actorFromCtx(c)
	if err != nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	userID := c.Params("user_id")
	user, err := h.DB.GetUser(c.Context(), userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user not found")
		}
		return apierror.InternalError(c, "failed to get user")
	}
	if !canManageUser(keyInfo.Role, user) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	links, err := h.DB.ListOAuthConnections(c.Context(), userID)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "admin user connections", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load connections")
	}

	items := make([]adminConnectionItem, 0, len(links))
	for _, link := range links {
		items = append(items, adminConnectionItem{
			Provider: link.Provider,
			Label:    link.Label,
			Linked:   true,
		})
	}
	return c.JSON(adminConnectionsResponse{Connections: items})
}

// AdminRevokeUserSessions handles DELETE /api/v1/admin/users/:user_id/sessions.
func (h *Handler) AdminRevokeUserSessions(c fiber.Ctx) error {
	keyInfo, err := actorFromCtx(c)
	if err != nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	userID := c.Params("user_id")
	user, err := h.DB.GetUser(c.Context(), userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user not found")
		}
		return apierror.InternalError(c, "failed to get user")
	}
	if !canManageUser(keyInfo.Role, user) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	if err := h.revokeAllUserSessions(c, userID); err != nil {
		h.Log.ErrorContext(c.Context(), "revoke user sessions", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to revoke sessions")
	}
	return c.SendStatus(fiber.StatusNoContent)
}