package auth

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
)

const (
	RoleMember = "member"
	RoleAdmin  = "admin"
	RoleRoot   = "root"

	// RoleSystemAdmin is a backward-compatible alias for RoleRoot.
	RoleSystemAdmin = RoleRoot
)

var roleRank = map[string]int{
	RoleMember: 0,
	RoleAdmin:  1,
	RoleRoot:   2,
}

// RoleRank returns the privilege level for a role string. Unknown roles map to -1.
func RoleRank(role string) int {
	r, ok := roleRank[role]
	if !ok {
		return -1
	}
	return r
}

func HasRole(role string, required string) bool {
	return RoleRank(role) >= RoleRank(required)
}

// CanManageRole reports whether actor may manage a user with targetRole.
// Actors can only manage strictly lower-privilege users.
func CanManageRole(actorRole, targetRole string) bool {
	return RoleRank(actorRole) > RoleRank(targetRole)
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