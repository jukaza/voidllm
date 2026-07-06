package admin

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
)

func requireAuth(c fiber.Ctx) (*auth.KeyInfo, bool) {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		_ = apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
		return nil, false
	}
	return keyInfo, true
}