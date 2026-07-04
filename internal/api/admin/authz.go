package admin

import (
	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/auth"
)

func requireAuth(c fiber.Ctx) (*auth.KeyInfo, bool) {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		_ = apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
		return nil, false
	}
	return keyInfo, true
}