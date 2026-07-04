// Package app manages the top-level VoidLLM server lifecycle: construction,
// startup, signal handling, and phased graceful shutdown.
package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/hkdf"

	"github.com/gofiber/fiber/v3"

	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/voidmind-io/voidllm/internal/api/admin"
	apihealth "github.com/voidmind-io/voidllm/internal/api/health"
	"github.com/voidmind-io/voidllm/internal/audit"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/cache"
	"github.com/voidmind-io/voidllm/internal/circuitbreaker"
	"github.com/voidmind-io/voidllm/internal/config"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/internal/docs"
	"github.com/voidmind-io/voidllm/internal/health"
	"github.com/voidmind-io/voidllm/internal/metrics"
	voidotel "github.com/voidmind-io/voidllm/internal/otel"
	"github.com/voidmind-io/voidllm/internal/pii"
	"github.com/voidmind-io/voidllm/internal/proxy"
	"github.com/voidmind-io/voidllm/internal/ratelimit"
	voidredis "github.com/voidmind-io/voidllm/internal/redis"
	"github.com/voidmind-io/voidllm/internal/retention"
	"github.com/voidmind-io/voidllm/internal/router"
	"github.com/voidmind-io/voidllm/internal/shutdown"
	"github.com/voidmind-io/voidllm/internal/update"
	"github.com/voidmind-io/voidllm/internal/usage"
	"github.com/voidmind-io/voidllm/internal/wallet"
	"github.com/voidmind-io/voidllm/pkg/crypto"
)

// Application is the top-level VoidLLM server lifecycle coordinator. It owns
// every long-lived dependency and orchestrates startup, signal handling, and
// phased graceful shutdown. All fields are unexported; callers interact only
// through New, Start, PrintBootstrapCredentials, and WaitForShutdown.
type Application struct {
	cfg             *config.Config
	log             *slog.Logger
	devMode         bool
	bootstrapResult *auth.BootstrapResult

	database   *db.DB
	encKey     []byte
	hmacSecret []byte

	registry    *proxy.Registry
	keyCache    *cache.Cache[string, auth.KeyInfo]
	aliasCache *proxy.AliasCache

	rateLimiter      ratelimit.Checker
	tokenCounter     *ratelimit.TokenCounter
	walletService    *wallet.Service
	upstreamLimiter  *ratelimit.UpstreamLimiter
	loginThrottle    *auth.LoginThrottle
	usageLogger      *usage.Logger
	auditLogger      *audit.Logger
	retentionCleaner *retention.Cleaner
	healthChecker    *health.Checker

	shutdownState *shutdown.State
	proxyHandler  *proxy.ProxyHandler
	adminHandler  *admin.Handler

	redisClient *voidredis.Client
	redisCancel context.CancelFunc

	proxyApp *fiber.App
	adminApp *fiber.App

	// otelShutdown flushes and closes the OTel TracerProvider on shutdown.
	// It is nil when OTel tracing is not enabled.
	otelShutdown func(context.Context) error

	// stopFuncs holds cleanup callbacks registered during Start. They are
	// invoked in LIFO order during cleanup().
	stopFuncs []func()
}

// dbUsageSeeder adapts *db.DB to the ratelimit.UsageSeeder interface. The DB
// method returns *sql.Rows (concrete type) while the interface requires
// ratelimit.RowScanner; this wrapper bridges the return type without introducing
// an import cycle between the db and ratelimit packages.
type dbUsageSeeder db.DB

func (s *dbUsageSeeder) QueryUsageSeed(ctx context.Context, since time.Time) (ratelimit.RowScanner, error) {
	return (*db.DB)(s).QueryUsageSeed(ctx, since)
}

// New constructs a fully-initialised Application by wiring all dependencies in
// the exact order required by their dependency graph. Startup order:
//
//  1. Open DB and run migrations
//  2. Parse encryption key
//  3. Sync YAML models to DB
//  4. Build registry from YAML + overlay DB models
//  5. HKDF-derive HMAC secret
//  6. Bootstrap auth, load keys into cache
//  7. Seed token counter from DB, create rate limiter
//  8. Start usage logger (and audit logger if enabled)
//  9. Load model access cache and alias cache from DB
//  10. Connect Redis (optional) and start pub/sub subscriber
//  11. Create shutdown state, proxy handler, admin handler
//
// New returns a non-nil error if any required step fails; in that case no
// goroutines have been started and no cleanup is needed by the caller.
func New(cfg *config.Config, log *slog.Logger, devMode bool) (*Application, error) {
	ctx := context.Background()

	// Step 1: open database and run migrations.
	database, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.RunMigrations(ctx, database.SQL(), database.Dialect(), log); err != nil {
		database.Close() //nolint:errcheck // best-effort on error path
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	// Declare variables that the deferred cleanup needs to reference before
	// they are assigned by the steps below.
	var (
		encKey           []byte
		hmacSecret       []byte
		usageLogger      *usage.Logger
		auditLogger      *audit.Logger
		retentionCleaner *retention.Cleaner
	)

	// From this point on, any early return must clean up in reverse order.
	// The defer fires on every return; success=true suppresses it on the
	// happy path.
	success := false
	defer func() {
		if success {
			return
		}
		if retentionCleaner != nil {
			retentionCleaner.Stop()
		}
		if usageLogger != nil {
			usageLogger.Stop()
		}
		if auditLogger != nil {
			auditLogger.Stop()
		}
		crypto.ZeroKey(hmacSecret)
		crypto.ZeroKey(encKey)
		database.Close() //nolint:errcheck // best-effort on error path
	}()

	// Step 2: parse encryption key.
	encKey, err = crypto.ParseKey(cfg.Settings.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("parse encryption key: %w", err)
	}

	// Step 3: sync YAML-configured models into the DB so they survive restarts
	// and can be discovered by the Admin API.
	if err := database.SyncYAMLModels(ctx, cfg.Models, encKey); err != nil {
		return nil, fmt.Errorf("sync YAML models: %w", err)
	}

	// Step 4a: build the in-memory registry from the YAML config.
	registry, err := proxy.NewRegistry(cfg.Models)
	if err != nil {
		return nil, fmt.Errorf("build model registry: %w", err)
	}

	// loadModelsIntoRegistry fetches all active models from the DB, decrypts
	// their API keys, and upserts each one into the registry. It is called once
	// at startup and again whenever a ChannelModels invalidation is received via
	// Redis pub/sub so that all instances stay consistent.
	loadModelsIntoRegistry := func(loadCtx context.Context) error {
		dbModels, loadErr := database.ListActiveModels(loadCtx)
		if loadErr != nil {
			return fmt.Errorf("list active models: %w", loadErr)
		}

		providerByID := make(map[string]db.Provider)
		providerKeys := make(map[string]string)
		if provs, provErr := database.ListProviders(loadCtx, "", 500); provErr != nil {
			return fmt.Errorf("list providers: %w", provErr)
		} else {
			for _, p := range provs {
				providerByID[p.ID] = p
				if p.APIKeyEncrypted != nil {
					if key, decErr := crypto.DecryptString(*p.APIKeyEncrypted, encKey, []byte("provider:"+p.ID)); decErr == nil {
						providerKeys[p.ID] = key
					}
				}
			}
		}

		// Build an id→name map from the loaded models so we can resolve
		// FallbackModelID to a name without extra DB round-trips.
		idToName := make(map[string]string, len(dbModels))
		for _, m := range dbModels {
			idToName[m.ID] = m.Name
		}

		for _, m := range dbModels {
			var apiKey string
			if m.APIKeyEncrypted != nil {
				var decErr error
				apiKey, decErr = crypto.DecryptString(*m.APIKeyEncrypted, encKey, []byte("model:"+m.ID))
				if decErr != nil {
					// A decryption failure here means the stored ciphertext is
					// corrupt or the encryption key has changed. Log and continue
					// so the remaining models are still loaded.
					log.LogAttrs(loadCtx, slog.LevelError, "failed to decrypt model api key",
						slog.String("model", m.Name),
						slog.String("error", decErr.Error()),
					)
				}
			}
			var aliases []string
			if m.Aliases != "" {
				aliases = strings.Split(m.Aliases, ",")
			}
			var timeout time.Duration
			if m.Timeout != "" {
				if d, parseErr := time.ParseDuration(m.Timeout); parseErr == nil {
					timeout = d
				} else {
					log.LogAttrs(loadCtx, slog.LevelWarn, "model: invalid timeout string, ignoring",
						slog.String("model", m.Name),
						slog.String("timeout", m.Timeout),
						slog.String("error", parseErr.Error()),
					)
				}
			}
			modelType := m.ModelType
			if modelType == "" {
				modelType = "chat"
			}

			// Load per-deployment endpoints for load-balanced models.
			dbDeps, depsErr := database.ListActiveDeployments(loadCtx, m.ID)
			if depsErr != nil {
				return fmt.Errorf("list active deployments for model %s: %w", m.Name, depsErr)
			}
			deployments := make([]proxy.Deployment, 0, len(dbDeps))
			for _, dep := range dbDeps {
				var depAPIKey string
				if dep.APIKeyEncrypted != nil {
					var decErr error
					depAPIKey, decErr = crypto.DecryptString(*dep.APIKeyEncrypted, encKey, deploymentAAD(dep.ID))
					if decErr != nil {
						log.LogAttrs(loadCtx, slog.LevelError, "failed to decrypt deployment api key",
							slog.String("model", m.Name),
							slog.String("deployment", dep.Name),
							slog.String("error", decErr.Error()),
						)
					}
				}
				depProvider := dep.Provider
				depBaseURL := dep.BaseURL
				provKey := ""
				if dep.ProviderID != nil {
					if p, ok := providerByID[*dep.ProviderID]; ok {
						provKey = providerKeys[p.ID]
						depProvider, depBaseURL, depAPIKey = db.OverlayProviderConnection(
							depProvider, depBaseURL, depAPIKey, &p, provKey,
						)
					}
				}
				providerID := ""
				if dep.ProviderID != nil {
					providerID = *dep.ProviderID
				}
				deployments = append(deployments, proxy.Deployment{
					ID:                   dep.ID,
					Name:                 dep.Name,
					Provider:             depProvider,
					BaseURL:              depBaseURL,
					APIKey:               depAPIKey,
					ProviderID:           providerID,
					AzureDeployment:      dep.AzureDeployment,
					AzureAPIVersion:      dep.AzureAPIVersion,
					GCPProject:           dep.GCPProject,
					GCPLocation:          dep.GCPLocation,
					Weight:               dep.Weight,
					Priority:             dep.Priority,
					RPMLimit:             dep.RPMLimit,
					TPMLimit:             dep.TPMLimit,
					DailyRequestLimit:    dep.DailyRequestLimit,
					CostInputPer1M:       dep.CostInputPer1M,
					CostOutputPer1M:      dep.CostOutputPer1M,
					CostCachedInputPer1M: dep.CostCachedInputPer1M,
					CostCacheWritePer1M:  dep.CostCacheWritePer1M,
					UpstreamModel:        dep.UpstreamModel,
					PIIFilter:            dep.PIIFilter,
				})
			}

			var fallbackName string
			if m.FallbackModelID != nil {
				fallbackName = idToName[*m.FallbackModelID]
			}

			strategy := m.Strategy
			if strategy == "" && len(deployments) > 0 {
				strategy = "priority"
			}

			registry.AddModel(proxy.Model{
				Name:                 m.Name,
				Provider:             m.Provider,
				Type:                 modelType,
				BaseURL:              m.BaseURL,
				APIKey:               apiKey,
				Aliases:              aliases,
				MaxContextTokens:     m.MaxContextTokens,
				Pricing:              config.PricingConfig{InputPer1M: m.InputPricePer1M, OutputPer1M: m.OutputPricePer1M},
				AzureDeployment:      m.AzureDeployment,
				AzureAPIVersion:      m.AzureAPIVersion,
				GCPProject:           m.GCPProject,
				GCPLocation:          m.GCPLocation,
				Timeout:              timeout,
				Strategy:             strategy,
				MaxRetries:           m.MaxRetries,
				Deployments:          deployments,
				FallbackModelName:    fallbackName,
				PIIFilter:            m.PIIFilter,
				SellInputPer1M:       m.SellInputPer1M,
				SellOutputPer1M:      m.SellOutputPer1M,
				SellCachedInputPer1M: m.SellCachedInputPer1M,
				BillPerToken:         m.BillPerToken,
				BillPerRequest:       m.BillPerRequest,
				SellPerRequest:       m.SellPerRequest,
			})
		}
		return nil
	}

	// Step 4b: overlay DB models on top of YAML registry.
	if err := loadModelsIntoRegistry(ctx); err != nil {
		return nil, fmt.Errorf("load models from database: %w", err)
	}

	// Warn operators who have configured fallback targets in their models but
	// have not set fallback_max_depth (which defaults to 0 = disabled). Without
	// a non-zero depth the fallback configuration is silently inactive, which is
	// a common misconfiguration after upgrading from a version that didn't have
	// this feature.
	if cfg.Settings.FallbackMaxDepth == 0 {
		modelsWithFallback := 0
		for _, m := range registry.AllModels() {
			if m.FallbackModelName != "" {
				modelsWithFallback++
			}
		}
		if modelsWithFallback > 0 {
			log.Warn("fallback chains are configured on some models but settings.fallback_max_depth is 0 (disabled). Set fallback_max_depth to enable.",
				slog.Int("models_with_fallback", modelsWithFallback))
		}
	}

	// Step 5: derive HMAC secret from the encryption key using HKDF (RFC 5869).
	// Hash: SHA-256, IKM: encKey, salt: nil, info: "voidllm-hmac-key".
	hkdfReader := hkdf.New(sha256.New, encKey, nil, []byte("voidllm-hmac-key"))
	hmacSecret = make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, hmacSecret); err != nil {
		return nil, fmt.Errorf("derive HMAC secret: %w", err)
	}

	// Step 6: bootstrap auth (creates system admin if absent), then load all
	// active API keys into the in-memory cache for hot-path lookups.
	keyCache := cache.New[string, auth.KeyInfo]()

	bootstrapResult, err := auth.Bootstrap(ctx, database.SQL(), database.Dialect(), keyCache, cfg.Settings, hmacSecret, log)
	if err != nil {
		return nil, fmt.Errorf("bootstrap auth: %w", err)
	}

	if err := auth.LoadKeysIntoCache(ctx, database, keyCache, log); err != nil {
		return nil, fmt.Errorf("load keys into cache: %w", err)
	}
	log.LogAttrs(ctx, slog.LevelInfo, "key cache seeded", slog.Int("keys", keyCache.Len()))

	// Step 7: seed token counter from recent usage events.
	tokenCounter := ratelimit.NewTokenCounter()
	if err := tokenCounter.Seed(ctx, (*dbUsageSeeder)(database)); err != nil {
		return nil, fmt.Errorf("seed token counter: %w", err)
	}

	// Step 7b: wallet service — seeds prepaid balances from the DB and feeds
	// the hot-path balance check plus the async debit pipeline.
	walletService, err := wallet.NewService(ctx, database, log)
	if err != nil {
		return nil, fmt.Errorf("create wallet service: %w", err)
	}

	// Step 8: start usage logger (always) and audit logger (if enabled).
	usageLogger = usage.NewLogger(database, cfg.Settings.Usage, log, tokenCounter)
	usageLogger.SetWallet(walletService)
	usageLogger.Start()

	metrics.RegisterDBCollectors(database.SQL())
	metrics.RegisterUsageCollector(usageLogger)

	if cfg.Settings.Audit.Enabled {
		auditLogger = audit.NewLogger(database, cfg.Settings.Audit, log)
		auditLogger.Start()
		log.LogAttrs(ctx, slog.LevelInfo, "audit logging enabled")
	}

	retentionCleaner = retention.New(database, cfg.Settings.Retention, log)
	retentionCleaner.Start()

	// Step 9: load alias cache from DB.
	aliasCache := proxy.NewAliasCache()
	orgAliases, teamAliases, err := database.LoadAllModelAliases(ctx)
	if err != nil {
		log.Error("load model aliases", slog.String("error", err.Error()))
	} else {
		aliasCache.Load(orgAliases, teamAliases)
	}

	// Step 10: connect Redis (optional). On failure, continue without Redis.
	redisCtx, redisCancel := context.WithCancel(context.Background())
	var redisClient *voidredis.Client

	if cfg.Redis.Enabled {
		var redisErr error
		redisClient, redisErr = voidredis.New(cfg.Redis.URL, cfg.Redis.KeyPrefix)
		if redisErr != nil {
			log.LogAttrs(ctx, slog.LevelError, "redis connection failed, continuing without redis",
				slog.String("error", redisErr.Error()),
			)
		} else {
			log.LogAttrs(ctx, slog.LevelInfo, "redis connected")

			// Start the pub/sub subscriber goroutine. It blocks until redisCtx
			// is canceled (on shutdown) and handles cache invalidation messages
			// published by other VoidLLM instances.
			go redisClient.SubscribeInvalidations(redisCtx, log, func(channel, payload string) {
				switch channel {
				case voidredis.ChannelKeys:
					// Payload is the key hash — evict exactly that entry.
					keyCache.Delete(payload)
					log.LogAttrs(context.Background(), slog.LevelDebug, "redis: key cache invalidated")
				case voidredis.ChannelModels:
					// Reload all active models into the registry from the DB.
					if loadErr := loadModelsIntoRegistry(context.Background()); loadErr != nil {
						log.LogAttrs(context.Background(), slog.LevelError,
							"redis: reload model registry failed",
							slog.String("error", loadErr.Error()),
						)
						return
					}
					log.LogAttrs(context.Background(), slog.LevelDebug, "redis: model registry reloaded")
				case voidredis.ChannelAliases:
					// Reload full alias cache from the database.
					orgAl, teamAl, loadErr := database.LoadAllModelAliases(context.Background())
					if loadErr != nil {
						log.LogAttrs(context.Background(), slog.LevelError,
							"redis: reload alias cache failed",
							slog.String("error", loadErr.Error()),
						)
						return
					}
					aliasCache.Load(orgAl, teamAl)
					log.LogAttrs(context.Background(), slog.LevelDebug, "redis: alias cache reloaded")
				}
			})
		}
	}

	// Select rate limiter implementation: distributed (Redis) when available,
	// in-memory otherwise. This must run after the Redis connection attempt so
	// that redisClient is non-nil only when the connection succeeded.
	var rateLimiter ratelimit.Checker
	if redisClient != nil {
		rateLimiter = ratelimit.NewRedisChecker(redisClient, log)
		log.LogAttrs(ctx, slog.LevelInfo, "rate limiting: distributed (Redis)")
	} else {
		rateLimiter = ratelimit.NewRateLimiter()
		log.LogAttrs(ctx, slog.LevelInfo, "rate limiting: in-memory (single instance)")
	}

	// Step 11: create shutdown state, proxy handler, admin handler.
	shutdownState := shutdown.New()

	var cbRegistry *circuitbreaker.Registry
	if cfg.Settings.CircuitBreaker.Enabled {
		cbRegistry = circuitbreaker.NewRegistry(circuitbreaker.Config{
			Enabled:     true,
			Threshold:   cfg.Settings.CircuitBreaker.Threshold,
			Timeout:     cfg.Settings.CircuitBreaker.Timeout,
			HalfOpenMax: cfg.Settings.CircuitBreaker.HalfOpenMax,
		})
		log.LogAttrs(ctx, slog.LevelInfo, "circuit breaker enabled",
			slog.Int("threshold", cfg.Settings.CircuitBreaker.Threshold),
			slog.Duration("timeout", cfg.Settings.CircuitBreaker.Timeout),
		)
	}

	// OTel tracing: initialise only when the feature is licensed and enabled.
	var otelShutdownFn func(context.Context) error
	var tracer trace.Tracer
	if cfg.Settings.OTel.Enabled {
		var setupErr error
		otelShutdownFn, setupErr = voidotel.Setup(ctx,
			cfg.Settings.OTel.Endpoint,
			cfg.Settings.OTel.Insecure,
			*cfg.Settings.OTel.SampleRate,
			"voidllm", apihealth.Version,
		)
		if setupErr != nil {
			log.LogAttrs(ctx, slog.LevelWarn, "otel setup failed, tracing disabled",
				slog.String("error", setupErr.Error()),
			)
		} else {
			tracer = otelapi.Tracer("voidllm.proxy")
			log.LogAttrs(ctx, slog.LevelInfo, "opentelemetry tracing enabled",
				slog.String("endpoint", cfg.Settings.OTel.Endpoint),
				slog.Float64("sample_rate", *cfg.Settings.OTel.SampleRate),
			)
		}
	}

	// Build the health checker when at least one probe level is enabled.
	var healthChecker *health.Checker
	hcCfg := cfg.Settings.HealthCheck
	if hcCfg.Health.Enabled || hcCfg.Models.Enabled || hcCfg.Functional.Enabled {
		healthChecker = health.NewChecker(registry, hcCfg, log)
	}
	if hcCfg.Functional.Enabled {
		log.LogAttrs(ctx, slog.LevelWarn,
			"functional health probe enabled — sends billable requests to upstream providers",
			slog.Duration("interval", hcCfg.Functional.Interval),
		)
	}

	// Create the deployment router for load balancing. Both dependencies are
	// optional: healthChecker may be nil when health probes are disabled, and
	// cbRegistry may be nil when circuit breaking is disabled.
	modelRouter := router.NewRouter(healthChecker, cbRegistry)

	upstreamLimiter := ratelimit.NewUpstreamLimiter()
	modelRouter.SetUpstreamLimiter(upstreamLimiter)

	proxyHandler := proxy.NewProxyHandler(registry, log)
	proxyHandler.AliasCache = aliasCache
	proxyHandler.CircuitBreakers = cbRegistry
	proxyHandler.Router = modelRouter
	proxyHandler.UsageLogger = usageLogger
	proxyHandler.RateLimiter = rateLimiter
	proxyHandler.TokenCounter = tokenCounter
	proxyHandler.Wallet = walletService
	proxyHandler.UpstreamLimiter = upstreamLimiter
	proxyHandler.ShutdownState = shutdownState
	proxyHandler.Tracer = tracer
	proxyHandler.MaxRequestBody = cfg.Server.Proxy.MaxRequestBody
	proxyHandler.MaxResponseBody = cfg.Server.Proxy.MaxResponseBody
	proxyHandler.MaxStreamDuration = cfg.Server.Proxy.MaxStreamDuration
	proxyHandler.FallbackMaxDepth = cfg.Settings.FallbackMaxDepth

	if cfg.Settings.PII.IsEnabled() {
		piiEngine, piiErr := buildPIIEngine(encKey, cfg.Settings.PII, log)
		if piiErr != nil {
			redisCancel()
			return nil, fmt.Errorf("pii engine: %w", piiErr)
		}
		proxyHandler.PIIEngine = piiEngine
	}

	loginThrottle := auth.NewLoginThrottle()

	adminHandler := &admin.Handler{
		DB:            database,
		HMACSecret:    hmacSecret,
		EncryptionKey: encKey,
		KeyCache:      keyCache,
		Registry:      registry,
		AliasCache: aliasCache,
		Redis:         redisClient,
		AuditLogger:   auditLogger,
		Log:           log,
		LoginThrottle: loginThrottle,
		Wallet:        walletService,
	}
	// Wire the in-process reload callback so deployment mutations can re-gate the
	// model registry immediately after storing a new license, even on
	// deployments that have no Redis (the default single-instance setup).
	adminHandler.ReloadModels = func(reloadCtx context.Context) error {
		return loadModelsIntoRegistry(reloadCtx)
	}
	// Only assign the health checker when it was actually created — a typed nil
	// (*health.Checker)(nil) satisfies the interface but is NOT == nil when
	// checked as ModelHealthProvider, causing nil-pointer panics.
	if healthChecker != nil {
		adminHandler.HealthChecker = healthChecker
	}

	success = true
	return &Application{
		cfg:              cfg,
		log:              log,
		devMode:          devMode,
		bootstrapResult:  bootstrapResult,
		database:         database,
		encKey:           encKey,
		hmacSecret:       hmacSecret,
		registry:         registry,
		keyCache:         keyCache,
		aliasCache: aliasCache,
		rateLimiter:      rateLimiter,
		tokenCounter:     tokenCounter,
		walletService:    walletService,
		upstreamLimiter:  upstreamLimiter,
		loginThrottle:    loginThrottle,
		usageLogger:      usageLogger,
		auditLogger:      auditLogger,
		retentionCleaner: retentionCleaner,
		healthChecker:    healthChecker,
		shutdownState:    shutdownState,
		proxyHandler:     proxyHandler,
		adminHandler:     adminHandler,
		redisClient:      redisClient,
		redisCancel:      redisCancel,
		otelShutdown:     otelShutdownFn,
	}, nil
}

// Start launches background goroutines (cache refresh tickers, pprof if dev
// mode) and begins listening on the configured port(s). Start must be called
// exactly once after New returns successfully.
//
// Listener errors are handled asynchronously; the error return is reserved for
// future synchronous startup checks and currently always returns nil.
func (a *Application) Start() error {
	// Cache refresh tickers. Stop functions are registered in LIFO order so
	// that the key refresh stops first on shutdown (matching startup order).
	a.stopFuncs = append(a.stopFuncs,
		auth.StartCacheRefresh(a.database, a.keyCache, a.cfg.Cache.KeyTTL, a.log),
		startTicker(a.cfg.Cache.AliasTTL, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			orgAliases, teamAliases, err := a.database.LoadAllModelAliases(ctx)
			if err != nil {
				a.log.LogAttrs(context.Background(), slog.LevelError, "alias cache refresh failed",
					slog.String("error", err.Error()),
				)
				return
			}
			a.aliasCache.Load(orgAliases, teamAliases)
		}),
		startTicker(5*time.Minute, func() {
			a.tokenCounter.EvictStale()
			a.loginThrottle.EvictStale()
			a.upstreamLimiter.EvictStale()
		}),
		startTicker(30*time.Second, func() {
			metrics.CacheSize.WithLabelValues("keys").Set(float64(a.keyCache.Len()))
			metrics.CacheSize.WithLabelValues("aliases").Set(float64(a.aliasCache.Len()))
		}),
	)

	// Register retention cleaner stop so it is halted during graceful shutdown.
	// LIFO ordering ensures retention stops before the usage and audit loggers.
	a.stopFuncs = append(a.stopFuncs, a.retentionCleaner.Stop)

	// The in-memory rate limiter accumulates counter entries that must be
	// periodically evicted to reclaim memory. The Redis-backed checker uses
	// TTL-keyed counters that self-expire, so no eviction goroutine is needed.
	if memRL, ok := a.rateLimiter.(*ratelimit.RateLimiter); ok {
		a.stopFuncs = append(a.stopFuncs, startTicker(5*time.Minute, memRL.EvictStale))
	}

	// Start upstream model health monitoring when at least one probe is enabled.
	if a.healthChecker != nil {
		a.stopFuncs = append(a.stopFuncs, a.healthChecker.Start())
	}

	// Start update checker when running a real build (not dev).
	// The checker fires after a 5-minute initial delay to keep startup fast.
	if apihealth.Version != "dev" {
		updateChecker := update.NewChecker(apihealth.Version, a.database, a.log)
		a.stopFuncs = append(a.stopFuncs, updateChecker.Start())
		a.adminHandler.UpdateChecker = updateChecker
	}

	// pprof profiling is enabled in dev mode. Handlers are registered on a
	// dedicated ServeMux so they are never reachable unless dev mode is
	// explicitly enabled. Always bound to localhost — never exposed externally.
	if a.devMode {
		pprofMux := http.NewServeMux()
		pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
		pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		pprofServer := &http.Server{
			Addr:              "localhost:6060",
			Handler:           pprofMux,
			ReadHeaderTimeout: 10 * time.Second,
		}
		a.log.LogAttrs(context.Background(), slog.LevelInfo, "pprof enabled (dev mode)",
			slog.String("addr", "localhost:6060"),
		)
		go func() {
			if err := pprofServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				a.log.LogAttrs(context.Background(), slog.LevelError, "pprof server stopped",
					slog.String("error", err.Error()),
				)
			}
		}()
		a.stopFuncs = append(a.stopFuncs, func() {
			pprofServer.Close() //nolint:errcheck // best-effort on shutdown
		})
	}

	// Populate Swagger spec metadata. Host is intentionally empty so that the
	// Swagger UI uses the current origin — no hard-coded address required.
	docs.SwaggerInfo.Title = "VoidLLM API"
	docs.SwaggerInfo.Description = "Lightweight LLM proxy with org/team/user hierarchy"
	docs.SwaggerInfo.Version = "0.2.0"
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Host = ""
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	a.setupRoutes()
	a.startListening()
	return nil
}

// WaitForShutdown blocks until SIGINT or SIGTERM is received, then performs a
// phased graceful shutdown:
//
//  1. Begin drain — signals load balancers via /readyz to stop sending traffic.
//  2. Wait for in-flight requests to finish (up to DrainTimeout).
//  3. Force-cancel any remaining requests if the timeout expires.
//  4. Stop the Fiber server(s).
//  5. LIFO cleanup: stop tickers, flush usage/audit loggers, close Redis, close DB.
//  6. Zero sensitive key material from memory.
//
// A second signal received while draining triggers an immediate os.Exit(1).
// ctx is reserved for future use and may be context.Background().
func (a *Application) WaitForShutdown(ctx context.Context) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	a.log.LogAttrs(ctx, slog.LevelInfo, "shutdown signal received, starting drain",
		slog.String("signal", sig.String()),
	)

	// A second signal bypasses the drain and exits immediately.
	go func() {
		sig := <-sigCh
		a.log.LogAttrs(ctx, slog.LevelWarn, "second signal received, forcing immediate exit (buffered usage events will be lost)",
			slog.String("signal", sig.String()),
		)
		os.Exit(1)
	}()

	// Phase 1: Begin drain — /readyz returns 503 from this point forward so
	// load balancers stop routing new requests to this instance.
	a.shutdownState.BeginDrain()

	// Phase 2: Wait for in-flight requests to finish, logging progress every
	// 5 seconds so operators can monitor drain progress.
	drainDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-drainDone:
				return
			case <-ticker.C:
				a.log.LogAttrs(ctx, slog.LevelInfo, "drain in progress",
					slog.Int64("in_flight", a.shutdownState.InFlight()),
				)
			}
		}
	}()

	drained := a.shutdownState.WaitForDrain(a.cfg.Server.Proxy.DrainTimeout)
	close(drainDone)

	if drained {
		a.log.LogAttrs(ctx, slog.LevelInfo, "all requests drained")
	} else {
		// Phase 3: Force-cancel remaining in-flight requests.
		a.log.LogAttrs(ctx, slog.LevelWarn, "drain timeout exceeded, canceling in-flight requests",
			slog.Int64("in_flight", a.shutdownState.InFlight()),
		)
		a.shutdownState.CancelInflight()
		time.Sleep(500 * time.Millisecond)
	}

	// Phase 4: Stop the Fiber server(s).
	if err := a.proxyApp.Shutdown(); err != nil {
		a.log.LogAttrs(ctx, slog.LevelError, "proxy shutdown error",
			slog.String("error", err.Error()),
		)
	}

	if a.adminApp != nil {
		if err := a.adminApp.Shutdown(); err != nil {
			a.log.LogAttrs(ctx, slog.LevelError, "admin shutdown error",
				slog.String("error", err.Error()),
			)
		}
	}

	// Phase 5: cleanup resources.
	a.cleanup(ctx)

	a.log.LogAttrs(ctx, slog.LevelInfo, "shutdown complete")
}

// PrintBootstrapCredentials writes the bootstrap credentials to stderr when a
// bootstrap was performed during startup. It must be called after Start so that
// the Fiber server banner has already been printed, preventing interleaving.
// If no bootstrap occurred this is a no-op.
//
// Intentional use of fmt.Fprintln instead of slog: the plaintext key and
// password must be shown to the operator on stderr exactly once but must NOT
// go through structured logging where they could be captured by log aggregation
// systems (ELK, Datadog, CloudWatch).
func (a *Application) PrintBootstrapCredentials() {
	if a.bootstrapResult == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "========================================")
	fmt.Fprintln(os.Stderr, " BOOTSTRAP COMPLETE — COPY THESE NOW")
	fmt.Fprintln(os.Stderr, "========================================")
	fmt.Fprintf(os.Stderr, "  API Key:    %s\n", a.bootstrapResult.APIKey)
	fmt.Fprintf(os.Stderr, "  Email:      %s\n", a.bootstrapResult.Email)
	fmt.Fprintf(os.Stderr, "  Password:   %s\n", a.bootstrapResult.Password)
	fmt.Fprintln(os.Stderr, "========================================")
	fmt.Fprintln(os.Stderr, "")
	a.bootstrapResult = nil
}

// deploymentAAD returns the additional authenticated data used when encrypting
// or decrypting a deployment API key. The AAD binds the ciphertext to the
// specific deployment row so that a ciphertext from one row cannot be replayed
// against a different row.
func deploymentAAD(id string) []byte {
	return []byte("deployment:" + id)
}

// isCodeModeTimeout reports whether a Code Mode execution error message
// indicates the script exceeded its wall-clock time limit.
func isCodeModeTimeout(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "interrupted") || strings.Contains(lower, "timeout")
}

// isCodeModeOOM reports whether a Code Mode execution error message indicates
// the script exceeded its memory limit inside the WASM sandbox.
func isCodeModeOOM(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "out of memory") || strings.Contains(lower, "stack overflow")
}

// cleanup tears down resources in reverse startup order: Redis pub/sub,
// background tickers (LIFO), usage/audit loggers, Redis connection, database.
// Sensitive key material is zeroed last, after all components that might still
// need to decrypt data have been stopped. cleanup must be called exactly once,
// by WaitForShutdown.
func (a *Application) cleanup(ctx context.Context) {
	// Cancel Redis pub/sub subscriber before stopping the client.
	a.redisCancel()
	if a.redisClient != nil {
		if err := a.redisClient.Close(); err != nil {
			a.log.LogAttrs(ctx, slog.LevelError, "redis close error",
				slog.String("error", err.Error()),
			)
		}
	}

	// Stop background tickers in LIFO order.
	for i := len(a.stopFuncs) - 1; i >= 0; i-- {
		a.stopFuncs[i]()
	}

	// Flush buffered usage and audit events.
	a.usageLogger.Stop()
	if a.auditLogger != nil {
		a.auditLogger.Stop()
	}

	// Flush buffered OTel spans. Use a bounded context so a slow collector
	// cannot block shutdown indefinitely.
	if a.otelShutdown != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := a.otelShutdown(shutdownCtx); err != nil {
			a.log.LogAttrs(ctx, slog.LevelWarn, "otel shutdown error",
				slog.String("error", err.Error()),
			)
		}
		shutdownCancel()
	}

	if err := a.database.Close(); err != nil {
		a.log.LogAttrs(ctx, slog.LevelError, "database close error",
			slog.String("error", err.Error()),
		)
	}

	// Zero PII Engine secret material before zeroing the main keys.
	// The Engine holds a copy of the derived pseudonym secret; Close zeroes it.
	if a.proxyHandler != nil && a.proxyHandler.PIIEngine != nil {
		a.proxyHandler.PIIEngine.Close()
	}

	// Zero sensitive key material after all components are stopped.
	crypto.ZeroKey(a.hmacSecret)
	crypto.ZeroKey(a.encKey)
}

// buildPIIEngine constructs a pii.Engine from the installation encryption key
// and the PII configuration. The pseudonym secret is derived from encKey using
// HKDF with the info string "voidllm-pii-pseudonym-v1", keeping it independent
// from the key-hashing HMAC secret ("voidllm-hmac-key") and preventing cross-
// subsystem secret reuse.
//
// Returns an error on any failure (HKDF error, bad custom regexp). The caller
// must treat an error as a startup failure — running without a functional PII
// engine when PII is enabled would silently drop the anonymization guarantee.
// The local piiSecret copy is zeroed after the engine is constructed so that
// the derived secret does not linger in the Go heap beyond its required lifetime.
func buildPIIEngine(encKey []byte, cfg config.PIIConfig, log *slog.Logger) (*pii.Engine, error) {
	// Derive a 32-byte pseudonym secret via HKDF-SHA256.
	hkdfReader := hkdf.New(sha256.New, encKey, nil, []byte("voidllm-pii-pseudonym-v1"))
	piiSecret := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, piiSecret); err != nil {
		return nil, fmt.Errorf("derive pseudonym secret: %w", err)
	}
	// Zero the local copy once the engine has taken its own internal copy.
	defer crypto.ZeroKey(piiSecret)

	// Build detector list: start with built-in defaults.
	patterns := pii.DefaultPatterns()

	// Append custom patterns from config.
	for _, cp := range cfg.Patterns {
		patterns = append(patterns, pii.Pattern{Type: cp.Type, Regexp: cp.Regexp})
	}

	detector, err := pii.NewRegexDetector(patterns)
	if err != nil {
		return nil, fmt.Errorf("compile detector patterns: %w", err)
	}

	detectors := []pii.Detector{detector}

	// When the gazetteer sub-system is also enabled, build and append it.
	// The gazetteer detector is built at startup and adds zero overhead when
	// disabled (it is simply not constructed and not appended to the slice).
	if cfg.Gazetteer.IsEnabled() {
		gazDetector, gazErr := pii.LoadGazetteerDetector(cfg.Gazetteer)
		if gazErr != nil {
			return nil, fmt.Errorf("build gazetteer detector: %w", gazErr)
		}
		detectors = append(detectors, gazDetector)
		log.LogAttrs(context.Background(), slog.LevelInfo,
			"pii gazetteer detector enabled",
			slog.Int("packs", len(cfg.Gazetteer.Packs)),
			slog.Int("dirs", len(cfg.Gazetteer.Dirs)),
			slog.Int("inline_term_groups", len(cfg.Gazetteer.Terms)),
		)
	}

	log.LogAttrs(context.Background(), slog.LevelInfo,
		"pii anonymization enabled",
		slog.Int("builtin_patterns", len(pii.DefaultPatterns())),
		slog.Int("custom_patterns", len(cfg.Patterns)),
	)
	return pii.NewEngine(piiSecret, detectors), nil
}
