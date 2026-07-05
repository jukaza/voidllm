package email

import (
	"context"
	"testing"
)

type memStore struct {
	data map[string]string
}

func (m *memStore) GetSetting(_ context.Context, key string) (string, error) {
	return m.data[key], nil
}

func (m *memStore) SetSetting(_ context.Context, key, value string) error {
	m.data[key] = value
	return nil
}

func (m *memStore) SetSettingIfNotExists(_ context.Context, key, value string) error {
	if _, ok := m.data[key]; !ok {
		m.data[key] = value
	}
	return nil
}

func TestEnsureDefaultsAndLoad(t *testing.T) {
	store := &memStore{data: map[string]string{}}
	ctx := context.Background()

	if err := EnsureDefaults(ctx, store); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(ctx, store)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != DefaultSMTPHost {
		t.Fatalf("host = %q, want %q", cfg.Host, DefaultSMTPHost)
	}
	if cfg.Port != DefaultSMTPPort {
		t.Fatalf("port = %d, want %d", cfg.Port, DefaultSMTPPort)
	}
	if cfg.Enabled {
		t.Fatal("expected disabled by default")
	}
}

func TestLoadAdminStripsPassword(t *testing.T) {
	store := &memStore{data: map[string]string{}}
	ctx := context.Background()
	if err := EnsureDefaults(ctx, store); err != nil {
		t.Fatal(err)
	}
	store.data[KeySMTPPassword] = "secret-app-password"

	admin, err := LoadAdmin(ctx, store)
	if err != nil {
		t.Fatal(err)
	}
	if admin.Password != "" {
		t.Fatal("password should not be returned to admin UI")
	}
	if !admin.PasswordConfigured {
		t.Fatal("expected password_configured true")
	}
}

func TestUpdatePreservesPasswordWhenBlank(t *testing.T) {
	store := &memStore{data: map[string]string{}}
	ctx := context.Background()
	if err := EnsureDefaults(ctx, store); err != nil {
		t.Fatal(err)
	}
	store.data[KeySMTPPassword] = "keep-me"

	_, err := Update(ctx, store, UpdateInput{
		SMTP: &SMTPConfig{
			Username: "user@gmail.com",
			From:     "VoidLLM <user@gmail.com>",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.data[KeySMTPPassword] != "keep-me" {
		t.Fatalf("password = %q, want keep-me", store.data[KeySMTPPassword])
	}
}