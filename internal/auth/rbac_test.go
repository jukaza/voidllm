package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/cache"
	"github.com/voidmind-io/voidllm/pkg/keygen"
)

func TestHasRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		role     string
		required string
		want     bool
	}{
		{name: "system_admin satisfies member", role: RoleSystemAdmin, required: RoleMember, want: true},
		{name: "system_admin satisfies system_admin", role: RoleSystemAdmin, required: RoleSystemAdmin, want: true},
		{name: "member satisfies member", role: RoleMember, required: RoleMember, want: true},
		{name: "member does not satisfy system_admin", role: RoleMember, required: RoleSystemAdmin, want: false},
		{name: "unknown role denied", role: "unknown", required: RoleMember, want: false},
		{name: "empty role denied", role: "", required: RoleMember, want: false},
		{name: "unknown required denied", role: RoleMember, required: "unknown_required", want: false},
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
		{
			name:       "system_admin passes system_admin requirement",
			callerRole: RoleSystemAdmin,
			required:   RoleSystemAdmin,
			wantStatus: fiber.StatusOK,
		},
		{
			name:       "member fails system_admin requirement",
			callerRole: RoleMember,
			required:   RoleSystemAdmin,
			wantStatus: fiber.StatusForbidden,
		},
		{
			name:       "member passes member requirement",
			callerRole: RoleMember,
			required:   RoleMember,
			wantStatus: fiber.StatusOK,
		},
		{
			name:       "system_admin passes member requirement",
			callerRole: RoleSystemAdmin,
			required:   RoleMember,
			wantStatus: fiber.StatusOK,
		},
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
	app.Use(RequireRole(RoleSystemAdmin))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	info := KeyInfo{ID: "x", KeyType: keygen.KeyTypeUser, Role: RoleSystemAdmin}
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