package keys

import (
	"context"
	"strconv"
)

// Settings keys stored in the settings table.
const (
	KeyMaxPerUser           = "keys.max_per_user"
	KeyAutoCreateOnRegister = "keys.auto_create_on_register"
	KeyDefaultExpiryDays    = "keys.default_expiry_days"
	KeyAllowCustomKey       = "keys.allow_custom_key"
	KeyTrustForwardedIP     = "keys.trust_forwarded_ip"
)

// SettingsStore reads key policy settings.
type SettingsStore interface {
	GetSetting(ctx context.Context, key string) (string, error)
}

// Policy holds admin-configurable API key policy.
type Policy struct {
	MaxPerUser           int  `json:"max_per_user"`
	AutoCreateOnRegister bool `json:"auto_create_on_register"`
	DefaultExpiryDays    int  `json:"default_expiry_days"`
	AllowCustomKey       bool `json:"allow_custom_key"`
	TrustForwardedIP     bool `json:"trust_forwarded_ip"`
}

// DefaultPolicy returns the default key policy.
func DefaultPolicy() Policy {
	return Policy{
		MaxPerUser:           10,
		AutoCreateOnRegister: true,
		DefaultExpiryDays:    0,
		AllowCustomKey:       false,
		TrustForwardedIP:     false,
	}
}

// LoadPolicy reads key policy from the settings table.
func LoadPolicy(ctx context.Context, store SettingsStore) (Policy, error) {
	p := DefaultPolicy()
	if store == nil {
		return p, nil
	}

	if v, err := store.GetSetting(ctx, KeyMaxPerUser); err != nil {
		return p, err
	} else if v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.MaxPerUser = n
		}
	}

	if v, err := store.GetSetting(ctx, KeyAutoCreateOnRegister); err != nil {
		return p, err
	} else if v != "" {
		p.AutoCreateOnRegister = v == "true" || v == "1"
	}

	if v, err := store.GetSetting(ctx, KeyDefaultExpiryDays); err != nil {
		return p, err
	} else if v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			p.DefaultExpiryDays = n
		}
	}

	if v, err := store.GetSetting(ctx, KeyAllowCustomKey); err != nil {
		return p, err
	} else if v != "" {
		p.AllowCustomKey = v == "true" || v == "1"
	}

	if v, err := store.GetSetting(ctx, KeyTrustForwardedIP); err != nil {
		return p, err
	} else if v != "" {
		p.TrustForwardedIP = v == "true" || v == "1"
	}

	return p, nil
}

// SavePolicy persists key policy to the settings table.
func SavePolicy(ctx context.Context, store interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
}, p Policy) error {
	if p.MaxPerUser <= 0 {
		p.MaxPerUser = DefaultPolicy().MaxPerUser
	}
	if err := store.SetSetting(ctx, KeyMaxPerUser, strconv.Itoa(p.MaxPerUser)); err != nil {
		return err
	}
	auto := "false"
	if p.AutoCreateOnRegister {
		auto = "true"
	}
	if err := store.SetSetting(ctx, KeyAutoCreateOnRegister, auto); err != nil {
		return err
	}
	if err := store.SetSetting(ctx, KeyDefaultExpiryDays, strconv.Itoa(p.DefaultExpiryDays)); err != nil {
		return err
	}
	custom := "false"
	if p.AllowCustomKey {
		custom = "true"
	}
	if err := store.SetSetting(ctx, KeyAllowCustomKey, custom); err != nil {
		return err
	}
	trust := "false"
	if p.TrustForwardedIP {
		trust = "true"
	}
	return store.SetSetting(ctx, KeyTrustForwardedIP, trust)
}