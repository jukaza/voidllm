package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/cache"
	"github.com/jukaza/tavo/pkg/keygen"
)

func TestHasRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		role     string
		required string
		want     bool
	}{
		{name: "root satisfies member", role: RoleRoot, required: RoleMember, want: true},
		{name: "root satisfies admin", role: RoleRoot, required: RoleAdmin, want: true},
		{name: "root satisfies root", role: RoleRoot, required: RoleRoot, want: true},
		{name: "admin satisfies member", role: RoleAdmin, required: RoleMember, want: true},
		{name: "admin satisfies admin", role: RoleAdmin, required: RoleAdmin, want: true},
		{name: "admin does not satisfy root", role: RoleAdmin, required: RoleRoot, want: false},
		{name: "member satisfies member", role: RoleMember, required: RoleMember, want: true},
		{name: "member does not satisfy admin", role: RoleMember, required: RoleAdmin, want: false},
		{name: "system_admin alias satisfies root", role: RoleSystemAdmin, required: RoleRoot, want: true},
		{name: "unknown role denied", role: "unknown", required: RoleMember, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := HasRole(tc.role, tc.required)
			if got != tc.want {
				t.Errorf("HasRole(%q, %q) = %v, want %v", tc.role, tc.required, got, tc.want)
			}
		})
	}
}

func TestCanManageRole(t *testing.T) {
	t.Parallel()
	if !CanManageRole(RoleRoot, RoleAdmin) {
		t.Error("root should manage admin")
	}
	if !CanManageRole(RoleRoot, RoleMember) {
		t.Error("root should manage member")
	}
	if !CanManageRole(RoleAdmin, RoleMember) {
		t.Error("admin should manage member")
	}
	if CanManageRole(RoleAdmin, RoleRoot) {
		t.Error("admin should not manage root")
	}
	if CanManageRole(RoleMember, RoleMember) {
		t.Error("member should not manage member")
	}
}

func setupRBACApp(t *testing.T, required string) (*fiber.App, *cache.Cache[string, KeyInfo]) {
	t.Helper()
	keyCache := cache.New[string, KeyInfo]()
	app := fiber.New()
	app.Use(Middleware(keyCache, testSecret))
	app.Use(RequireRole(required))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app, keyCache
}

func storeKeyWithRole(t *testing.T, kc *cache.Cache[string, KeyInfo], role string) string {
	t.Helper()
	info := KeyInfo{
		ID:      "rbac-key-" + role,
		KeyType: keygen.KeyTypeUser,
		Role:    role,
	}
	return storeKey(t, kc, info, keygen.KeyTypeUser)
}

func TestRequireRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		callerRole string
		required   string
		wantStatus int
	}{
		{name: "root passes root requirement", callerRole: RoleRoot, required: RoleRoot, wantStatus: fiber.StatusOK},
		{name: "admin fails root requirement", callerRole: RoleAdmin, required: RoleRoot, wantStatus: fiber.StatusForbidden},
		{name: "admin passes admin requirement", callerRole: RoleAdmin, required: RoleAdmin, wantStatus: fiber.StatusOK},
		{name: "member fails admin requirement", callerRole: RoleMember, required: RoleAdmin, wantStatus: fiber.StatusForbidden},
		{name: "root passes admin requirement", callerRole: RoleRoot, required: RoleAdmin, wantStatus: fiber.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app, keyCache := setupRBACApp(t, tc.required)
			raw := storeKeyWithRole(t, keyCache, tc.callerRole)

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer "+raw)

			resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestRequireRoleWithNoAuth(t *testing.T) {
	t.Parallel()

	keyCache := cache.New[string, KeyInfo]()
	app := fiber.New()
	app.Use(RequireRole(RoleRoot))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	info := KeyInfo{ID: "x", KeyType: keygen.KeyTypeUser, Role: RoleRoot}
	_ = storeKey(t, keyCache, info, keygen.KeyTypeUser)

	req := httptest.NewRequest("GET", "/", nil)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusUnauthorized)
	}
}