package admin

import (
	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/audit"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/cache"
)

// RegisterRoutes mounts all admin API routes under /api/v1 on the given Fiber app.
// The login route at POST /api/v1/auth/login is public. All other routes require
// a valid Bearer API key via auth.Middleware. Individual routes apply additional
// role requirements where needed.
//
// When auditLogger is non-nil, the audit middleware is mounted on the
// authenticated /api/v1 group so that all successful mutations are recorded.
func RegisterRoutes(app *fiber.App, handler *Handler, keyCache *cache.Cache[string, auth.KeyInfo], hmacSecret []byte, auditLogger *audit.Logger) {
	// Public routes — no auth required.
	app.Post("/api/v1/auth/login", handler.Login)
	app.Post("/api/v1/auth/register", handler.Register)
	app.Get("/api/v1/auth/providers", handler.AuthProviders)
	app.Get("/api/v1/public/models", handler.PublicModels)

	var apiMiddlewares []any
	apiMiddlewares = append(apiMiddlewares, auth.Middleware(keyCache, hmacSecret))
	if auditLogger != nil {
		apiMiddlewares = append(apiMiddlewares, audit.Middleware(auditLogger))
	}

	api := app.Group("/api/v1", apiMiddlewares...)

	// Current user profile — no role restriction.
	api.Get("/me", handler.Me)
	api.Post("/me/password", handler.ChangeOwnPassword)
	api.Get("/me/available-models", handler.AvailableModels)

	// Customer wallet — any authenticated user sees their own balance/ledger.
	api.Get("/me/wallet", handler.MyWallet)
	api.Get("/me/transactions", handler.MyTransactions)
	api.Post("/me/topups", handler.CreateMyTopup)
	api.Get("/me/topups", handler.MyTopups)

	// Own usage — no role restriction; any authenticated key sees its own data.
	api.Get("/usage/me", handler.MyUsage)

	// Dashboard stats — no role restriction.
	api.Get("/dashboard/stats", handler.DashboardStats)

	// Organizations
	api.Post("/orgs", auth.RequireRole(auth.RoleSystemAdmin), handler.CreateOrg)
	api.Get("/orgs", auth.RequireRole(auth.RoleOrgAdmin), handler.ListOrgs)
	api.Get("/orgs/:org_id", auth.RequireRole(auth.RoleOrgAdmin), handler.GetOrg)
	api.Patch("/orgs/:org_id", auth.RequireRole(auth.RoleOrgAdmin), handler.UpdateOrg)
	api.Delete("/orgs/:org_id", auth.RequireRole(auth.RoleSystemAdmin), handler.DeleteOrg)

	// Users
	api.Post("/users", auth.RequireRole(auth.RoleOrgAdmin), handler.CreateUser)
	api.Get("/users", auth.RequireRole(auth.RoleSystemAdmin), handler.ListUsers)
	api.Get("/users/:user_id", auth.RequireRole(auth.RoleOrgAdmin), handler.GetUser)
	api.Patch("/users/:user_id", auth.RequireRole(auth.RoleOrgAdmin), handler.UpdateUser)
	api.Delete("/users/:user_id", auth.RequireRole(auth.RoleSystemAdmin), handler.DeleteUser)

	// Org Memberships
	api.Post("/orgs/:org_id/members", auth.RequireRole(auth.RoleOrgAdmin), handler.CreateOrgMembership)
	api.Get("/orgs/:org_id/members", auth.RequireRole(auth.RoleOrgAdmin), handler.ListOrgMemberships)
	api.Patch("/orgs/:org_id/members/:membership_id", auth.RequireRole(auth.RoleOrgAdmin), handler.UpdateOrgMembership)
	api.Delete("/orgs/:org_id/members/:membership_id", auth.RequireRole(auth.RoleOrgAdmin), handler.DeleteOrgMembership)

	// API Keys
	api.Post("/orgs/:org_id/keys", auth.RequireRole(auth.RoleMember), handler.CreateAPIKey)
	api.Get("/orgs/:org_id/keys", auth.RequireRole(auth.RoleMember), handler.ListAPIKeys)
	api.Get("/orgs/:org_id/keys/:key_id", auth.RequireRole(auth.RoleMember), handler.GetAPIKey)
	api.Patch("/orgs/:org_id/keys/:key_id", auth.RequireRole(auth.RoleMember), handler.UpdateAPIKey)
	api.Delete("/orgs/:org_id/keys/:key_id", auth.RequireRole(auth.RoleMember), handler.DeleteAPIKey)
	api.Post("/orgs/:org_id/keys/:key_id/rotate", auth.RequireRole(auth.RoleMember), handler.RotateAPIKey)

	// Models — global resources managed by system admins only.
	// An org_admin in a multi-org deployment must not be able to add or modify
	// models that are visible to all organisations.
	// Static sub-paths (health, test-connection) are registered before
	// /:model_id so Fiber does not treat them as model_id parameter values.
	api.Get("/models/health", auth.RequireRole(auth.RoleMember), handler.GetModelHealth)
	api.Post("/models/test-connection", auth.RequireRole(auth.RoleSystemAdmin), handler.TestModelConnection)
	api.Post("/models", auth.RequireRole(auth.RoleSystemAdmin), handler.CreateModel)
	api.Get("/models", auth.RequireRole(auth.RoleSystemAdmin), handler.ListModels)
	api.Get("/models/:model_id", auth.RequireRole(auth.RoleSystemAdmin), handler.GetModel)
	api.Patch("/models/:model_id", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateModel)
	api.Delete("/models/:model_id", auth.RequireRole(auth.RoleSystemAdmin), handler.DeleteModel)
	api.Patch("/models/:model_id/activate", auth.RequireRole(auth.RoleSystemAdmin), handler.ActivateModel)
	api.Patch("/models/:model_id/deactivate", auth.RequireRole(auth.RoleSystemAdmin), handler.DeactivateModel)

	// Model Deployments — sub-resources of a model, managed by system admins.
	api.Post("/models/:model_id/deployments", auth.RequireRole(auth.RoleSystemAdmin), handler.createDeployment)
	api.Get("/models/:model_id/deployments", auth.RequireRole(auth.RoleSystemAdmin), handler.listDeployments)
	api.Patch("/models/:model_id/deployments/:deployment_id", auth.RequireRole(auth.RoleSystemAdmin), handler.updateDeployment)
	api.Delete("/models/:model_id/deployments/:deployment_id", auth.RequireRole(auth.RoleSystemAdmin), handler.deleteDeployment)

	// Model Access Control
	api.Get("/orgs/:org_id/model-access", auth.RequireRole(auth.RoleOrgAdmin), handler.GetOrgModelAccess)
	api.Put("/orgs/:org_id/model-access", auth.RequireRole(auth.RoleOrgAdmin), handler.SetOrgModelAccess)
	api.Get("/orgs/:org_id/keys/:key_id/model-access", auth.RequireRole(auth.RoleOrgAdmin), handler.GetKeyModelAccess)
	api.Put("/orgs/:org_id/keys/:key_id/model-access", auth.RequireRole(auth.RoleOrgAdmin), handler.SetKeyModelAccess)

	// Model Aliases
	api.Post("/orgs/:org_id/model-aliases", auth.RequireRole(auth.RoleOrgAdmin), handler.CreateOrgAlias)
	api.Get("/orgs/:org_id/model-aliases", auth.RequireRole(auth.RoleOrgAdmin), handler.ListOrgAliases)
	api.Delete("/orgs/:org_id/model-aliases/:alias_id", auth.RequireRole(auth.RoleOrgAdmin), handler.DeleteOrgAlias)

	// Usage
	api.Get("/usage", auth.RequireRole(auth.RoleSystemAdmin), handler.SystemAdminUsage)
	api.Get("/orgs/:org_id/usage", auth.RequireRole(auth.RoleOrgAdmin), handler.GetOrgUsage)

	// Providers (upstream partners) — system_admin only.
	api.Post("/providers", auth.RequireRole(auth.RoleSystemAdmin), handler.CreateProvider)
	api.Get("/providers", auth.RequireRole(auth.RoleSystemAdmin), handler.ListProviders)
	api.Get("/providers/:provider_id", auth.RequireRole(auth.RoleSystemAdmin), handler.GetProvider)
	api.Patch("/providers/:provider_id", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateProvider)
	api.Delete("/providers/:provider_id", auth.RequireRole(auth.RoleSystemAdmin), handler.DeleteProvider)

	// Top-up review queue and wallet administration — system_admin only.
	api.Get("/topups", auth.RequireRole(auth.RoleSystemAdmin), handler.ListTopups)
	api.Post("/topups/:topup_id/review", auth.RequireRole(auth.RoleSystemAdmin), handler.ReviewTopup)
	api.Get("/users/:user_id/wallet", auth.RequireRole(auth.RoleSystemAdmin), handler.GetUserWallet)
	api.Post("/users/:user_id/wallet/adjust", auth.RequireRole(auth.RoleSystemAdmin), handler.AdjustWallet)

	// Update check — any authenticated user may read the cached update status.
	// Version info is not sensitive; no additional role gate required.
	api.Get("/system/update-check", handler.GetUpdateStatus)

}
