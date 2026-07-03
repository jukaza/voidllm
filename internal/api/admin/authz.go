// Package admin provides HTTP handlers for the VoidLLM admin API.
// This file contains shared authorization helpers used across admin handlers.
package admin

import (
	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/auth"
)

// requireOrgAccess checks that the caller is authenticated.
// On success it returns the caller's KeyInfo and true.
// On failure it writes the HTTP error response to c and returns nil, false.
func requireOrgAccess(c fiber.Ctx, orgID string) (*auth.KeyInfo, bool) {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		_ = apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
		return nil, false
	}
	return keyInfo, true
}

// requireOrgAdmin checks that the caller is authenticated.
// On success it returns the caller's KeyInfo and true.
// On failure it writes the HTTP error response to c and returns nil, false.
func requireOrgAdmin(c fiber.Ctx, orgID string) (*auth.KeyInfo, bool) {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		_ = apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
		return nil, false
	}
	return keyInfo, true
}
