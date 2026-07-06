package features_test

import (
	"context"
	"testing"

	"github.com/voidmind-io/voidllm/internal/config"
	"github.com/voidmind-io/voidllm/internal/features"
)

type mapStore struct {
	data map[string]string
}

func (m *mapStore) GetSetting(_ context.Context, key string) (string, error) {
	return m.data[key], nil
}

func (m *mapStore) SetSetting(_ context.Context, key, value string) error {
	m.data[key] = value
	return nil
}

func (m *mapStore) SetSettingIfNotExists(_ context.Context, key, value string) error {
	if _, ok := m.data[key]; ok {
		return nil
	}
	m.data[key] = value
	return nil
}

func TestLoadDefaults(t *testing.T) {
	store := &mapStore{data: map[string]string{}}
	cfg, err := features.Load(context.Background(), store)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Wallet.EnforceBalance {
		t.Error("expected enforce_balance false by default")
	}
	if cfg.Modules.PublicCatalog != true || cfg.Modules.Playground != true {
		t.Errorf("modules defaults: %+v", cfg.Modules)
	}
}

func TestUpdateValidation(t *testing.T) {
	store := &mapStore{data: map[string]string{}}
	_, err := features.Load(context.Background(), store)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	tooHigh := float64(features.MaxInitialBalanceVND + 1)
	_, err = features.Update(context.Background(), store, features.UpdateInput{
		Wallet: &features.WalletConfig{InitialBalanceVND: tooHigh},
	})
	if err == nil {
		t.Fatal("expected validation error for high initial balance")
	}
}

func TestUpdateRoundTrip(t *testing.T) {
	store := &mapStore{data: map[string]string{}}
	if _, err := features.Load(context.Background(), store); err != nil {
		t.Fatalf("seed: %v", err)
	}

	updated, err := features.Update(context.Background(), store, features.UpdateInput{
		Wallet: &features.WalletConfig{
			EnforceBalance:    true,
			InitialBalanceVND: 50000,
		},
		Modules: &features.ModulesConfig{
			PublicCatalog: false,
			Playground:    false,
		},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.Wallet.EnforceBalance || updated.Wallet.InitialBalanceVND != 50000 {
		t.Errorf("wallet: %+v", updated.Wallet)
	}
	if updated.Modules.PublicCatalog || updated.Modules.Playground {
		t.Errorf("modules: %+v", updated.Modules)
	}

	loaded, err := features.Load(context.Background(), store)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != updated {
		t.Errorf("round-trip mismatch: got %+v want %+v", loaded, updated)
	}
}

func TestSeedYAMLDefaults(t *testing.T) {
	store := &mapStore{data: map[string]string{}}
	enforce := true
	_, err := features.LoadWithYAMLSeed(context.Background(), store, config.WalletConfig{
		EnforceBalance: &enforce,
	})
	if err != nil {
		t.Fatalf("LoadWithYAMLSeed: %v", err)
	}
	if store.data[features.KeyWalletEnforceBalance] != "true" {
		t.Errorf("yaml seed = %q", store.data[features.KeyWalletEnforceBalance])
	}
}

func TestRuntime(t *testing.T) {
	rt := features.NewRuntime(features.DefaultConfig())
	rt.Set(features.Config{
		Wallet: features.WalletConfig{EnforceBalance: true},
	})
	if !rt.Get().Wallet.EnforceBalance {
		t.Error("expected enforce true")
	}
}