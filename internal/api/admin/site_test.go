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
	"github.com/voidmind-io/voidllm/internal/site"
)

func TestPublicSiteSeedsDefaults(t *testing.T) {
	app, _, _ := setupTestApp(t, ":memory:")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/site", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}

	var cfg site.Config
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.SystemName != site.DefaultSystemName {
		t.Errorf("system_name = %q, want %q", cfg.SystemName, site.DefaultSystemName)
	}
	if cfg.Logo != site.DefaultLogo {
		t.Errorf("logo = %q, want %q", cfg.Logo, site.DefaultLogo)
	}
	if !cfg.RegisterEnabled {
		t.Error("expected register_enabled true by default")
	}
	if !cfg.UserAgreementEnabled {
		t.Error("expected user_agreement_enabled true by default")
	}
	if !cfg.PrivacyPolicyEnabled {
		t.Error("expected privacy_policy_enabled true by default")
	}
}

func TestAdminSiteUpdate(t *testing.T) {
	app, _, keyCache := setupTestApp(t, ":memory:")
	token := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	body := `{"system_name":"AI Shop VN","footer":"Powered by AI Shop","notice_enabled":true,"announcements":[{"id":"a1","content":"**Bảo trì** tối nay","publish_date":"2026-07-05T10:00:00Z","type":"warning"}]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/site", strings.NewReader(body))
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

	var updated site.Config
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if updated.SystemName != "AI Shop VN" {
		t.Errorf("system_name = %q", updated.SystemName)
	}
	if !updated.NoticeEnabled || len(updated.Announcements) != 1 {
		t.Errorf("announcements = %d enabled=%v", len(updated.Announcements), updated.NoticeEnabled)
	}

	publicReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/site", nil)
	publicResp, err := app.Test(publicReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("public: %v", err)
	}
	defer publicResp.Body.Close()

	var public site.Config
	if err := json.NewDecoder(publicResp.Body).Decode(&public); err != nil {
		t.Fatalf("decode public: %v", err)
	}
	if public.SystemName != "AI Shop VN" {
		t.Errorf("public system_name = %q", public.SystemName)
	}
}

func TestAdminSiteRequiresSystemAdmin(t *testing.T) {
	app, _, keyCache := setupTestApp(t, ":memory:")
	token := addTestKey(t, keyCache, auth.RoleMember)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/site", strings.NewReader(`{"system_name":"X"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d, want 403: %s", resp.StatusCode, raw)
	}
}