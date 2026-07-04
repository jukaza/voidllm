package auth

import (
	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/apierror"
)

const (
	RoleSystemAdmin = "system_admin"
	RoleMember      = "member"
)

var roleRank = map[string]int{
	RoleMember:      0,
	RoleSystemAdmin: 1,
}

func HasRole(role string, required string) bool {
	r, ok := roleRank[role]
	if !ok {
		return false
	}
	req, reqOK := roleRank[required]
	if !reqOK {
		return false
	}
	return r >= req
}

func RequireRole(required string) fiber.Handler {
	return func(c fiber.Ctx) error {
		keyInfo := KeyInfoFromCtx(c)
		if keyInfo == nil {
			return apierror.Unauthorized(c, "missing authorization header")
		}
		if !HasRole(keyInfo.Role, required) {
			return apierror.Forbidden(c, "insufficient permissions")
		}
		return c.Next()
	}
}