package admin_test

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/pkg/keygen"
)

func TestKeyModelAccess_RoundTrip(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestAppWithRegistry(t, "file:TestKeyModelAccess_RoundTrip?mode=memory&cache=private")

	user, err := database.CreateUser(context.Background(), db.CreateUserParams{
		Email:       "member@example.com",
		DisplayName: "Member User",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	testKey, err := keygen.Generate(keygen.KeyTypeUser)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	apiKey, err := database.CreateAPIKey(context.Background(), db.CreateAPIKeyParams{
		KeyHash:   keygen.Hash(testKey, testHMACSecret),
		KeyHint:   keygen.Hint(testKey),
		KeyType:   keygen.KeyTypeUser,
		Name:      "Member Key",
		UserID:    &user.ID,
		CreatedBy: user.ID,
	})
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	keyCache.Set(keygen.Hash(testKey, testHMACSecret), auth.KeyInfo{
		ID:      apiKey.ID,
		KeyType: keygen.KeyTypeUser,
		Role:    auth.RoleMember,
		UserID:  user.ID,
		Name:    "Member Key",
	})

	keyID := apiKey.ID

	// 1. Get initial model access (should be empty by default, allowing all)
	reqGet := httptest.NewRequest("GET", "/api/v1/keys/"+keyID+"/model-access", nil)
	reqGet.Header.Set("Authorization", "Bearer "+testKey)

	respGet, err := app.Test(reqGet, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("get initial: %v", err)
	}
	defer respGet.Body.Close()

	if respGet.StatusCode != fiber.StatusOK {
		t.Fatalf("initial get status = %d, want 200", respGet.StatusCode)
	}

	var gotInitial map[string]any
	decodeBody(t, respGet.Body, &gotInitial)
	modelsInit, _ := gotInitial["models"].([]any)
	if len(modelsInit) != 0 {
		t.Errorf("initial models len = %d, want 0", len(modelsInit))
	}

	// 2. Set model access to a specific list of models
	body := map[string]any{"models": []string{"gpt-4o", "claude-3-5-sonnet"}}
	reqPut := httptest.NewRequest("PUT", "/api/v1/keys/"+keyID+"/model-access", bodyJSON(t, body))
	reqPut.Header.Set("Content-Type", "application/json")
	reqPut.Header.Set("Authorization", "Bearer "+testKey)

	respPut, err := app.Test(reqPut, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("set models: %v", err)
	}
	defer respPut.Body.Close()

	if respPut.StatusCode != fiber.StatusOK {
		b, _ := io.ReadAll(respPut.Body)
		t.Fatalf("set status = %d, want 200; body: %s", respPut.StatusCode, b)
	}

	// 3. Get model access again and verify the new list
	reqGet2 := httptest.NewRequest("GET", "/api/v1/keys/"+keyID+"/model-access", nil)
	reqGet2.Header.Set("Authorization", "Bearer "+testKey)

	respGet2, err := app.Test(reqGet2, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	defer respGet2.Body.Close()

	if respGet2.StatusCode != fiber.StatusOK {
		t.Fatalf("updated get status = %d, want 200", respGet2.StatusCode)
	}

	var gotUpdated map[string]any
	decodeBody(t, respGet2.Body, &gotUpdated)
	modelsUpdated, _ := gotUpdated["models"].([]any)
	if len(modelsUpdated) != 2 {
		t.Fatalf("updated models len = %d, want 2", len(modelsUpdated))
	}

	hasGPT4 := false
	hasClaude3 := false
	for _, m := range modelsUpdated {
		if m == "gpt-4o" {
			hasGPT4 = true
		}
		if m == "claude-3-5-sonnet" {
			hasClaude3 = true
		}
	}
	if !hasGPT4 || !hasClaude3 {
		t.Errorf("got models: %v, want both gpt-4o and claude-3-5-sonnet", modelsUpdated)
	}
}
