package admin_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/features"
)

func TestPublicFeaturesSeedsDefaults(t *testing.T) {
	app, _, _ := setupTestApp(t, ":memory:")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/features", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}

	var cfg features.PublicConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !cfg.Modules.PublicCatalog || !cfg.Modules.Playground {
		t.Errorf("modules defaults: %+v", cfg.Modules)
	}
}

func TestAdminFeaturesUpdate(t *testing.T) {
	app, _, keyCache := setupTestApp(t, ":memory:")
	token := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	body := `{"wallet":{"enforce_balance":true,"initial_balance_vnd":25000},"modules":{"public_catalog":false,"playground":false}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/features", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("update status %d: %s", resp.StatusCode, raw)
	}

	var updated features.Config
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !updated.Wallet.EnforceBalance || updated.Wallet.InitialBalanceVND != 25000 {
		t.Errorf("wallet: %+v", updated.Wallet)
	}
	if updated.Modules.PublicCatalog || updated.Modules.Playground {
		t.Errorf("modules: %+v", updated.Modules)
	}
}

func TestPublicCatalogDisabled(t *testing.T) {
	app, database, keyCache := setupTestApp(t, ":memory:")
	token := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/features", strings.NewReader(
		`{"modules":{"public_catalog":false,"playground":true}}`,
	))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("Authorization", "Bearer "+token)
	putResp, err := app.Test(putReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("put status %d", putResp.StatusCode)
	}

	_ = database

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/catalog", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d want 404: %s", resp.StatusCode, body)
	}
}