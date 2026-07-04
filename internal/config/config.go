// Package config handles loading, environment variable interpolation, defaulting,
// and validation of the VoidLLM configuration file (voidllm.yaml).
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration structure for VoidLLM.
type Config struct {
	Server     ServerConfig      `yaml:"server"`
	Database   DatabaseConfig    `yaml:"database"`
	Cache      CacheConfig       `yaml:"cache"`
	Redis      RedisConfig       `yaml:"redis"`
	Models     []ModelConfig     `yaml:"models"`
	Settings   SettingsConfig    `yaml:"settings"`
	Logging    LoggingConfig     `yaml:"logging"`
}

// ServerConfig holds configuration for both the proxy and admin HTTP servers.
type ServerConfig struct {
	Proxy ProxyConfig `yaml:"proxy"`
	Admin AdminConfig `yaml:"admin"`
}

// ProxyConfig holds configuration for the proxy (hot path) HTTP server.
type ProxyConfig struct {
	Port              int           `yaml:"port"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	MaxRequestBody    int           `yaml:"max_request_body"`    // bytes, 0 = use default
	MaxResponseBody   int           `yaml:"max_response_body"`   // bytes, 0 = use default
	MaxStreamDuration time.Duration `yaml:"max_stream_duration"` // 0 = use default (5m)
	DrainTimeout      time.Duration `yaml:"drain_timeout"`       // graceful shutdown drain window; default 25s
}

// AdminConfig holds configuration for the admin HTTP server.
type AdminConfig struct {
	Port int       `yaml:"port"`
	TLS  TLSConfig `yaml:"tls"`
}

// TLSConfig holds TLS certificate configuration for the admin server.
type TLSConfig struct {
	Enabled bool   `yaml:"enabled"`
	Cert    string `yaml:"cert"`
	Key     string `yaml:"key"`
}

// DatabaseConfig holds configuration for the primary data store.
type DatabaseConfig struct {
	Driver          string        `yaml:"driver"`
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// LogValue implements slog.LogValuer to prevent the DSN (which may contain
// credentials) from appearing in logs.
func (d DatabaseConfig) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("driver", d.Driver),
		slog.String("dsn", "[REDACTED]"),
		slog.Int("max_open_conns", d.MaxOpenConns),
		slog.Int("max_idle_conns", d.MaxIdleConns),
	)
}

// CacheConfig holds TTL settings for the in-memory cache.
type CacheConfig struct {
	KeyTTL   time.Duration `yaml:"key_ttl"`
	ModelTTL time.Duration `yaml:"model_ttl"`
	AliasTTL time.Duration `yaml:"alias_ttl"`
}

// RedisConfig holds configuration for the optional Redis integration.
type RedisConfig struct {
	Enabled   bool   `yaml:"enabled"`
	URL       string `yaml:"url"`
	KeyPrefix string `yaml:"key_prefix"`
}

// DeploymentConfig defines a single deployment endpoint within a multi-deployment model.
// When a ModelConfig has one or more deployments, the proxy selects among them using
// the strategy defined on the parent ModelConfig.
type DeploymentConfig struct {
	// Name is a unique identifier for this deployment within the model (required).
	Name string `yaml:"name"`
	// Provider is the upstream provider for this deployment (required).
	Provider string `yaml:"provider"`
	// BaseURL is the base URL for this deployment's API endpoint (required).
	BaseURL string `yaml:"base_url"`
	// APIKey is the API key for this deployment. Redacted in logs.
	APIKey string `yaml:"api_key" json:"-"`
	// AzureDeployment is the Azure deployment name. Required when provider is "azure".
	AzureDeployment string `yaml:"azure_deployment"`
	// AzureAPIVersion is the Azure API version string used in request URLs.
	AzureAPIVersion string `yaml:"azure_api_version"`
	// GCPProject is the Google Cloud project ID. Required when provider is "vertex".
	GCPProject string `yaml:"gcp_project" json:"gcp_project,omitempty"`
	// GCPLocation is the Google Cloud region (e.g. "us-central1"). Required when provider is "vertex".
	GCPLocation string `yaml:"gcp_location" json:"gcp_location,omitempty"`
	// Weight is the relative routing weight for the "weighted" strategy.
	// A value of 0 means this deployment is only used as a fallback when all
	// weighted deployments are unavailable.
	Weight int `yaml:"weight"`
	// Priority is the preference rank for the "priority" strategy. Lower
	// values indicate higher priority.
	Priority int `yaml:"priority"`
	// PIIFilter explicitly enables or disables PII anonymization for requests
	// routed to this deployment. When set it overrides both the model-level
	// pii_filter and the network-based default (see ModelConfig.PIIFilter).
	// nil means "not set — defer to model-level flag or network default".
	//
	// UI/DB persistence of this flag is a follow-up; for now it is YAML-only.
	PIIFilter *bool `yaml:"pii_filter"`
}

// LogValue implements slog.LogValuer to prevent API keys from appearing in logs.
func (d DeploymentConfig) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("name", d.Name),
		slog.String("provider", d.Provider),
		slog.String("base_url", d.BaseURL),
		slog.String("api_key", "[REDACTED]"),
	)
}

// ModelConfig defines a single model entry in the static model registry.
type ModelConfig struct {
	Name     string `yaml:"name"`
	Provider string `yaml:"provider"`
	// "completion", "image", "audio_transcription", or "tts". Defaults to "chat".
	Type             string        `yaml:"type"`
	BaseURL          string        `yaml:"base_url"`
	APIKey           string        `yaml:"api_key" json:"-"`
	Aliases          []string      `yaml:"aliases"`
	MaxContextTokens int           `yaml:"max_context_tokens"`
	Pricing          PricingConfig `yaml:"pricing"`
	AzureDeployment  string        `yaml:"azure_deployment"`
	AzureAPIVersion  string        `yaml:"azure_api_version"`
	// GCPProject is the Google Cloud project ID. Required when provider is "vertex".
	GCPProject string `yaml:"gcp_project" json:"gcp_project,omitempty"`
	// GCPLocation is the Google Cloud region (e.g. "us-central1"). Required when provider is "vertex".
	GCPLocation string `yaml:"gcp_location" json:"gcp_location,omitempty"`
	// Timeout is the per-model upstream timeout as a duration string (e.g. "30s",
	// "2m"). When non-empty it overrides the global stream/response timeout for
	// this model. Zero or empty means use the global default.
	Timeout string `yaml:"timeout"`
	// Strategy is the deployment selection strategy used when Deployments is
	// non-empty. Valid values: round-robin, least-latency, weighted, priority.
	Strategy string `yaml:"strategy"`
	// MaxRetries is the number of times the proxy will retry a failed upstream
	// request across the available deployments. Must be >= 0.
	MaxRetries int `yaml:"max_retries"`
	// Fallback is the canonical name (or alias) of another model to retry
	// when all deployments of this model are unavailable. Resolved at
	// registry build time. Empty disables fallback for this model.
	Fallback string `yaml:"fallback" json:"fallback,omitempty"`
	// Deployments is the list of backend endpoints for this model. When set,
	// the model-level Provider and BaseURL fields are ignored in favour of the
	// per-deployment values, and Strategy must be set.
	Deployments []DeploymentConfig `yaml:"deployments"`
	// PIIFilter explicitly enables or disables PII anonymization for requests
	// routed to this model. When set it overrides the network-based default
	// (public destination = anonymize, private destination = pass through).
	// A per-deployment pii_filter takes precedence over this model-level flag.
	// nil means "not set — use the network-based default".
	//
	// UI/DB persistence of this flag is a follow-up; for now it is YAML-only.
	PIIFilter *bool `yaml:"pii_filter"`
}

// LogValue implements slog.LogValuer to prevent API keys from appearing in logs.
func (m ModelConfig) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("name", m.Name),
		slog.String("provider", m.Provider),
		slog.String("base_url", m.BaseURL),
		slog.String("api_key", "[REDACTED]"),
	)
}

// PricingConfig holds per-million-token pricing for a model.
type PricingConfig struct {
	InputPer1M  float64 `yaml:"input_per_1m"`
	OutputPer1M float64 `yaml:"output_per_1m"`
	// CachedInputPer1M is the cost per 1M cache-hit prompt tokens. Zero means
	// bill cache hits at InputPer1M.
	CachedInputPer1M float64 `yaml:"cached_input_per_1m"`
	// CacheWritePer1M is the cost per 1M cache-write tokens (Anthropic-style
	// prompt caching bills writes above the input rate). Zero means cache
	// writes are not billed separately.
	CacheWritePer1M float64 `yaml:"cache_write_per_1m"`
}

// LoggingConfig controls log output level and format.
type LoggingConfig struct {
	// Level sets the minimum log level. Valid values: debug, info, warn, error.
	Level string `yaml:"level"`
	// Format sets the log output format. Valid values: json (default), text (local dev).
	Format string `yaml:"format"`
}

// BootstrapConfig controls the initial system-admin user created on first
// startup when settings.admin_key is set and the database is empty.
type BootstrapConfig struct {
	// AdminEmail is the email address of the initial system-admin user.
	// Defaults to "admin@voidllm.local".
	AdminEmail string `yaml:"admin_email"`
}

// AuditConfig controls the enterprise audit logging subsystem.
type AuditConfig struct {
	// Enabled controls whether audit events are recorded. Defaults to false.
	Enabled bool `yaml:"enabled"`
	// BufferSize sets the capacity of the async event channel. Defaults to 500.
	BufferSize int `yaml:"buffer_size"`
	// FlushInterval is how often buffered events are written to the database.
	// Defaults to 5 seconds.
	FlushInterval time.Duration `yaml:"flush_interval"`
}

// CircuitBreakerConfig holds per-model circuit breaker configuration.
type CircuitBreakerConfig struct {
	// Enabled activates circuit breaker functionality. Defaults to false.
	Enabled bool `yaml:"enabled"`
	// Threshold is the number of consecutive upstream failures required before
	// the circuit opens and starts rejecting requests. Defaults to 5.
	Threshold int `yaml:"threshold"`
	// Timeout is how long the circuit stays open before transitioning to
	// half-open to probe for recovery. Defaults to 30 seconds.
	Timeout time.Duration `yaml:"timeout"`
	// HalfOpenMax is the maximum number of concurrent probe requests allowed
	// while the circuit is in half-open state. Defaults to 1.
	HalfOpenMax int `yaml:"half_open_max"`
}

// OTelConfig holds OpenTelemetry tracing configuration. Tracing is an
// enterprise feature; the Enabled flag is ignored unless a valid enterprise
// license with the otel_tracing feature is present.
type OTelConfig struct {
	// Enabled activates OpenTelemetry trace export. Requires an enterprise
	// license with the otel_tracing feature.
	Enabled bool `yaml:"enabled"`
	// Endpoint is the OTLP/gRPC collector address (host:port). Defaults to
	// "localhost:4317".
	Endpoint string `yaml:"endpoint"`
	// Insecure disables TLS on the gRPC connection to the collector. Suitable
	// for local collectors running without TLS (e.g. Jaeger all-in-one).
	Insecure bool `yaml:"insecure"`
	// SampleRate is the fraction of traces to export, in the range [0.0, 1.0].
	// 1.0 exports all traces; 0.0 exports none. Defaults to 1.0.
	// A pointer is used so that an explicit 0.0 can be distinguished from
	// the zero value after unmarshalling.
	SampleRate *float64 `yaml:"sample_rate"`
}

// HealthCheckConfig holds configuration for the upstream model health monitoring subsystem.
type HealthCheckConfig struct {
	// Health configures the lightweight GET / reachability probe.
	Health HealthProbeConfig `yaml:"health"`
	// Models configures the GET /models API availability probe.
	Models HealthProbeConfig `yaml:"models"`
	// Functional configures the POST /chat/completions end-to-end probe.
	Functional HealthProbeConfig `yaml:"functional"`
}

// HealthProbeConfig holds the enable flag and polling interval for a single
// health probe level.
type HealthProbeConfig struct {
	// Enabled controls whether this probe level is active.
	Enabled bool `yaml:"enabled"`
	// Interval is how often the probe is executed for each registered model.
	Interval time.Duration `yaml:"interval"`
}

// RetentionConfig controls periodic deletion of old usage and audit records.
// Retention is opt-in: a zero duration means "keep forever".
type RetentionConfig struct {
	// UsageEvents is the maximum age of rows in the usage_events table.
	// A zero value means rows are kept forever.
	UsageEvents time.Duration `yaml:"usage_events" json:"usage_events"`
	// AuditLogs is the maximum age of rows in the audit_logs table.
	// A zero value means rows are kept forever.
	AuditLogs time.Duration `yaml:"audit_logs" json:"audit_logs"`
	// Interval controls how often the cleanup job runs.
	// Defaults to 24h when retention is enabled.
	Interval time.Duration `yaml:"interval" json:"interval"`
}

// Enabled reports whether any retention job is active.
func (r RetentionConfig) Enabled() bool {
	return r.UsageEvents > 0 || r.AuditLogs > 0
}

// PIIPatternConfig defines a single custom PII detection pattern to add on
// top of the built-in defaults. Both fields are required.
type PIIPatternConfig struct {
	// Type is the PII category label (e.g. "PASSPORT_NO"). It must be non-empty.
	Type string `yaml:"type"`
	// Regexp is a valid Go regular expression matching a single PII value.
	Regexp string `yaml:"regexp"`
}

// PIIGazetteerTermConfig defines a set of inline operator-supplied terms for a
// single PII type to detect via the gazetteer detector.
type PIIGazetteerTermConfig struct {
	// Type is the PII category label to assign to matched terms (e.g. "ORG").
	// Must not be empty.
	Type string `yaml:"type"`
	// Values is the list of exact terms to match.
	Values []string `yaml:"values"`
}

// PIIGazetteerOptionsConfig controls matching behaviour of the gazetteer detector.
type PIIGazetteerOptionsConfig struct {
	// CaseInsensitive folds case before matching. Defaults to true.
	// A pointer is used so that an explicit "false" in YAML can be
	// distinguished from the zero value (allowing the default of true to
	// be applied by setDefaults).
	CaseInsensitive *bool `yaml:"case_insensitive"`
}

// PIIGazetteerConfig controls the gazetteer-based PII detector. It is an
// opt-in subsystem within pii: the outer pii.enabled flag must also be true
// for any detection to occur.
type PIIGazetteerConfig struct {
	// Enabled activates the gazetteer detector. Defaults to false (opt-in).
	// A pointer is used so that an explicit "enabled: false" can be
	// distinguished from the zero value after unmarshalling.
	Enabled *bool `yaml:"enabled"`
	// Packs is the list of embedded pack names to load (e.g. "company-forms",
	// "de-cities"). Unknown names cause a startup error.
	Packs []string `yaml:"packs"`
	// Dirs is a list of directory paths. Every *.txt file in each directory
	// is loaded as a gazetteer file (same format as embedded packs). A
	// nonexistent directory causes a startup error.
	Dirs []string `yaml:"dirs"`
	// Terms is a list of inline operator-supplied term sets. Empty type causes
	// a startup error.
	Terms []PIIGazetteerTermConfig `yaml:"terms"`
	// Options controls matching behaviour.
	Options PIIGazetteerOptionsConfig `yaml:"options"`
}

// IsEnabled returns true only when the gazetteer detector has been explicitly
// enabled. A nil pointer (field absent from YAML) returns false.
func (g PIIGazetteerConfig) IsEnabled() bool {
	if g.Enabled == nil {
		return false
	}
	return *g.Enabled
}

// IsCaseInsensitive returns whether case-insensitive matching is active.
// A nil pointer (field absent from YAML) returns true (the default).
func (o PIIGazetteerOptionsConfig) IsCaseInsensitive() bool {
	if o.CaseInsensitive == nil {
		return true
	}
	return *o.CaseInsensitive
}

// PIIConfig controls in-memory PII anonymization of outbound LLM requests.
// When enabled, PII detected in message content is replaced with
// deterministic pseudonyms before the request leaves the proxy. The
// pseudonyms are restored in the response so the caller receives the
// original values. No PII is ever persisted or logged.
type PIIConfig struct {
	// Enabled activates PII anonymization. Defaults to false (opt-in).
	// A pointer is used so that an explicit "enabled: false" in YAML can
	// be distinguished from the zero value after unmarshalling.
	Enabled *bool `yaml:"enabled"`
	// Action is the anonymization strategy. Currently only "pseudonymize"
	// is supported. Defaults to "pseudonymize".
	Action string `yaml:"action"`
	// Patterns is a list of additional custom PII patterns that are
	// applied in addition to the built-in defaults. An empty list means
	// only the built-in patterns are used.
	Patterns []PIIPatternConfig `yaml:"patterns"`
	// Gazetteer controls the optional gazetteer-based PII detector. Disabled
	// by default; set gazetteer.enabled: true to activate.
	Gazetteer PIIGazetteerConfig `yaml:"gazetteer"`
}

// IsEnabled returns true only when PII anonymization has been explicitly
// enabled. A nil pointer (field absent from YAML) returns false so that
// PII anonymization is opt-in rather than on by default.
func (p PIIConfig) IsEnabled() bool {
	if p.Enabled == nil {
		return false
	}
	return *p.Enabled
}

// SettingsConfig holds application-level settings.
type SettingsConfig struct {
	AdminKey      string `yaml:"admin_key" json:"-"`
	EncryptionKey string `yaml:"encryption_key" json:"-"`
	Bootstrap      BootstrapConfig      `yaml:"bootstrap"`
	Wallet         WalletConfig         `yaml:"wallet"`
	Usage          UsageConfig          `yaml:"usage"`
	Audit          AuditConfig          `yaml:"audit"`
	OTel           OTelConfig           `yaml:"otel"`
	TokenCounting  TokenCountingConfig  `yaml:"token_counting"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
	HealthCheck    HealthCheckConfig    `yaml:"health_check"`
	Retention      RetentionConfig      `yaml:"retention"`
	// FallbackMaxDepth limits how deep the model fallback chain can recurse
	// per request. Default 3, valid range [1, 10]. Ignored when no model has
	// fallback configured or when the license lacks FeatureFallbackChains.
	FallbackMaxDepth int `yaml:"fallback_max_depth" json:"fallback_max_depth"`
	// SoftLimitThreshold uses *float64 so that an explicit 0.0 can be
	// distinguished from the zero value after unmarshalling. Use
	// GetSoftLimitThreshold to read the value.
	SoftLimitThreshold *float64 `yaml:"soft_limit_threshold"`
	// PII holds settings for in-memory PII anonymization. Disabled by default.
	PII PIIConfig `yaml:"pii"`
}

// LogValue implements slog.LogValuer to prevent secrets from appearing in logs.
func (s SettingsConfig) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("admin_key", "[REDACTED]"),
		slog.String("encryption_key", "[REDACTED]"),
	)
}

// GetSoftLimitThreshold returns the configured threshold, defaulting to 0.9 if not set.
func (s SettingsConfig) GetSoftLimitThreshold() float64 {
	if s.SoftLimitThreshold == nil {
		return 0.9
	}
	return *s.SoftLimitThreshold
}

// WalletConfig controls prepaid-wallet billing on the proxy hot path.
type WalletConfig struct {
	// EnforceBalance rejects proxy requests when the user's wallet balance is
	// zero or negative (HTTP 402). Defaults to false so early deployments can
	// issue API keys and route traffic before top-up flows are live.
	EnforceBalance *bool `yaml:"enforce_balance"`
}

// ShouldEnforceBalance returns false when enforce_balance is unset or false.
func (w WalletConfig) ShouldEnforceBalance() bool {
	if w.EnforceBalance == nil {
		return false
	}
	return *w.EnforceBalance
}

// UsageConfig holds settings for the async usage logging subsystem.
type UsageConfig struct {
	BufferSize    int           `yaml:"buffer_size"`
	FlushInterval time.Duration `yaml:"flush_interval"`
	// DropOnFull defaults to true. A *bool is used so that an explicit false
	// can be distinguished from the zero value after unmarshalling.
	DropOnFull *bool `yaml:"drop_on_full"`
}

// ShouldDropOnFull returns true when the field is nil (not set) or explicitly true.
func (u UsageConfig) ShouldDropOnFull() bool {
	if u.DropOnFull == nil {
		return true
	}
	return *u.DropOnFull
}

// TokenCountingConfig holds settings for the token counting pre-check subsystem.
type TokenCountingConfig struct {
	// Enabled defaults to true. A *bool is used so that an explicit false
	// can be distinguished from the zero value after unmarshalling.
	Enabled *bool `yaml:"enabled"`
}

// IsEnabled returns true when the field is nil (not set) or explicitly true.
func (t TokenCountingConfig) IsEnabled() bool {
	if t.Enabled == nil {
		return true
	}
	return *t.Enabled
}

// Load reads the configuration file at path, applies environment variable
// interpolation, unmarshals the YAML, applies defaults, and validates the result.
// If path is empty, Load calls findConfigFile to locate the file automatically.
// The second return value is true when no config file was found and defaults
// were used; callers should log this after their logger is initialised.
func Load(path string) (*Config, bool, error) {
	if path == "" {
		var err error
		path, err = findConfigFile()
		if err != nil {
			cfg, defErr := loadDefaults()
			if defErr != nil {
				return nil, false, defErr
			}
			return cfg, true, nil
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("config: read file %q: %w", path, err)
	}

	raw = interpolateEnv(raw)

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, false, fmt.Errorf("config: unmarshal yaml: %w", err)
	}

	cfg.setDefaults()

	if err := cfg.validate(); err != nil {
		return nil, false, fmt.Errorf("config: %w", err)
	}

	return &cfg, false, nil
}

// findConfigFile returns the path to the configuration file by checking, in order:
//  1. The VOIDLLM_CONFIG environment variable
//  2. ./voidllm.yaml in the current working directory
//  3. /etc/voidllm/voidllm.yaml
func findConfigFile() (string, error) {
	if v := os.Getenv("VOIDLLM_CONFIG"); v != "" {
		return v, nil
	}

	candidates := []string{
		"./voidllm.yaml",
		"/etc/voidllm/voidllm.yaml",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("no config file found; set VOIDLLM_CONFIG or place voidllm.yaml in the current directory")
}

// loadDefaults returns a Config populated entirely from environment
// variables and built-in defaults. It is used when no configuration
// file is found.
func loadDefaults() (*Config, error) {
	var cfg Config
	cfg.Settings.AdminKey = os.Getenv("VOIDLLM_ADMIN_KEY")
	cfg.Settings.EncryptionKey = os.Getenv("VOIDLLM_ENCRYPTION_KEY")
	cfg.Database.DSN = os.Getenv("VOIDLLM_DATABASE_DSN")
	cfg.Database.Driver = os.Getenv("VOIDLLM_DATABASE_DRIVER")
	cfg.setDefaults()
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return &cfg, nil
}

// setDefaults populates zero-value fields with their documented defaults.
func (c *Config) setDefaults() {
	// Server proxy
	if c.Server.Proxy.Port == 0 {
		c.Server.Proxy.Port = 8080
	}
	if c.Server.Proxy.ReadTimeout == 0 {
		c.Server.Proxy.ReadTimeout = 30 * time.Second
	}
	if c.Server.Proxy.WriteTimeout == 0 {
		c.Server.Proxy.WriteTimeout = 120 * time.Second
	}
	if c.Server.Proxy.IdleTimeout == 0 {
		c.Server.Proxy.IdleTimeout = 60 * time.Second
	}
	if c.Server.Proxy.MaxRequestBody <= 0 {
		c.Server.Proxy.MaxRequestBody = 20 * 1024 * 1024 // 20 MB
	}
	if c.Server.Proxy.MaxResponseBody <= 0 {
		c.Server.Proxy.MaxResponseBody = 50 * 1024 * 1024 // 50 MB
	}
	if c.Server.Proxy.MaxStreamDuration <= 0 {
		c.Server.Proxy.MaxStreamDuration = 5 * time.Minute
	}
	if c.Server.Proxy.DrainTimeout <= 0 {
		c.Server.Proxy.DrainTimeout = 25 * time.Second
	}

	// Database
	if c.Database.Driver == "" {
		c.Database.Driver = "sqlite"
	}
	if c.Database.DSN == "" && c.Database.Driver == "sqlite" {
		c.Database.DSN = "voidllm.db"
	}
	if c.Database.Driver == "postgres" {
		if c.Database.MaxOpenConns == 0 {
			c.Database.MaxOpenConns = 25
		}
		if c.Database.MaxIdleConns == 0 {
			c.Database.MaxIdleConns = 5
		}
		if c.Database.ConnMaxLifetime == 0 {
			c.Database.ConnMaxLifetime = 5 * time.Minute
		}
	}

	// Cache
	if c.Cache.KeyTTL == 0 {
		c.Cache.KeyTTL = 30 * time.Second
	}
	if c.Cache.ModelTTL == 0 {
		c.Cache.ModelTTL = 60 * time.Second
	}
	if c.Cache.AliasTTL == 0 {
		c.Cache.AliasTTL = 60 * time.Second
	}

	// Redis
	if c.Redis.KeyPrefix == "" {
		c.Redis.KeyPrefix = "voidllm:"
	}

	// Settings usage
	if c.Settings.Usage.BufferSize == 0 {
		c.Settings.Usage.BufferSize = 1000
	}
	if c.Settings.Usage.FlushInterval == 0 {
		c.Settings.Usage.FlushInterval = 5 * time.Second
	}

	// Settings audit
	if c.Settings.Audit.BufferSize == 0 {
		c.Settings.Audit.BufferSize = 500
	}
	if c.Settings.Audit.FlushInterval == 0 {
		c.Settings.Audit.FlushInterval = 5 * time.Second
	}

	// Settings retention
	if c.Settings.Retention.Interval <= 0 {
		c.Settings.Retention.Interval = 24 * time.Hour
	}

	// Settings fallback: promote strictly negative to 3; 0 is a valid explicit
	// "disabled" value and must not be overwritten to the default.
	if c.Settings.FallbackMaxDepth < 0 {
		c.Settings.FallbackMaxDepth = 3
	}

	// Bootstrap
	if c.Settings.Bootstrap.AdminEmail == "" {
		c.Settings.Bootstrap.AdminEmail = "admin@voidllm.local"
	}

	// OTel
	if c.Settings.OTel.Endpoint == "" {
		c.Settings.OTel.Endpoint = "localhost:4317"
	}
	if c.Settings.OTel.SampleRate == nil {
		v := 1.0
		c.Settings.OTel.SampleRate = &v
	}

	// Circuit breaker
	if c.Settings.CircuitBreaker.Threshold == 0 {
		c.Settings.CircuitBreaker.Threshold = 5
	}
	if c.Settings.CircuitBreaker.Timeout == 0 {
		c.Settings.CircuitBreaker.Timeout = 30 * time.Second
	}
	if c.Settings.CircuitBreaker.HalfOpenMax == 0 {
		c.Settings.CircuitBreaker.HalfOpenMax = 1
	}


	// Health check — only set interval defaults when the probe is explicitly
	// enabled; never auto-enable a probe that the user has not opted into.
	if c.Settings.HealthCheck.Health.Enabled && c.Settings.HealthCheck.Health.Interval == 0 {
		c.Settings.HealthCheck.Health.Interval = 30 * time.Second
	}
	if c.Settings.HealthCheck.Models.Enabled && c.Settings.HealthCheck.Models.Interval == 0 {
		c.Settings.HealthCheck.Models.Interval = 60 * time.Second
	}
	if c.Settings.HealthCheck.Functional.Enabled && c.Settings.HealthCheck.Functional.Interval == 0 {
		c.Settings.HealthCheck.Functional.Interval = 5 * time.Minute
	}

	// Enforce minimum polling intervals to prevent accidental DoS of upstreams.
	if c.Settings.HealthCheck.Health.Enabled && c.Settings.HealthCheck.Health.Interval < 10*time.Second {
		c.Settings.HealthCheck.Health.Interval = 10 * time.Second
	}
	if c.Settings.HealthCheck.Models.Enabled && c.Settings.HealthCheck.Models.Interval < 10*time.Second {
		c.Settings.HealthCheck.Models.Interval = 10 * time.Second
	}
	if c.Settings.HealthCheck.Functional.Enabled && c.Settings.HealthCheck.Functional.Interval < 60*time.Second {
		c.Settings.HealthCheck.Functional.Interval = 60 * time.Second
	}

	// PII anonymization — disabled by default (opt-in).
	if c.Settings.PII.Enabled == nil {
		disabled := false
		c.Settings.PII.Enabled = &disabled
	}
	if c.Settings.PII.Action == "" {
		c.Settings.PII.Action = "pseudonymize"
	}

	// PII gazetteer — disabled by default (opt-in).
	if c.Settings.PII.Gazetteer.Enabled == nil {
		disabled := false
		c.Settings.PII.Gazetteer.Enabled = &disabled
	}
	// options.case_insensitive defaults to true (opt-out).
	if c.Settings.PII.Gazetteer.Options.CaseInsensitive == nil {
		v := true
		c.Settings.PII.Gazetteer.Options.CaseInsensitive = &v
	}

	// Logging
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
}

// deriveSlug converts a display name to a URL-safe slug. It lowercases the
// input, replaces spaces with hyphens, strips any character that is not
// a-z, 0-9, or a hyphen, and trims leading and trailing hyphens. If the
// result is empty, "default" is returned.
func deriveSlug(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	var buf strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			buf.WriteRune(c)
		}
	}
	slug := strings.Trim(buf.String(), "-")
	if slug == "" {
		return "default"
	}
	return slug
}
