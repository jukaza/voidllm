package admin_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
)

func TestDiscoverProviderModels_UsesConnectionKey(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4o"}]}`))
	}))
	defer upstream.Close()

	app, database, keyCache := setupTestAppWithEncKey(t, "file:discover-conn?mode=memory&cache=private")
	adminKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	prov, err := database.CreateProvider(context.Background(), db.CreateProviderParams{
		Name: "Mock OpenAI", Status: "active", Protocol: "openai", BaseURL: upstream.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/providers/%s/connections", prov.ID), bodyJSON(t, map[string]any{
		"name":    "primary",
		"api_key": "sk-conn-test",
	}))
	createReq.Header.Set("Authorization", "Bearer "+adminKey)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create connection status = %d", createResp.StatusCode)
	}

	discoverReq := httptest.NewRequest(http.MethodPost, "/api/v1/providers/discover-models", bodyJSON(t, map[string]any{
		"provider_id": prov.ID,
	}))
	discoverReq.Header.Set("Authorization", "Bearer "+adminKey)
	discoverReq.Header.Set("Content-Type", "application/json")
	discoverResp, err := app.Test(discoverReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("discover request: %v", err)
	}
	if discoverResp.StatusCode != http.StatusOK {
		t.Fatalf("discover status = %d", discoverResp.StatusCode)
	}

	var out struct {
		Success bool `json:"success"`
		Data    []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeBody(t, discoverResp.Body, &out)
	if !out.Success {
		t.Fatalf("discover success=false")
	}
	if len(out.Data) != 1 || out.Data[0].ID != "gpt-4o" {
		t.Fatalf("discover data = %+v", out.Data)
	}
	if gotAuth != "Bearer sk-conn-test" {
		t.Fatalf("upstream auth = %q, want Bearer sk-conn-test", gotAuth)
	}
}