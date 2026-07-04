package admin_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
)

func TestProviderConnections_CRUD(t *testing.T) {
	app, database, keyCache := setupTestAppWithEncKey(t, "file:provider-conn-api?mode=memory&cache=private")
	adminKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	prov, err := database.CreateProvider(context.Background(), db.CreateProviderParams{Name: "OpenAI Test", Status: "active"})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/providers/%s/connections", prov.ID), bodyJSON(t, map[string]any{
		"name":    "primary",
		"api_key": "sk-test-key",
	}))
	createReq.Header.Set("Authorization", "Bearer "+adminKey)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", createResp.StatusCode)
	}
	var created map[string]any
	decodeBody(t, createResp.Body, &created)
	connID, _ := created["id"].(string)
	if connID == "" || created["has_api_key"] != true {
		t.Fatalf("create body: %+v", created)
	}

	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/providers/%s/connections", prov.ID), nil)
	listReq.Header.Set("Authorization", "Bearer "+adminKey)
	listResp, err := app.Test(listReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("list request: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", listResp.StatusCode)
	}
	var listWrap struct {
		Data []map[string]any `json:"data"`
	}
	decodeBody(t, listResp.Body, &listWrap)
	if len(listWrap.Data) != 1 {
		t.Fatalf("list len = %d", len(listWrap.Data))
	}

	unlockReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/providers/%s/connections/%s/unlock", prov.ID, connID), nil)
	unlockReq.Header.Set("Authorization", "Bearer "+adminKey)
	unlockResp, err := app.Test(unlockReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("unlock request: %v", err)
	}
	if unlockResp.StatusCode != http.StatusOK {
		t.Fatalf("unlock status = %d", unlockResp.StatusCode)
	}

	reorderReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/providers/%s/connections/reorder", prov.ID), bodyJSON(t, map[string]any{
		"ordered_ids": []string{connID},
	}))
	reorderReq.Header.Set("Authorization", "Bearer "+adminKey)
	reorderReq.Header.Set("Content-Type", "application/json")
	reorderResp, err := app.Test(reorderReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("reorder request: %v", err)
	}
	if reorderResp.StatusCode != http.StatusNoContent {
		t.Fatalf("reorder status = %d", reorderResp.StatusCode)
	}

	patchReq := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/providers/%s/connections/%s", prov.ID, connID), bodyJSON(t, map[string]any{
		"name": "primary-updated",
	}))
	patchReq.Header.Set("Authorization", "Bearer "+adminKey)
	patchReq.Header.Set("Content-Type", "application/json")
	patchResp, err := app.Test(patchReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("patch request: %v", err)
	}
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d", patchResp.StatusCode)
	}

	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/providers/%s/connections/%s", prov.ID, connID), nil)
	delReq.Header.Set("Authorization", "Bearer "+adminKey)
	delResp, err := app.Test(delReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("delete request: %v", err)
	}
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", delResp.StatusCode)
	}
}

func TestProviderConnections_BulkCreate(t *testing.T) {
	app, database, keyCache := setupTestAppWithEncKey(t, "file:provider-bulk?mode=memory&cache=private")
	adminKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	prov, err := database.CreateProvider(context.Background(), db.CreateProviderParams{Name: "Bulk Test", Status: "active"})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/providers/%s/connections", prov.ID), bodyJSON(t, map[string]any{
		"bulk": "key-a|sk-a\nkey-b|sk-b",
	}))
	createReq.Header.Set("Authorization", "Bearer "+adminKey)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("bulk create: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		body, _ := json.Marshal(createResp.Body)
		t.Fatalf("bulk create status = %d body=%s", createResp.StatusCode, string(body))
	}
	var bulkWrap struct {
		Results []map[string]any `json:"results"`
	}
	decodeBody(t, createResp.Body, &bulkWrap)
	if len(bulkWrap.Results) != 2 {
		t.Fatalf("bulk len = %d", len(bulkWrap.Results))
	}
	names := []string{bulkWrap.Results[0]["name"].(string), bulkWrap.Results[1]["name"].(string)}
	if !strings.Contains(names[0], "key") || !strings.Contains(names[1], "key") {
		t.Fatalf("bulk names: %v", names)
	}
}

func TestProviderUpstreamModels_CRUD(t *testing.T) {
	app, database, keyCache := setupTestAppWithEncKey(t, "file:upstream-api?mode=memory&cache=private")
	adminKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	prov, err := database.CreateProvider(context.Background(), db.CreateProviderParams{Name: "Upstream Test", Status: "active"})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	importReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/providers/%s/upstream-models/import", prov.ID), bodyJSON(t, map[string]any{
		"models": []map[string]any{{"upstream_id": "gpt-4o"}},
	}))
	importReq.Header.Set("Authorization", "Bearer "+adminKey)
	importReq.Header.Set("Content-Type", "application/json")
	importResp, err := app.Test(importReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("import request: %v", err)
	}
	if importResp.StatusCode != http.StatusCreated {
		t.Fatalf("import status = %d", importResp.StatusCode)
	}

	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/providers/%s/upstream-models", prov.ID), nil)
	listReq.Header.Set("Authorization", "Bearer "+adminKey)
	listResp, err := app.Test(listReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("list request: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", listResp.StatusCode)
	}
	var listWrap struct {
		Data []map[string]any `json:"data"`
	}
	decodeBody(t, listResp.Body, &listWrap)
	if len(listWrap.Data) != 1 || listWrap.Data[0]["upstream_id"] != "gpt-4o" {
		t.Fatalf("list body: %+v", listWrap.Data)
	}
	modelID, _ := listWrap.Data[0]["id"].(string)

	pickerReq := httptest.NewRequest(http.MethodGet, "/api/v1/upstream-models", nil)
	pickerReq.Header.Set("Authorization", "Bearer "+adminKey)
	pickerResp, err := app.Test(pickerReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("picker request: %v", err)
	}
	if pickerResp.StatusCode != http.StatusOK {
		t.Fatalf("picker status = %d", pickerResp.StatusCode)
	}

	patchReq := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/providers/%s/upstream-models/%s", prov.ID, modelID), bodyJSON(t, map[string]any{
		"display_name": "GPT-4o",
	}))
	patchReq.Header.Set("Authorization", "Bearer "+adminKey)
	patchReq.Header.Set("Content-Type", "application/json")
	patchResp, err := app.Test(patchReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("patch request: %v", err)
	}
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d", patchResp.StatusCode)
	}

	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/providers/%s/upstream-models/%s", prov.ID, modelID), nil)
	delReq.Header.Set("Authorization", "Bearer "+adminKey)
	delResp, err := app.Test(delReq, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("delete request: %v", err)
	}
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", delResp.StatusCode)
	}
}