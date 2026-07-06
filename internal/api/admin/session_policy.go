package admin

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/security"
)

func clientIP(c fiber.Ctx) string {
	return c.IP()
}

func clientUserAgent(c fiber.Ctx) string {
	return strings.TrimSpace(c.Get("User-Agent"))
}

func (h *Handler) loadSessionPolicy(c fiber.Ctx) (security.SessionPolicyConfig, error) {
	cfg, err := security.Load(c.Context(), h.DB)
	if err != nil {
		return security.SessionPolicyConfig{}, err
	}
	return cfg.Session, nil
}