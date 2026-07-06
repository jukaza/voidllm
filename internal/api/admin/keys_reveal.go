package admin

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/pkg/keygen"
)

var errAPIKeyNotRetrievable = errors.New("api key not retrievable")

type revealAPIKeyResponse struct {
	Key string `json:"key"`
}

// RevealAPIKey handles GET /api/v1/keys/:key_id/reveal — returns the full key for copy.
func (h *Handler) RevealAPIKey(c fiber.Ctx) error {
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}

	apiKey, err := h.DB.GetAPIKey(c.Context(), c.Params("key_id"))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "api key not found")
		}
		return apierror.InternalError(c, "failed to get api key")
	}

	if apiKey.KeyType != keygen.KeyTypeUser {
		return apierror.BadRequest(c, "only user keys can be revealed")
	}

	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		if !apiKeyVisibleToCallerKey(apiKey, keyInfo) {
			return apierror.Send(c, fiber.StatusForbidden, "forbidden", "you can only reveal keys within your scope")
		}
	}

	plaintext, err := h.decryptStoredAPIKey(apiKey)
	if err != nil {
		if errors.Is(err, errAPIKeyNotRetrievable) {
			return apierror.Send(c, fiber.StatusNotFound, "key_not_retrievable",
				"full key is unavailable for this record — rotate the key once to enable copy from any device")
		}
		h.Log.ErrorContext(c.Context(), "reveal api key", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to reveal api key")
	}

	return c.JSON(revealAPIKeyResponse{Key: plaintext})
}