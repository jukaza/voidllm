package admin_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/pkg/keygen"
)

func TestRevealAPIKey_ReturnsStoredPlaintext(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestRevealAPIKey?mode=memory&cache=private")
	user := mustCreateUser(t, database, "reveal@example.com", "Reveal")
	testKey := addTestKeyWithUser(t, keyCache, auth.RoleMember, user.ID)

	createBody := map[string]any{"name": "Reveal Me", "key_type": keygen.KeyTypeUser}
	req := httptest.NewRequest("POST", "/api/v1/keys", bodyJSON(t, createBody))
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
	plaintext := created["key"].(string)

	reqReveal := httptest.NewRequest("GET", "/api/v1/keys/"+keyID+"/reveal", nil)
	reqReveal.Header.Set("Authorization", "Bearer "+testKey)
	respReveal, err := app.Test(reqReveal, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("reveal: %v", err)
	}
	defer respReveal.Body.Close()

	if respReveal.StatusCode != fiber.StatusOK {
		b, _ := io.ReadAll(respReveal.Body)
		t.Fatalf("status = %d, want 200; body: %s", respReveal.StatusCode, b)
	}

	var got map[string]any
	decodeBody(t, respReveal.Body, &got)
	if got["key"] != plaintext {
		t.Fatalf("reveal key = %v, want %q", got["key"], plaintext)
	}
}