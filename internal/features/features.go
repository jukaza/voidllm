package features

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"

	"github.com/voidmind-io/voidllm/internal/config"
)

// SettingsStore reads and writes feature settings.
type SettingsStore interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	SetSettingIfNotExists(ctx context.Context, key, value string) error
}

// WalletConfig controls prepaid marketplace billing behavior.
type WalletConfig struct {
	EnforceBalance    bool    `json:"enforce_balance"`
	InitialBalanceVND float64 `json:"initial_balance_vnd"`
}

// ModulesConfig toggles member-facing UI modules.
type ModulesConfig struct {
	PublicCatalog bool `json:"public_catalog"`
	Playground    bool `json:"playground"`
}

// Config is the full features configuration.
type Config struct {
	Wallet  WalletConfig  `json:"wallet"`
	Modules ModulesConfig `json:"modules"`
}

// PublicConfig is returned by GET /public/features.
type PublicConfig struct {
	Wallet  WalletConfig  `json:"wallet"`
	Modules ModulesConfig `json:"modules"`
}

// UpdateInput is the admin payload for PUT /admin/settings/features.
type UpdateInput struct {
	Wallet  *WalletConfig  `json:"wallet"`
	Modules *ModulesConfig `json:"modules"`
}

// Runtime holds the in-process features config for hot-reload without restart.
type Runtime struct {
	mu  sync.RWMutex
	cfg Config
}

// NewRuntime constructs a runtime seeded with cfg.
func NewRuntime(cfg Config) *Runtime {
	return &Runtime{cfg: cfg}
}

// Get returns a copy of the current config.
func (r *Runtime) Get() Config {
	if r == nil {
		return DefaultConfig()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cfg
}

// Set replaces the in-process config.
func (r *Runtime) Set(cfg Config) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.cfg = cfg
	r.mu.Unlock()
}

// PublicView strips admin-only fields for unauthenticated clients.
func PublicView(cfg Config) PublicConfig {
	return PublicConfig{
		Wallet:  cfg.Wallet,
		Modules: cfg.Modules,
	}
}

// EnsureDefaults seeds first-run feature settings without overwriting operator edits.
func EnsureDefaults(ctx context.Context, store SettingsStore) error {
	def := DefaultConfig()
	defaults := map[string]string{
		KeyWalletEnforceBalance: strconv.FormatBool(def.Wallet.EnforceBalance),
		KeyWalletInitialBalance: strconv.FormatFloat(def.Wallet.InitialBalanceVND, 'f', 0, 64),
		KeyModulesPublicCatalog: strconv.FormatBool(def.Modules.PublicCatalog),
		KeyModulesPlayground:    strconv.FormatBool(def.Modules.Playground),
	}
	for key, value := range defaults {
		if err := store.SetSettingIfNotExists(ctx, key, value); err != nil {
			return fmt.Errorf("seed %s: %w", key, err)
		}
	}
	return nil
}

// SeedYAMLDefaults writes YAML wallet settings only when the DB key is absent.
func SeedYAMLDefaults(ctx context.Context, store SettingsStore, yamlWallet config.WalletConfig) error {
	if yamlWallet.EnforceBalance != nil {
		if err := store.SetSettingIfNotExists(ctx, KeyWalletEnforceBalance, strconv.FormatBool(*yamlWallet.EnforceBalance)); err != nil {
			return fmt.Errorf("seed yaml enforce_balance: %w", err)
		}
	}
	return EnsureDefaults(ctx, store)
}

// Load reads the current features configuration, seeding defaults when missing.
func Load(ctx context.Context, store SettingsStore) (Config, error) {
	if err := EnsureDefaults(ctx, store); err != nil {
		return Config{}, err
	}
	return loadFromStore(ctx, store)
}

// LoadWithYAMLSeed seeds YAML defaults then loads from the store.
func LoadWithYAMLSeed(ctx context.Context, store SettingsStore, yamlWallet config.WalletConfig) (Config, error) {
	if err := SeedYAMLDefaults(ctx, store, yamlWallet); err != nil {
		return Config{}, err
	}
	return loadFromStore(ctx, store)
}

func loadFromStore(ctx context.Context, store SettingsStore) (Config, error) {
	def := DefaultConfig()

	enforceBalance, err := loadBool(ctx, store, KeyWalletEnforceBalance, def.Wallet.EnforceBalance)
	if err != nil {
		return Config{}, err
	}
	initialBalance, err := loadFloat(ctx, store, KeyWalletInitialBalance, def.Wallet.InitialBalanceVND)
	if err != nil {
		return Config{}, err
	}
	publicCatalog, err := loadBool(ctx, store, KeyModulesPublicCatalog, def.Modules.PublicCatalog)
	if err != nil {
		return Config{}, err
	}
	playground, err := loadBool(ctx, store, KeyModulesPlayground, def.Modules.Playground)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Wallet: WalletConfig{
			EnforceBalance:    enforceBalance,
			InitialBalanceVND: initialBalance,
		},
		Modules: ModulesConfig{
			PublicCatalog: publicCatalog,
			Playground:    playground,
		},
	}, nil
}

// Update validates and persists feature settings.
func Update(ctx context.Context, store SettingsStore, input UpdateInput) (Config, error) {
	current, err := Load(ctx, store)
	if err != nil {
		return Config{}, err
	}

	if input.Wallet != nil {
		if input.Wallet.InitialBalanceVND < 0 || input.Wallet.InitialBalanceVND > MaxInitialBalanceVND {
			return Config{}, fmt.Errorf("initial_balance_vnd must be between 0 and %d", MaxInitialBalanceVND)
		}
		current.Wallet = *input.Wallet
	}
	if input.Modules != nil {
		current.Modules = *input.Modules
	}

	if err := store.SetSetting(ctx, KeyWalletEnforceBalance, strconv.FormatBool(current.Wallet.EnforceBalance)); err != nil {
		return Config{}, err
	}
	if err := store.SetSetting(ctx, KeyWalletInitialBalance, strconv.FormatFloat(current.Wallet.InitialBalanceVND, 'f', 0, 64)); err != nil {
		return Config{}, err
	}
	if err := store.SetSetting(ctx, KeyModulesPublicCatalog, strconv.FormatBool(current.Modules.PublicCatalog)); err != nil {
		return Config{}, err
	}
	if err := store.SetSetting(ctx, KeyModulesPlayground, strconv.FormatBool(current.Modules.Playground)); err != nil {
		return Config{}, err
	}

	return current, nil
}

func loadBool(ctx context.Context, store SettingsStore, key string, fallback bool) (bool, error) {
	raw, err := store.GetSetting(ctx, key)
	if err != nil {
		return false, err
	}
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback, nil
	}
	return parsed, nil
}

func loadFloat(ctx context.Context, store SettingsStore, key string, fallback float64) (float64, error) {
	raw, err := store.GetSetting(ctx, key)
	if err != nil {
		return 0, err
	}
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return fallback, nil
	}
	if parsed < 0 {
		return 0, nil
	}
	return parsed, nil
}