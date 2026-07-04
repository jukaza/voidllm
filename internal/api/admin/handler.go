// Package admin provides HTTP handlers for the VoidLLM Admin API.
package admin

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/audit"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/cache"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/internal/health"
	"github.com/voidmind-io/voidllm/internal/proxy"
	voidredis "github.com/voidmind-io/voidllm/internal/redis"
	"github.com/voidmind-io/voidllm/internal/update"
	"github.com/voidmind-io/voidllm/internal/wallet"
)

// ModelHealthProvider provides upstream model health status for the admin API.
// It is implemented by *health.Checker and may be nil when health monitoring
// is not enabled.
type ModelHealthProvider interface {
	GetAllHealth() []health.ModelHealth
}

// Handler holds shared dependencies for all admin API handlers.
type Handler struct {
	// fallbackMu serializes fallback mutations (cycle-check + DB write) to make
	// them atomic at the process level. Acquired only when a CreateModel or
	// UpdateModel request includes a fallback_model_name change.
	//
	// Multi-instance cluster-wide serialization would require DB-level locking
	// (SELECT FOR UPDATE / advisory lock). For single-instance and typical
	// enterprise deployments the process-level mutex is sufficient.
	fallbackMu sync.Mutex

	DB            *db.DB
	HMACSecret    []byte
	EncryptionKey []byte // AES-256-GCM key for upstream API key encryption
	KeyCache      *cache.Cache[string, auth.KeyInfo]
	Registry      *proxy.Registry
	AliasCache    *proxy.AliasCache // in-memory model alias cache; nil disables refresh
	Redis         *voidredis.Client       // nil when Redis is not configured
	AuditLogger   *audit.Logger           // nil when audit logging is disabled
	Log           *slog.Logger
	// HealthChecker provides upstream model health status. Nil when health
	// monitoring is not enabled.
	HealthChecker ModelHealthProvider
	// LoginThrottle enforces per-IP and per-account brute-force protection on
	// the login endpoint. Nil disables throttling (test environments only).
	LoginThrottle *auth.LoginThrottle
	// Wallet is the prepaid wallet service. Nil disables wallet features
	// (signup still creates the DB wallet row).
	Wallet *wallet.Service
	// UpdateChecker provides cached update status read from the settings table.
	// Nil in dev builds (Version == "dev") — GetUpdateStatus returns a static
	// response in that case.
	UpdateChecker *update.Checker
	// ReloadModels triggers an in-process rebuild of the model registry.
	// Called after model/deployment mutations to apply changes immediately,
	// independent of Redis pub/sub.
	// Must be safe to call concurrently. May be nil — callers must nil-check.
	ReloadModels func(context.Context) error
}

// swaggerErrorResponse is the standard API error envelope used in OpenAPI docs.
// It is an alias for apierror.SwaggerResponse kept here for Swagger annotation compatibility.
// The alias is referenced only in swagger @Failure comments (invisible to staticcheck).
//
//lint:ignore U1000 referenced in swagger @Failure annotations which staticcheck cannot see
type swaggerErrorResponse = apierror.SwaggerResponse

// paginationParams holds the parsed cursor and limit for paginated list endpoints.
type paginationParams struct {
	Limit  int
	Cursor string
}

// parsePagination extracts and clamps pagination query parameters from the request.
// limit defaults to 20 and is clamped to [1, 100].
// cursor is a raw UUIDv7 string used as a keyset pagination lower bound.
// An error is returned if cursor is non-empty but not a valid UUID.
func parsePagination(c fiber.Ctx) (paginationParams, error) {
	limit := fiber.Query[int](c, "limit", 20)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	cursor := c.Query("cursor", "")
	if cursor != "" {
		if _, err := uuid.Parse(cursor); err != nil {
			return paginationParams{}, fmt.Errorf("invalid cursor format")
		}
	}
	return paginationParams{Limit: limit, Cursor: cursor}, nil
}


