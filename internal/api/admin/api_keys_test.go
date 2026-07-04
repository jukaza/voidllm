package admin_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/pkg/keygen"
)

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestCreateAPIKey?mode=memory&cache=private")
	user := mustCreateUser(t, database, "creator@example.com", "Creator")
	testKey := addTestKeyWithUser(t, keyCache, auth.RoleMember, user.ID)

	body := map[string]any{
		"name":     "My User Key",
		"key_type": keygen.KeyTypeUser,
	}
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

	key, ok := got["key"].(string)
	if !ok || key == "" {
		t.Errorf("response missing non-empty 'key' field; got: %v", got)
	}
}

func TestGetAPIKey(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestGetAPIKey?mode=memory&cache=private")
	user := mustCreateUser(t, database, "getter@example.com", "Getter")
	testKey := addTestKeyWithUser(t, keyCache, auth.RoleMember, user.ID)

	body := map[string]any{
		"name":     "Fetch Me",
		"key_type": keygen.KeyTypeUser,
	}
	req := httptest.NewRequest("POST", "/api/v1/keys", bodyJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testKey)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()

	var created map[string]any
	decodeBody(t, resp.Body, &created)
	keyID := created["id"].(string)

	reqGet := httptest.NewRequest("GET", "/api/v1/keys/"+keyID, nil)
	reqGet.Header.Set("Authorization", "Bearer "+testKey)
	respGet, err := app.Test(reqGet, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer respGet.Body.Close()

	if respGet.StatusCode != fiber.StatusOK {
		b, _ := io.ReadAll(respGet.Body)
		t.Fatalf("status = %d, want 200; body: %s", respGet.StatusCode, b)
	}
}

func TestDeleteAPIKey(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestDeleteAPIKey?mode=memory&cache=private")
	user := mustCreateUser(t, database, "deleter@example.com", "Deleter")
	testKey := addTestKeyWithUser(t, keyCache, auth.RoleMember, user.ID)

	body := map[string]any{
		"name":     "Delete Me",
		"key_type": keygen.KeyTypeUser,
	}
	req := httptest.NewRequest("POST", "/api/v1/keys", bodyJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testKey)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()

	var created map[string]any
	decodeBody(t, resp.Body, &created)
	keyID := created["id"].(string)

	reqDel := httptest.NewRequest("DELETE", "/api/v1/keys/"+keyID, nil)
	reqDel.Header.Set("Authorization", "Bearer "+testKey)
	respDel, err := app.Test(reqDel, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer respDel.Body.Close()

	if respDel.StatusCode != fiber.StatusNoContent {
		b, _ := io.ReadAll(respDel.Body)
		t.Fatalf("status = %d, want 204; body: %s", respDel.StatusCode, b)
	}
}
