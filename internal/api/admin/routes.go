package admin

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/audit"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/cache"
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
	app.Get("/api/v1/auth/oauth/:provider", handler.StartOAuth)
	app.Get("/api/v1/auth/oauth/:provider/callback", handler.OAuthCallback)
	app.Post("/api/v1/auth/oauth/exchange", handler.OAuthExchange)
	app.Get("/api/v1/auth/providers", handler.AuthProviders)
	app.Get("/api/v1/public/auth-config", handler.GetPublicAuthConfig)
	app.Get("/api/v1/public/catalog", handler.PublicCatalog)
	app.Get("/api/v1/public/models", handler.PublicModels)
	app.Get("/api/v1/public/site", handler.GetPublicSite)
	app.Get("/api/v1/public/features", handler.GetPublicFeatures)
	app.Get("/api/v1/public/topup-config", handler.GetPublicTopupConfig)
	app.Post("/api/v1/webhooks/sepay", handler.SepayWebhook)

	var apiMiddlewares []any
	apiMiddlewares = append(apiMiddlewares, auth.Middleware(keyCache, hmacSecret, handler.DB))
	if auditLogger != nil {
		apiMiddlewares = append(apiMiddlewares, audit.Middleware(auditLogger))
	}

	api := app.Group("/api/v1", apiMiddlewares...)

	// Current user profile — no role restriction.
	api.Get("/me", handler.Me)
	api.Post("/me/password", handler.ChangeOwnPassword)
	api.Post("/me/password/set", handler.SetOwnPassword)
	api.Get("/me/sessions", handler.MeSessions)
	api.Delete("/me/sessions", handler.RevokeOtherMeSessions)
	api.Delete("/me/sessions/:id", handler.RevokeMeSession)
	api.Get("/me/available-models", handler.AvailableModels)
	api.Get("/me/connections", handler.MeConnections)
	api.Post("/me/connections/:provider/link", handler.LinkMeConnection)
	api.Delete("/me/connections/:provider", handler.DeleteMeConnection)

	// Customer wallet — any authenticated user sees their own balance/ledger.
	api.Get("/me/wallet", handler.MyWallet)
	api.Get("/me/transactions", handler.MyTransactions)
	api.Post("/me/topups", handler.CreateMyTopup)
	api.Post("/me/topups/quote", handler.QuoteTopup)
	api.Get("/me/topups/:trade_no/status", handler.GetTopupStatus)
	api.Get("/me/topups", handler.MyTopups)

	// Own usage — no role restriction; any authenticated key sees its own data.
	api.Get("/usage/me", handler.MyUsage)
	api.Get("/usage/me/logs", handler.MyUsageLogs)
	api.Get("/usage/me/logs/:request_id", handler.MyUsageLogDetail)
	api.Get("/usage/stream", handler.UsageStream)
	api.Get("/usage/live", handler.UsageLive)

	// Dashboard stats — no role restriction.
	api.Get("/dashboard/stats", handler.DashboardStats)

	// Users — managed by system admins.
	api.Post("/users", auth.RequireRole(auth.RoleSystemAdmin), handler.CreateUser)
	api.Get("/users", auth.RequireRole(auth.RoleSystemAdmin), handler.ListUsers)
	api.Get("/users/:user_id", auth.RequireRole(auth.RoleSystemAdmin), handler.GetUser)
	api.Patch("/users/:user_id", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateUser)
	api.Delete("/users/:user_id", auth.RequireRole(auth.RoleSystemAdmin), handler.DeleteUser)
	api.Get("/admin/users/:user_id/keys", auth.RequireRole(auth.RoleSystemAdmin), handler.ListUserAPIKeys)
	api.Post("/admin/users/:user_id/keys/revoke-all", auth.RequireRole(auth.RoleSystemAdmin), handler.RevokeAllUserAPIKeys)

	// API Keys
	api.Post("/keys/batch", auth.RequireRole(auth.RoleMember), handler.BatchCreateAPIKeys)
	api.Post("/keys/batch/delete", auth.RequireRole(auth.RoleMember), handler.BatchDeleteAPIKeys)
	api.Post("/keys", auth.RequireRole(auth.RoleMember), handler.CreateAPIKey)
	api.Get("/keys", auth.RequireRole(auth.RoleMember), handler.ListAPIKeys)
	api.Get("/keys/:key_id/usage", auth.RequireRole(auth.RoleMember), handler.GetAPIKeyUsage)
	api.Get("/keys/:key_id/limits", auth.RequireRole(auth.RoleMember), handler.GetAPIKeyLimits)
	api.Get("/keys/:key_id", auth.RequireRole(auth.RoleMember), handler.GetAPIKey)
	api.Patch("/keys/:key_id", auth.RequireRole(auth.RoleMember), handler.UpdateAPIKey)
	api.Delete("/keys/:key_id", auth.RequireRole(auth.RoleMember), handler.DeleteAPIKey)
	api.Post("/keys/:key_id/rotate", auth.RequireRole(auth.RoleMember), handler.RotateAPIKey)

	// Models — global catalog resources managed by system admins only.
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

	// Model combo routes (product → upstream hops).
	api.Get("/models/:model_id/routes", auth.RequireRole(auth.RoleSystemAdmin), handler.ListModelRoutes)
	api.Put("/models/:model_id/routes", auth.RequireRole(auth.RoleSystemAdmin), handler.ReplaceModelRoutes)

	// Model Aliases
	api.Post("/model-aliases", auth.RequireRole(auth.RoleMember), handler.CreateModelAlias)
	api.Get("/model-aliases", auth.RequireRole(auth.RoleMember), handler.ListModelAliases)
	api.Delete("/model-aliases/:alias_id", auth.RequireRole(auth.RoleMember), handler.DeleteModelAlias)

	// Usage
	api.Get("/usage", auth.RequireRole(auth.RoleSystemAdmin), handler.SystemAdminUsage)
	api.Get("/usage/logs", auth.RequireRole(auth.RoleSystemAdmin), handler.SystemUsageLogs)
	api.Get("/usage/logs/:request_id", auth.RequireRole(auth.RoleSystemAdmin), handler.SystemUsageLogDetail)
	api.Get("/usage/profit", auth.RequireRole(auth.RoleSystemAdmin), handler.UsageProfit)
	api.Get("/usage/channels", auth.RequireRole(auth.RoleSystemAdmin), handler.UsageChannels)

	// Providers (upstream partners) — system_admin only.
	// Static sub-paths (presets, discover-models, import) are registered before
	// /:provider_id so Fiber does not treat them as provider_id parameter values.
	api.Get("/providers/usage", auth.RequireRole(auth.RoleSystemAdmin), handler.ProviderUsage)
	api.Get("/providers/presets", auth.RequireRole(auth.RoleSystemAdmin), handler.ListProviderPresets)
	api.Post("/providers/discover-models", auth.RequireRole(auth.RoleSystemAdmin), handler.DiscoverProviderModels)
	api.Post("/providers/import", auth.RequireRole(auth.RoleSystemAdmin), handler.ImportProvider)
	api.Post("/providers", auth.RequireRole(auth.RoleSystemAdmin), handler.CreateProvider)
	api.Get("/providers", auth.RequireRole(auth.RoleSystemAdmin), handler.ListProviders)
	api.Get("/providers/:provider_id/connections", auth.RequireRole(auth.RoleSystemAdmin), handler.ListProviderConnections)
	api.Post("/providers/:provider_id/connections", auth.RequireRole(auth.RoleSystemAdmin), handler.CreateProviderConnection)
	api.Post("/providers/:provider_id/connections/reorder", auth.RequireRole(auth.RoleSystemAdmin), handler.ReorderProviderConnections)
	api.Patch("/providers/:provider_id/connections/:connection_id", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateProviderConnection)
	api.Post("/providers/:provider_id/connections/:connection_id/unlock", auth.RequireRole(auth.RoleSystemAdmin), handler.UnlockProviderConnection)
	api.Delete("/providers/:provider_id/connections/:connection_id", auth.RequireRole(auth.RoleSystemAdmin), handler.DeleteProviderConnection)
	api.Post("/providers/:provider_id/connections/:connection_id/test", auth.RequireRole(auth.RoleSystemAdmin), handler.TestProviderConnection)
	api.Get("/providers/:provider_id/upstream-models", auth.RequireRole(auth.RoleSystemAdmin), handler.ListProviderUpstreamModels)
	api.Post("/providers/:provider_id/upstream-models/import", auth.RequireRole(auth.RoleSystemAdmin), handler.ImportProviderUpstreamModels)
	api.Patch("/providers/:provider_id/upstream-models/:model_id", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateProviderUpstreamModel)
	api.Delete("/providers/:provider_id/upstream-models/:model_id", auth.RequireRole(auth.RoleSystemAdmin), handler.DeleteProviderUpstreamModel)
	api.Get("/upstream-models", auth.RequireRole(auth.RoleSystemAdmin), handler.ListAllUpstreamModels)
	api.Get("/providers/:provider_id", auth.RequireRole(auth.RoleSystemAdmin), handler.GetProvider)
	api.Patch("/providers/:provider_id", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateProvider)
	api.Delete("/providers/:provider_id", auth.RequireRole(auth.RoleSystemAdmin), handler.DeleteProvider)

	// Finance monitoring — system_admin only.
	api.Get("/admin/finance/summary", auth.RequireRole(auth.RoleSystemAdmin), handler.FinanceSummary)
	api.Get("/admin/finance/topups", auth.RequireRole(auth.RoleSystemAdmin), handler.FinanceTopups)
	api.Post("/admin/finance/topups/:topup_id/review", auth.RequireRole(auth.RoleSystemAdmin), handler.FinanceReviewTopup)
	api.Get("/admin/finance/transactions", auth.RequireRole(auth.RoleSystemAdmin), handler.FinanceTransactions)

	// Top-up list (deprecated — use /admin/finance/topups).
	api.Get("/topups", auth.RequireRole(auth.RoleSystemAdmin), handler.ListTopups)
	api.Get("/users/:user_id/wallet", auth.RequireRole(auth.RoleSystemAdmin), handler.GetUserWallet)
	api.Post("/users/:user_id/wallet/adjust", auth.RequireRole(auth.RoleSystemAdmin), handler.AdjustWallet)

	// Update check — any authenticated user may read the cached update status.
	// Version info is not sensitive; no additional role gate required.
	api.Get("/system/update-check", handler.GetUpdateStatus)

	// Site branding and legal settings — system_admin only.
	api.Get("/admin/settings/site", auth.RequireRole(auth.RoleSystemAdmin), handler.GetAdminSite)
	api.Put("/admin/settings/site", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateAdminSite)
	api.Post("/admin/settings/site/logo", auth.RequireRole(auth.RoleSystemAdmin), handler.UploadSiteLogo)
	api.Delete("/admin/settings/site/logo", auth.RequireRole(auth.RoleSystemAdmin), handler.ResetSiteLogo)
	api.Get("/admin/settings/security", auth.RequireRole(auth.RoleSystemAdmin), handler.GetAdminSecuritySettings)
	api.Put("/admin/settings/security", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateAdminSecuritySettings)
	api.Get("/admin/settings/payment", auth.RequireRole(auth.RoleSystemAdmin), handler.GetAdminPaymentSettings)
	api.Put("/admin/settings/payment", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateAdminPaymentSettings)
	api.Get("/admin/settings/features", auth.RequireRole(auth.RoleSystemAdmin), handler.GetAdminFeaturesSettings)
	api.Put("/admin/settings/features", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateAdminFeaturesSettings)
	api.Get("/admin/settings/keys", auth.RequireRole(auth.RoleSystemAdmin), handler.GetAdminKeysPolicy)
	api.Put("/admin/settings/keys", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateAdminKeysPolicy)

	api.Get("/admin/settings/export", auth.RequireRole(auth.RoleSystemAdmin), handler.ExportAdminSettings)
	api.Post("/admin/settings/import/preview", auth.RequireRole(auth.RoleSystemAdmin), handler.PreviewAdminSettingsImport)
	api.Post("/admin/settings/import", auth.RequireRole(auth.RoleSystemAdmin), handler.ImportAdminSettings)

	api.Get("/admin/backups/s3-config", auth.RequireRole(auth.RoleSystemAdmin), handler.GetBackupS3Config)
	api.Put("/admin/backups/s3-config", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateBackupS3Config)
	api.Post("/admin/backups/s3-config/test", auth.RequireRole(auth.RoleSystemAdmin), handler.TestBackupS3Config)
	api.Get("/admin/backups/schedule", auth.RequireRole(auth.RoleSystemAdmin), handler.GetBackupSchedule)
	api.Put("/admin/backups/schedule", auth.RequireRole(auth.RoleSystemAdmin), handler.UpdateBackupSchedule)
	api.Post("/admin/backups", auth.RequireRole(auth.RoleSystemAdmin), handler.CreateBackup)
	api.Get("/admin/backups", auth.RequireRole(auth.RoleSystemAdmin), handler.ListBackups)
	api.Get("/admin/backups/:id", auth.RequireRole(auth.RoleSystemAdmin), handler.GetBackup)
	api.Delete("/admin/backups/:id", auth.RequireRole(auth.RoleSystemAdmin), handler.DeleteBackup)
	api.Get("/admin/backups/:id/download-url", auth.RequireRole(auth.RoleSystemAdmin), handler.GetBackupDownloadURL)
	api.Post("/admin/backups/:id/restore", auth.RequireRole(auth.RoleSystemAdmin), handler.RestoreBackup)

}
