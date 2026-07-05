package email

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// SettingsStore reads and writes email settings.
type SettingsStore interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	SetSettingIfNotExists(ctx context.Context, key, value string) error
}

// SMTPConfig holds outbound SMTP credentials for transactional email.
type SMTPConfig struct {
	Enabled    bool   `json:"enabled"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	From       string `json:"from"`
	SSLEnabled bool   `json:"ssl_enabled"`
}

// AdminConfig is returned to the settings UI without exposing the raw password.
type AdminConfig struct {
	SMTPConfig
	PasswordConfigured bool `json:"password_configured"`
}

// UpdateInput is the admin payload for PUT /admin/settings/email.
type UpdateInput struct {
	SMTP *SMTPConfig `json:"smtp"`
}

// EnsureDefaults seeds first-run email settings.
func EnsureDefaults(ctx context.Context, store SettingsStore) error {
	defaults := map[string]string{
		KeySMTPEnabled:    "false",
		KeySMTPHost:       DefaultSMTPHost,
		KeySMTPPort:       strconv.Itoa(DefaultSMTPPort),
		KeySMTPUsername:   "",
		KeySMTPPassword:   "",
		KeySMTPFrom:       "",
		KeySMTPSSLEnabled: strconv.FormatBool(DefaultSMTPSSLEnabled),
	}
	for key, value := range defaults {
		if err := store.SetSettingIfNotExists(ctx, key, value); err != nil {
			return fmt.Errorf("seed %s: %w", key, err)
		}
	}
	return nil
}

// Load reads email settings from the store.
func Load(ctx context.Context, store SettingsStore) (SMTPConfig, error) {
	if err := EnsureDefaults(ctx, store); err != nil {
		return SMTPConfig{}, err
	}

	enabled, err := loadBool(ctx, store, KeySMTPEnabled, false)
	if err != nil {
		return SMTPConfig{}, err
	}
	host, err := store.GetSetting(ctx, KeySMTPHost)
	if err != nil {
		return SMTPConfig{}, err
	}
	port, err := loadInt(ctx, store, KeySMTPPort, DefaultSMTPPort)
	if err != nil {
		return SMTPConfig{}, err
	}
	username, err := store.GetSetting(ctx, KeySMTPUsername)
	if err != nil {
		return SMTPConfig{}, err
	}
	password, err := store.GetSetting(ctx, KeySMTPPassword)
	if err != nil {
		return SMTPConfig{}, err
	}
	from, err := store.GetSetting(ctx, KeySMTPFrom)
	if err != nil {
		return SMTPConfig{}, err
	}
	sslEnabled, err := loadBool(ctx, store, KeySMTPSSLEnabled, DefaultSMTPSSLEnabled)
	if err != nil {
		return SMTPConfig{}, err
	}

	host = strings.TrimSpace(host)
	if host == "" {
		host = DefaultSMTPHost
	}
	if port <= 0 || port > 65535 {
		port = DefaultSMTPPort
	}

	return SMTPConfig{
		Enabled:    enabled,
		Host:       host,
		Port:       port,
		Username:   strings.TrimSpace(username),
		Password:   password,
		From:       strings.TrimSpace(from),
		SSLEnabled: sslEnabled,
	}, nil
}

// LoadAdmin returns settings for the admin UI without exposing the raw password.
func LoadAdmin(ctx context.Context, store SettingsStore) (AdminConfig, error) {
	cfg, err := Load(ctx, store)
	if err != nil {
		return AdminConfig{}, err
	}
	return AdminConfig{
		SMTPConfig: SMTPConfig{
			Enabled:    cfg.Enabled,
			Host:       cfg.Host,
			Port:       cfg.Port,
			Username:   cfg.Username,
			From:       cfg.From,
			SSLEnabled: cfg.SSLEnabled,
		},
		PasswordConfigured: strings.TrimSpace(cfg.Password) != "",
	}, nil
}

// IsConfigured reports whether SMTP is ready to send (password may be stored separately).
func (c SMTPConfig) IsConfigured() bool {
	return strings.TrimSpace(c.Host) != "" &&
		c.Port > 0 &&
		strings.TrimSpace(c.Username) != "" &&
		strings.TrimSpace(c.From) != ""
}

// Update applies partial admin changes.
func Update(ctx context.Context, store SettingsStore, input UpdateInput) (SMTPConfig, error) {
	if err := EnsureDefaults(ctx, store); err != nil {
		return SMTPConfig{}, err
	}
	if input.SMTP == nil {
		return Load(ctx, store)
	}

	s := *input.SMTP
	if err := store.SetSetting(ctx, KeySMTPEnabled, strconv.FormatBool(s.Enabled)); err != nil {
		return SMTPConfig{}, err
	}

	host := strings.TrimSpace(s.Host)
	if host == "" {
		host = DefaultSMTPHost
	}
	if err := store.SetSetting(ctx, KeySMTPHost, host); err != nil {
		return SMTPConfig{}, err
	}

	port := s.Port
	if port <= 0 {
		port = DefaultSMTPPort
	}
	if port > 65535 {
		return SMTPConfig{}, fmt.Errorf("smtp port must be between 1 and 65535")
	}
	if err := store.SetSetting(ctx, KeySMTPPort, strconv.Itoa(port)); err != nil {
		return SMTPConfig{}, err
	}

	if err := store.SetSetting(ctx, KeySMTPUsername, strings.TrimSpace(s.Username)); err != nil {
		return SMTPConfig{}, err
	}
	if err := store.SetSetting(ctx, KeySMTPFrom, strings.TrimSpace(s.From)); err != nil {
		return SMTPConfig{}, err
	}
	if err := store.SetSetting(ctx, KeySMTPSSLEnabled, strconv.FormatBool(s.SSLEnabled)); err != nil {
		return SMTPConfig{}, err
	}
	if password := strings.TrimSpace(s.Password); password != "" {
		if err := store.SetSetting(ctx, KeySMTPPassword, password); err != nil {
			return SMTPConfig{}, err
		}
	}

	return Load(ctx, store)
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
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback, nil
	}
	return parsed, nil
}