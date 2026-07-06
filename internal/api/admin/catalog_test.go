package admin_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/voidmind-io/voidllm/internal/db"
)

func mustCreatePricedCatalogModel(
	t *testing.T,
	database *db.DB,
	name string,
	public bool,
	sellIn float64,
) *db.Model {
	t.Helper()
	m := mustCreateModelForDeployment(t, database, name)
	sellInCopy := sellIn
	publicFlag := public
	updated, err := database.UpdateModel(context.Background(), m.ID, db.UpdateModelParams{
		IsPublic:       &publicFlag,
		SellInputPer1M: &sellInCopy,
	})
	if err != nil {
		t.Fatalf("price model %q: %v", name, err)
	}
	return updated
}

func decodeCatalog(t *testing.T, resp *http.Response) []map[string]any {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var payload struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode catalog: %v body=%s", err, body)
	}
	return payload.Data
}

func catalogNames(items []map[string]any) []string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i], _ = item["name"].(string)
	}
	return names
}

func TestPublicCatalog_ScopeLandingFiltersIsPublic(t *testing.T) {
	app, database, _ := setupTestApp(t, "file:TestPublicCatalog_ScopeLanding?mode=memory&cache=private")
	_ = mustCreatePricedCatalogModel(t, database, "public-chat", true, 1000)
	_ = mustCreatePricedCatalogModel(t, database, "member-only-chat", false, 2000)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/catalog?scope=landing", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d want 200", resp.StatusCode)
	}
	items := decodeCatalog(t, resp)
	if len(items) != 1 {
		t.Fatalf("landing items = %d want 1: %v", len(items), catalogNames(items))
	}
	if items[0]["name"] != "public-chat" {
		t.Fatalf("name = %v want public-chat", items[0]["name"])
	}
}

func TestPublicCatalog_ScopeMemberReturnsAllPriced(t *testing.T) {
	app, database, _ := setupTestApp(t, "file:TestPublicCatalog_ScopeMember?mode=memory&cache=private")
	_ = mustCreatePricedCatalogModel(t, database, "public-chat", true, 1000)
	_ = mustCreatePricedCatalogModel(t, database, "member-only-chat", false, 2000)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/catalog?scope=member", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d want 200", resp.StatusCode)
	}
	items := decodeCatalog(t, resp)
	if len(items) != 2 {
		t.Fatalf("member items = %d want 2: %v", len(items), catalogNames(items))
	}
}

func TestPublicCatalog_IncludesMinPriceAndCapabilities(t *testing.T) {
	app, database, _ := setupTestApp(t, "file:TestPublicCatalog_Fields?mode=memory&cache=private")
	m := mustCreateModelForDeployment(t, database, "capable-chat")
	sellIn := 5000.0
	sellMin := 250.0
	tools := true
	vision := true
	billMin := true
	ctxTokens := 200000
	_, err := database.UpdateModel(context.Background(), m.ID, db.UpdateModelParams{
		SellInputPer1M:     &sellIn,
		BillMinPerRequest:  &billMin,
		SellMinPerRequest:  &sellMin,
		SupportsTools:      &tools,
		SupportsVision:     &vision,
		MaxContextTokens:   &ctxTokens,
	})
	if err != nil {
		t.Fatalf("update model: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/catalog?scope=member", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d want 200", resp.StatusCode)
	}
	items := decodeCatalog(t, resp)
	if len(items) != 1 {
		t.Fatalf("items = %d want 1", len(items))
	}
	item := items[0]
	if item["bill_min_per_request"] != true {
		t.Fatalf("bill_min_per_request = %v", item["bill_min_per_request"])
	}
	if item["sell_min_per_request"] != sellMin {
		t.Fatalf("sell_min_per_request = %v want %v", item["sell_min_per_request"], sellMin)
	}
	if item["supports_tools"] != true || item["supports_vision"] != true {
		t.Fatalf("capabilities = tools %v vision %v", item["supports_tools"], item["supports_vision"])
	}
	if item["max_context_tokens"] != float64(ctxTokens) {
		t.Fatalf("max_context_tokens = %v want %d", item["max_context_tokens"], ctxTokens)
	}
}