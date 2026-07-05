package security

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// SettingsStore reads and writes security settings.
type SettingsStore interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
}

// TurnstileConfig holds Cloudflare Turnstile settings.
type TurnstileConfig struct {
	Enabled              bool   `json:"enabled"`
	SiteKey              string `json:"site_key"`
	SecretKeyConfigured  bool   `json:"secret_key_configured"`
}

// TurnstilePublic is exposed to unauthenticated clients.
type TurnstilePublic struct {
	Enabled bool   `json:"enabled"`
	SiteKey string `json:"site_key,omitempty"`
}

// OAuthProviderConfig is one OAuth provider's admin configuration.
type OAuthProviderConfig struct {
	Enabled                 bool   `json:"enabled"`
	AllowLogin              bool   `json:"allow_login"`
	AllowSignup             bool   `json:"allow_signup"`
	ClientID                string `json:"client_id"`
	ClientSecretConfigured  bool   `json:"client_secret_configured"`
}

// OAuthProviderPublic is the public subset for storefront auth UI.
type OAuthProviderPublic struct {
	Enabled bool `json:"enabled"`
	Login   bool `json:"login"`
	Signup  bool `json:"signup"`
}

// OAuthConfig groups all OAuth providers.
type OAuthConfig struct {
	Google OAuthProviderConfig `json:"google"`
	GitHub OAuthProviderConfig `json:"github"`
}

// OAuthPublic is exposed on the public auth config endpoint.
type OAuthPublic struct {
	Google OAuthProviderPublic `json:"google"`
	GitHub OAuthProviderPublic `json:"github"`
}

// TwoFAConfig holds admin policy for user-managed TOTP 2FA.
type TwoFAConfig struct {
	AllowUserEnable    bool `json:"allow_user_enable"`
	RequireSystemAdmin bool `json:"require_system_admin"`
}

// TwoFAPublic is the public subset for storefront auth UI.
type TwoFAPublic struct {
	Available bool `json:"available"`
}

// SessionPolicyConfig controls login session lifetime and concurrency.
type SessionPolicyConfig struct {
	TTLHours      int  `json:"ttl_hours"`
	AllowMultiple bool `json:"allow_multiple"`
	MaxConcurrent int  `json:"max_concurrent"`
}

// PasswordPolicyConfig controls password requirements for local accounts.
type PasswordPolicyConfig struct {
	MinLength             int  `json:"min_length"`
	AllowOAuthSetPassword bool `json:"allow_oauth_set_password"`
}

// Config is the full admin-facing security configuration (secrets stripped).
type Config struct {
	Turnstile TurnstileConfig      `json:"turnstile"`
	OAuth     OAuthConfig          `json:"oauth"`
	TwoFA     TwoFAConfig          `json:"two_fa"`
	Session   SessionPolicyConfig  `json:"session"`
	Password  PasswordPolicyConfig `json:"password"`
}

// PublicConfig is safe to expose without authentication.
type PublicConfig struct {
	Local           bool            `json:"local"`
	RegisterEnabled bool            `json:"register_enabled"`
	Turnstile       TurnstilePublic `json:"turnstile"`
	OAuth           OAuthPublic     `json:"oauth"`
	TwoFA           TwoFAPublic     `json:"two_fa"`
}

// UpdateInput is the admin PUT payload.
type UpdateInput struct {
	Turnstile *TurnstileUpdate       `json:"turnstile"`
	OAuth     *OAuthUpdate           `json:"oauth"`
	TwoFA     *TwoFAUpdate           `json:"two_fa"`
	Session   *SessionPolicyUpdate   `json:"session"`
	Password  *PasswordPolicyUpdate  `json:"password"`
}

// TwoFAUpdate accepts partial two-factor policy changes.
type TwoFAUpdate struct {
	AllowUserEnable    *bool `json:"allow_user_enable"`
	RequireSystemAdmin *bool `json:"require_system_admin"`
}

// SessionPolicyUpdate accepts partial session policy changes.
type SessionPolicyUpdate struct {
	TTLHours      *int  `json:"ttl_hours"`
	AllowMultiple *bool `json:"allow_multiple"`
	MaxConcurrent *int  `json:"max_concurrent"`
}

// PasswordPolicyUpdate accepts partial password policy changes.
type PasswordPolicyUpdate struct {
	MinLength             *int  `json:"min_length"`
	AllowOAuthSetPassword *bool `json:"allow_oauth_set_password"`
}

const (
	DefaultSessionTTLHours      = 24
	DefaultSessionMaxConcurrent = 5
	DefaultPasswordMinLength    = 8

	minSessionTTLHours      = 1
	maxSessionTTLHours      = 720
	minSessionMaxConcurrent = 1
	maxSessionMaxConcurrent = 50
	minPasswordLength       = 8
	maxPasswordLength       = 72
)

// TurnstileUpdate accepts partial turnstile changes.
type TurnstileUpdate struct {
	Enabled   *bool   `json:"enabled"`
	SiteKey   *string `json:"site_key"`
	SecretKey *string `json:"secret_key"`
}

// OAuthProviderUpdate accepts partial provider changes.
type OAuthProviderUpdate struct {
	Enabled      *bool   `json:"enabled"`
	AllowLogin   *bool   `json:"allow_login"`
	AllowSignup  *bool   `json:"allow_signup"`
	ClientID     *string `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
}

// OAuthUpdate accepts partial OAuth changes.
type OAuthUpdate struct {
	Google *OAuthProviderUpdate `json:"google"`
	GitHub *OAuthProviderUpdate `json:"github"`
}

// Load reads security settings for admin (secrets masked).
func Load(ctx context.Context, store SettingsStore) (Config, error) {
	turnstileSecret, err := store.GetSetting(ctx, KeyTurnstileSecretKey)
	if err != nil {
		return Config{}, err
	}
	googleSecret, err := store.GetSetting(ctx, KeyOAuthGoogleClientSecret)
	if err != nil {
		return Config{}, err
	}
	githubSecret, err := store.GetSetting(ctx, KeyOAuthGitHubClientSecret)
	if err != nil {
		return Config{}, err
	}
	turnstileEnabled, err := loadBool(ctx, store, KeyTurnstileEnabled, false)
	if err != nil {
		return Config{}, err
	}
	turnstileSiteKey, err := store.GetSetting(ctx, KeyTurnstileSiteKey)
	if err != nil {
		return Config{}, err
	}

	google, err := loadOAuthProvider(ctx, store, "google")
	if err != nil {
		return Config{}, err
	}
	github, err := loadOAuthProvider(ctx, store, "github")
	if err != nil {
		return Config{}, err
	}
	twoFA, err := loadTwoFA(ctx, store)
	if err != nil {
		return Config{}, err
	}
	session, err := loadSessionPolicy(ctx, store)
	if err != nil {
		return Config{}, err
	}
	password, err := loadPasswordPolicy(ctx, store)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Turnstile: TurnstileConfig{
			Enabled:             turnstileEnabled,
			SiteKey:             turnstileSiteKey,
			SecretKeyConfigured: strings.TrimSpace(turnstileSecret) != "",
		},
		OAuth: OAuthConfig{
			Google: maskProvider(google, googleSecret),
			GitHub: maskProvider(github, githubSecret),
		},
		TwoFA:    twoFA,
		Session:  session,
		Password: password,
	}, nil
}

// InternalConfig includes secrets for server-side verification (never JSON-encoded).
type InternalConfig struct {
	Config
	TurnstileSecret string
	GoogleSecret    string
	GitHubSecret    string
}

// LoadInternal reads secrets for server-side OAuth/Turnstile verification.
func LoadInternal(ctx context.Context, store SettingsStore) (InternalConfig, error) {
	cfg, err := Load(ctx, store)
	if err != nil {
		return InternalConfig{}, err
	}
	turnstileSecret, err := store.GetSetting(ctx, KeyTurnstileSecretKey)
	if err != nil {
		return InternalConfig{}, err
	}
	googleSecret, err := store.GetSetting(ctx, KeyOAuthGoogleClientSecret)
	if err != nil {
		return InternalConfig{}, err
	}
	githubSecret, err := store.GetSetting(ctx, KeyOAuthGitHubClientSecret)
	if err != nil {
		return InternalConfig{}, err
	}
	return InternalConfig{
		Config:          cfg,
		TurnstileSecret: turnstileSecret,
		GoogleSecret:    googleSecret,
		GitHubSecret:    githubSecret,
	}, nil
}

// PublicFrom builds the public auth config.
func PublicFrom(cfg Config, registerEnabled bool) PublicConfig {
	pub := func(p OAuthProviderConfig) OAuthProviderPublic {
		configured := p.Enabled && strings.TrimSpace(p.ClientID) != ""
		return OAuthProviderPublic{
			Enabled: configured,
			Login:   configured && p.AllowLogin,
			Signup:  configured && p.AllowSignup,
		}
	}
	turnstile := TurnstilePublic{Enabled: cfg.Turnstile.Enabled}
	if cfg.Turnstile.Enabled {
		turnstile.SiteKey = cfg.Turnstile.SiteKey
	}
	return PublicConfig{
		Local:           true,
		RegisterEnabled: registerEnabled,
		Turnstile:       turnstile,
		OAuth: OAuthPublic{
			Google: pub(cfg.OAuth.Google),
			GitHub: pub(cfg.OAuth.GitHub),
		},
		TwoFA: TwoFAPublic{
			Available: cfg.TwoFA.AllowUserEnable,
		},
	}
}

// Update applies admin changes.
func Update(ctx context.Context, store SettingsStore, input UpdateInput) (Config, error) {
	if input.Turnstile != nil {
		if err := applyTurnstile(ctx, store, *input.Turnstile); err != nil {
			return Config{}, err
		}
	}
	if input.OAuth != nil {
		if input.OAuth.Google != nil {
			if err := applyOAuthProvider(ctx, store, "google", *input.OAuth.Google); err != nil {
				return Config{}, err
			}
		}
		if input.OAuth.GitHub != nil {
			if err := applyOAuthProvider(ctx, store, "github", *input.OAuth.GitHub); err != nil {
				return Config{}, err
			}
		}
	}
	if input.TwoFA != nil {
		if err := applyTwoFA(ctx, store, *input.TwoFA); err != nil {
			return Config{}, err
		}
	}
	if input.Session != nil {
		if err := applySessionPolicy(ctx, store, *input.Session); err != nil {
			return Config{}, err
		}
	}
	if input.Password != nil {
		if err := applyPasswordPolicy(ctx, store, *input.Password); err != nil {
			return Config{}, err
		}
	}
	return Load(ctx, store)
}

var (
	ErrTurnstileNotConfigured         = errors.New("turnstile site key and secret are required when enabled")
	ErrSessionTTLOutOfRange           = errors.New("session ttl_hours must be between 1 and 720")
	ErrSessionMaxConcurrentOutOfRange = errors.New("session max_concurrent must be between 1 and 50")
	ErrPasswordMinLengthOutOfRange    = errors.New("password min_length must be between 8 and 72")
)

func applyTurnstile(ctx context.Context, store SettingsStore, in TurnstileUpdate) error {
	if in.Enabled != nil && *in.Enabled {
		siteKey, err := store.GetSetting(ctx, KeyTurnstileSiteKey)
		if err != nil {
			return err
		}
		secret, err := store.GetSetting(ctx, KeyTurnstileSecretKey)
		if err != nil {
			return err
		}
		if in.SiteKey != nil {
			siteKey = strings.TrimSpace(*in.SiteKey)
		}
		if in.SecretKey != nil && strings.TrimSpace(*in.SecretKey) != "" {
			secret = strings.TrimSpace(*in.SecretKey)
		}
		if strings.TrimSpace(siteKey) == "" || strings.TrimSpace(secret) == "" {
			return ErrTurnstileNotConfigured
		}
	}
	if in.Enabled != nil {
		if err := store.SetSetting(ctx, KeyTurnstileEnabled, strconv.FormatBool(*in.Enabled)); err != nil {
			return err
		}
	}
	if in.SiteKey != nil {
		if err := store.SetSetting(ctx, KeyTurnstileSiteKey, strings.TrimSpace(*in.SiteKey)); err != nil {
			return err
		}
	}
	if in.SecretKey != nil && strings.TrimSpace(*in.SecretKey) != "" {
		if err := store.SetSetting(ctx, KeyTurnstileSecretKey, strings.TrimSpace(*in.SecretKey)); err != nil {
			return err
		}
	}
	return nil
}

func applyOAuthProvider(ctx context.Context, store SettingsStore, name string, in OAuthProviderUpdate) error {
	keys := oauthKeys(name)
	if in.Enabled != nil {
		if err := store.SetSetting(ctx, keys.enabled, strconv.FormatBool(*in.Enabled)); err != nil {
			return err
		}
	}
	if in.AllowLogin != nil {
		if err := store.SetSetting(ctx, keys.allowLogin, strconv.FormatBool(*in.AllowLogin)); err != nil {
			return err
		}
	}
	if in.AllowSignup != nil {
		if err := store.SetSetting(ctx, keys.allowSignup, strconv.FormatBool(*in.AllowSignup)); err != nil {
			return err
		}
	}
	if in.ClientID != nil {
		if err := store.SetSetting(ctx, keys.clientID, strings.TrimSpace(*in.ClientID)); err != nil {
			return err
		}
	}
	if in.ClientSecret != nil && strings.TrimSpace(*in.ClientSecret) != "" {
		if err := store.SetSetting(ctx, keys.clientSecret, strings.TrimSpace(*in.ClientSecret)); err != nil {
			return err
		}
	}
	return nil
}

type oauthKeySet struct {
	enabled, allowLogin, allowSignup, clientID, clientSecret string
}

func oauthKeys(name string) oauthKeySet {
	switch name {
	case "google":
		return oauthKeySet{KeyOAuthGoogleEnabled, KeyOAuthGoogleAllowLogin, KeyOAuthGoogleAllowSignup, KeyOAuthGoogleClientID, KeyOAuthGoogleClientSecret}
	case "github":
		return oauthKeySet{KeyOAuthGitHubEnabled, KeyOAuthGitHubAllowLogin, KeyOAuthGitHubAllowSignup, KeyOAuthGitHubClientID, KeyOAuthGitHubClientSecret}
	default:
		return oauthKeySet{}
	}
}

func loadOAuthProvider(ctx context.Context, store SettingsStore, name string) (OAuthProviderConfig, error) {
	keys := oauthKeys(name)
	enabled, err := loadBool(ctx, store, keys.enabled, false)
	if err != nil {
		return OAuthProviderConfig{}, err
	}
	allowLogin, err := loadBool(ctx, store, keys.allowLogin, true)
	if err != nil {
		return OAuthProviderConfig{}, err
	}
	allowSignup, err := loadBool(ctx, store, keys.allowSignup, true)
	if err != nil {
		return OAuthProviderConfig{}, err
	}
	clientID, err := store.GetSetting(ctx, keys.clientID)
	if err != nil {
		return OAuthProviderConfig{}, err
	}
	return OAuthProviderConfig{
		Enabled:     enabled,
		AllowLogin:  allowLogin,
		AllowSignup: allowSignup,
		ClientID:    clientID,
	}, nil
}

func maskProvider(p OAuthProviderConfig, secret string) OAuthProviderConfig {
	p.ClientSecretConfigured = strings.TrimSpace(secret) != ""
	return p
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

func loadInt(ctx context.Context, store SettingsStore, key string, fallback int) (int, error) {
	raw, err := store.GetSetting(ctx, key)
	if err != nil {
		return 0, err
	}
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback, nil
	}
	return parsed, nil
}

func loadTwoFA(ctx context.Context, store SettingsStore) (TwoFAConfig, error) {
	allowUser, err := loadBool(ctx, store, KeyTwoFAAllowUserEnable, false)
	if err != nil {
		return TwoFAConfig{}, err
	}
	requireAdmin, err := loadBool(ctx, store, KeyTwoFARequireSystemAdmin, false)
	if err != nil {
		return TwoFAConfig{}, err
	}
	return TwoFAConfig{
		AllowUserEnable:    allowUser,
		RequireSystemAdmin: requireAdmin,
	}, nil
}

func loadSessionPolicy(ctx context.Context, store SettingsStore) (SessionPolicyConfig, error) {
	ttl, err := loadInt(ctx, store, KeySessionTTLHours, DefaultSessionTTLHours)
	if err != nil {
		return SessionPolicyConfig{}, err
	}
	allowMultiple, err := loadBool(ctx, store, KeySessionAllowMultiple, false)
	if err != nil {
		return SessionPolicyConfig{}, err
	}
	maxConcurrent, err := loadInt(ctx, store, KeySessionMaxConcurrent, DefaultSessionMaxConcurrent)
	if err != nil {
		return SessionPolicyConfig{}, err
	}
	return SessionPolicyConfig{
		TTLHours:      clampInt(ttl, minSessionTTLHours, maxSessionTTLHours),
		AllowMultiple: allowMultiple,
		MaxConcurrent: clampInt(maxConcurrent, minSessionMaxConcurrent, maxSessionMaxConcurrent),
	}, nil
}

func loadPasswordPolicy(ctx context.Context, store SettingsStore) (PasswordPolicyConfig, error) {
	minLength, err := loadInt(ctx, store, KeyPasswordMinLength, DefaultPasswordMinLength)
	if err != nil {
		return PasswordPolicyConfig{}, err
	}
	allowOAuthSet, err := loadBool(ctx, store, KeyPasswordAllowOAuthSetPassword, true)
	if err != nil {
		return PasswordPolicyConfig{}, err
	}
	return PasswordPolicyConfig{
		MinLength:             clampInt(minLength, minPasswordLength, maxPasswordLength),
		AllowOAuthSetPassword: allowOAuthSet,
	}, nil
}

func applyTwoFA(ctx context.Context, store SettingsStore, in TwoFAUpdate) error {
	if in.AllowUserEnable != nil {
		if err := store.SetSetting(ctx, KeyTwoFAAllowUserEnable, strconv.FormatBool(*in.AllowUserEnable)); err != nil {
			return err
		}
	}
	if in.RequireSystemAdmin != nil {
		if err := store.SetSetting(ctx, KeyTwoFARequireSystemAdmin, strconv.FormatBool(*in.RequireSystemAdmin)); err != nil {
			return err
		}
	}
	return nil
}

func applySessionPolicy(ctx context.Context, store SettingsStore, in SessionPolicyUpdate) error {
	if in.TTLHours != nil {
		if *in.TTLHours < minSessionTTLHours || *in.TTLHours > maxSessionTTLHours {
			return ErrSessionTTLOutOfRange
		}
		if err := store.SetSetting(ctx, KeySessionTTLHours, strconv.Itoa(*in.TTLHours)); err != nil {
			return err
		}
	}
	if in.AllowMultiple != nil {
		if err := store.SetSetting(ctx, KeySessionAllowMultiple, strconv.FormatBool(*in.AllowMultiple)); err != nil {
			return err
		}
	}
	if in.MaxConcurrent != nil {
		if *in.MaxConcurrent < minSessionMaxConcurrent || *in.MaxConcurrent > maxSessionMaxConcurrent {
			return ErrSessionMaxConcurrentOutOfRange
		}
		if err := store.SetSetting(ctx, KeySessionMaxConcurrent, strconv.Itoa(*in.MaxConcurrent)); err != nil {
			return err
		}
	}
	return nil
}

func applyPasswordPolicy(ctx context.Context, store SettingsStore, in PasswordPolicyUpdate) error {
	if in.MinLength != nil {
		if *in.MinLength < minPasswordLength || *in.MinLength > maxPasswordLength {
			return ErrPasswordMinLengthOutOfRange
		}
		if err := store.SetSetting(ctx, KeyPasswordMinLength, strconv.Itoa(*in.MinLength)); err != nil {
			return err
		}
	}
	if in.AllowOAuthSetPassword != nil {
		if err := store.SetSetting(ctx, KeyPasswordAllowOAuthSetPassword, strconv.FormatBool(*in.AllowOAuthSetPassword)); err != nil {
			return err
		}
	}
	return nil
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}