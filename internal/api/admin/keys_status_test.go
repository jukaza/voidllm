package admin_test

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/keys"
	"github.com/jukaza/tavo/pkg/keygen"
)

func TestCreateAPIKey_RejectsDisabledStatusForMembers(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestCreateAPIKeyStatus?mode=memory&cache=private")
	user := mustCreateUser(t, database, "status@example.com", "Status User")
	testKey := addTestKeyWithUser(t, keyCache, auth.RoleMember, user.ID)

	body := map[string]any{
		"name":   "Disabled Key",
		"status": keys.StatusDisabled,
	}
	req := httptest.NewRequest("POST", "/api/v1/keys", bodyJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 400; body: %s", resp.StatusCode, b)
	}
}

func TestUpdateAPIKey_StatusAndSpendCap(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestUpdateAPIKeyStatus?mode=memory&cache=private")
	user := mustCreateUser(t, database, "spend@example.com", "Spend User")
	testKey := addTestKeyWithUser(t, keyCache, auth.RoleMember, user.ID)

	createBody := map[string]any{
		"name":      "Spend Key",
		"spend_cap": 25.5,
	}
	req := httptest.NewRequest("POST", "/api/v1/keys", bodyJSON(t, createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testKey)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create status = %d, want 201; body: %s", resp.StatusCode, b)
	}

	var created map[string]any
	decodeBody(t, resp.Body, &created)
	keyID := created["id"].(string)

	if got := created["spend_cap"].(float64); got != 25.5 {
		t.Fatalf("create spend_cap = %v, want 25.5", got)
	}

	patchBody := map[string]any{
		"status":     keys.StatusDisabled,
		"spend_used": 10.0,
	}
	reqPatch := httptest.NewRequest("PATCH", "/api/v1/keys/"+keyID, bodyJSON(t, patchBody))
	reqPatch.Header.Set("Content-Type", "application/json")
	reqPatch.Header.Set("Authorization", "Bearer "+testKey)
	respPatch, err := app.Test(reqPatch, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	defer respPatch.Body.Close()
	if respPatch.StatusCode != fiber.StatusOK {
		b, _ := io.ReadAll(respPatch.Body)
		t.Fatalf("patch status = %d, want 200; body: %s", respPatch.StatusCode, b)
	}

	var updated map[string]any
	decodeBody(t, respPatch.Body, &updated)
	if updated["status"] != keys.StatusDisabled {
		t.Fatalf("status = %v, want %q", updated["status"], keys.StatusDisabled)
	}
	if updated["spend_used"].(float64) != 10.0 {
		t.Fatalf("spend_used = %v, want 10", updated["spend_used"])
	}
}

func TestCreateAPIKey_MaxPerUser(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestCreateAPIKeyMax?mode=memory&cache=private")
	user := mustCreateUser(t, database, "maxkeys@example.com", "Max Keys")
	testKey := addTestKeyWithUser(t, keyCache, auth.RoleMember, user.ID)

	ctx := context.Background()
	if err := database.SetSetting(ctx, keys.KeyMaxPerUser, "1"); err != nil {
		t.Fatalf("set max_per_user: %v", err)
	}

	body := map[string]any{"name": "First Key"}
	req := httptest.NewRequest("POST", "/api/v1/keys", bodyJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testKey)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("first create status = %d, want 201", resp.StatusCode)
	}

	req2 := httptest.NewRequest("POST", "/api/v1/keys", bodyJSON(t, map[string]any{"name": "Second Key"}))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+testKey)
	resp2, err := app.Test(req2, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != fiber.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("second create status = %d, want 400; body: %s", resp2.StatusCode, b)
	}
}

func TestCreateAPIKey_ReturnsStatusField(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestCreateAPIKeyStatusField?mode=memory&cache=private")
	user := mustCreateUser(t, database, "active@example.com", "Active User")
	testKey := addTestKeyWithUser(t, keyCache, auth.RoleMember, user.ID)

	body := map[string]any{"name": "Active Key"}
	req := httptest.NewRequest("POST", "/api/v1/keys", bodyJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testKey)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 201; body: %s", resp.StatusCode, b)
	}

	var got map[string]any
	decodeBody(t, resp.Body, &got)
	if got["status"] != keys.StatusActive {
		t.Fatalf("status = %v, want %q", got["status"], keys.StatusActive)
	}
	key, ok := got["key"].(string)
	if !ok || key == "" {
		t.Fatalf("response missing plaintext key")
	}
	if _, err := keygen.ValidatePrefix(key); err != nil {
		t.Fatalf("invalid key prefix: %v", err)
	}
}