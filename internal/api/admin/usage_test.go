package admin_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
)

func insertUsageEventHTTP(t *testing.T, database *db.DB, id, keyID, modelName string, promptTokens, compTokens, totalTokens int64, createdAt time.Time) {
	t.Helper()
	query := fmt.Sprintf(
		`INSERT INTO usage_events
			(id, key_id, key_type, model_name,
			 prompt_tokens, completion_tokens, total_tokens, status_code, created_at)
		 VALUES
			('%s', '%s', 'user_key', '%s',
			 %d, %d, %d, 200, '%s')`,
		id, keyID, modelName,
		promptTokens, compTokens, totalTokens,
		createdAt.UTC().Format(time.RFC3339),
	)
	if _, err := database.SQL().ExecContext(context.Background(), query); err != nil {
		t.Fatalf("insertUsageEventHTTP id=%q: %v", id, err)
	}
}

func usageURL(from, to, groupBy string) string {
	params := []string{}
	if from != "" {
		params = append(params, "from="+from)
	}
	if to != "" {
		params = append(params, "to="+to)
	}
	if groupBy != "" {
		params = append(params, "group_by="+groupBy)
	}
	u := "/api/v1/usage"
	if len(params) > 0 {
		u += "?" + strings.Join(params, "&")
	}
	return u
}



// ---- GET /api/v1/usage ------------------------------------------------------

func TestGetUsage_ValidRange_ReturnsCorrectTotals(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestGetUsage_ValidRange?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	now := time.Now().UTC()
	from := now.Add(-2 * time.Hour).Format(time.RFC3339)
	to := now.Add(time.Minute).Format(time.RFC3339)

	insertUsageEventHTTP(t, database, "uh-1", "key-1", "gpt-4", 100, 50, 150, now.Add(-90*time.Minute))
	insertUsageEventHTTP(t, database, "uh-2", "key-1", "gpt-4", 200, 80, 280, now.Add(-60*time.Minute))

	req := httptest.NewRequest("GET", usageURL(from, to, ""), nil)
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body: %s", resp.StatusCode, body)
	}

	var envelope struct {
		Data []struct {
			TotalRequests    int64   `json:"total_requests"`
			PromptTokens     int64   `json:"prompt_tokens"`
			CompletionTokens int64   `json:"completion_tokens"`
			TotalTokens      int64   `json:"total_tokens"`
			CostEstimate     float64 `json:"cost_estimate"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(envelope.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(envelope.Data))
	}
	row := envelope.Data[0]
	if row.TotalRequests != 2 {
		t.Errorf("total_requests = %d, want 2", row.TotalRequests)
	}
	if row.PromptTokens != 300 {
		t.Errorf("prompt_tokens = %d, want 300", row.PromptTokens)
	}
	if row.CompletionTokens != 130 {
		t.Errorf("completion_tokens = %d, want 130", row.CompletionTokens)
	}
	if row.TotalTokens != 430 {
		t.Errorf("total_tokens = %d, want 430", row.TotalTokens)
	}
}

func TestGetUsage_FromAfterTo_Returns400(t *testing.T) {
	t.Parallel()

	app, _, keyCache := setupTestApp(t, "file:TestGetUsage_FromAfterTo?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	now := time.Now().UTC()
	from := now.Add(time.Hour).Format(time.RFC3339)
	to := now.Add(-time.Hour).Format(time.RFC3339)

	req := httptest.NewRequest("GET", usageURL(from, to, ""), nil)
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want 400; body: %s", resp.StatusCode, body)
	}
}

func TestGetUsage_InvalidFromFormat_Returns400(t *testing.T) {
	t.Parallel()

	app, _, keyCache := setupTestApp(t, "file:TestGetUsage_InvalidFrom?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	now := time.Now().UTC()
	to := now.Add(time.Minute).Format(time.RFC3339)

	req := httptest.NewRequest("GET", usageURL("not-a-date", to, ""), nil)
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want 400; body: %s", resp.StatusCode, body)
	}
}

func TestGetUsage_InvalidGroupBy_Returns400(t *testing.T) {
	t.Parallel()

	app, _, keyCache := setupTestApp(t, "file:TestGetUsage_InvalidGroupBy?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	now := time.Now().UTC()
	from := now.Add(-time.Hour).Format(time.RFC3339)
	to := now.Add(time.Minute).Format(time.RFC3339)

	req := httptest.NewRequest("GET", usageURL(from, to, "invalid_col"), nil)
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want 400; body: %s", resp.StatusCode, body)
	}
}

func TestGetUsage_RangeExceeds90Days_Returns400(t *testing.T) {
	t.Parallel()

	app, _, keyCache := setupTestApp(t, "file:TestGetUsage_91Days?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	now := time.Now().UTC()
	from := now.Add(-91 * 24 * time.Hour).Format(time.RFC3339)
	to := now.Format(time.RFC3339)

	req := httptest.NewRequest("GET", usageURL(from, to, ""), nil)
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want 400; body: %s", resp.StatusCode, body)
	}
}

func TestGetUsage_NoAuth_Returns401(t *testing.T) {
	t.Parallel()

	app, _, _ := setupTestApp(t, "file:TestGetUsage_NoAuth?mode=memory&cache=private")

	now := time.Now().UTC()
	from := now.Add(-time.Hour).Format(time.RFC3339)
	to := now.Add(time.Minute).Format(time.RFC3339)

	req := httptest.NewRequest("GET", usageURL(from, to, ""), nil)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want 401; body: %s", resp.StatusCode, body)
	}
}

func TestGetUsage_MemberRole_Returns403(t *testing.T) {
	t.Parallel()

	app, _, keyCache := setupTestApp(t, "file:TestGetUsage_Member?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleMember)

	now := time.Now().UTC()
	from := now.Add(-time.Hour).Format(time.RFC3339)
	to := now.Add(time.Minute).Format(time.RFC3339)

	req := httptest.NewRequest("GET", usageURL(from, to, ""), nil)
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want 403; body: %s", resp.StatusCode, body)
	}
}

func TestGetUsage_ResponseShape(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestGetUsage_Shape?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	now := time.Now().UTC()
	from := now.Add(-time.Hour)
	to := now.Add(time.Minute)

	insertUsageEventHTTP(t, database, "shape-1", "key-shape", "gpt-4", 100, 50, 150, now.Add(-30*time.Minute))

	req := httptest.NewRequest("GET", usageURL(from.Format(time.RFC3339), to.Format(time.RFC3339), ""), nil)
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body: %s", resp.StatusCode, body)
	}

	var envelope struct {
		From    string `json:"from"`
		To      string `json:"to"`
		GroupBy string `json:"group_by"`
		Data    []struct {
			GroupKey         string  `json:"group_key"`
			TotalRequests    int64   `json:"total_requests"`
			PromptTokens     int64   `json:"prompt_tokens"`
			CompletionTokens int64   `json:"completion_tokens"`
			TotalTokens      int64   `json:"total_tokens"`
			CostEstimate     float64 `json:"cost_estimate"`
			AvgDurationMS    float64 `json:"avg_duration_ms"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if envelope.From == "" {
		t.Error("from is empty")
	}
	if envelope.To == "" {
		t.Error("to is empty")
	}
	if len(envelope.Data) == 0 {
		t.Fatal("data is empty, want at least one row")
	}

	row := envelope.Data[0]
	if row.TotalRequests != 1 {
		t.Errorf("total_requests = %d, want 1", row.TotalRequests)
	}
	if row.PromptTokens != 100 {
		t.Errorf("prompt_tokens = %d, want 100", row.PromptTokens)
	}
	if row.CompletionTokens != 50 {
		t.Errorf("completion_tokens = %d, want 50", row.CompletionTokens)
	}
	if row.TotalTokens != 150 {
		t.Errorf("total_tokens = %d, want 150", row.TotalTokens)
	}
}

func TestGetUsage_EventsOutsideRange_NotCounted(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestGetUsage_Range?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	now := time.Now().UTC()
	from := now.Add(-time.Hour)
	to := now.Add(-10 * time.Minute)

	// In range.
	insertUsageEventHTTP(t, database, "range-in", "key-range", "gpt-4", 100, 50, 150, now.Add(-30*time.Minute))
	// Before range.
	insertUsageEventHTTP(t, database, "range-before", "key-range", "gpt-4", 9999, 9999, 9999, now.Add(-2*time.Hour))
	// After range.
	insertUsageEventHTTP(t, database, "range-after", "key-range", "gpt-4", 9999, 9999, 9999, now.Add(-5*time.Minute))

	req := httptest.NewRequest("GET", usageURL(from.Format(time.RFC3339), to.Format(time.RFC3339), ""), nil)
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body: %s", resp.StatusCode, body)
	}

	var got map[string]any
	decodeBody(t, resp.Body, &got)

	data, ok := got["data"].([]any)
	if !ok || len(data) == 0 {
		t.Fatal("data is empty or wrong type")
	}

	row := data[0].(map[string]any)
	totalRequests := row["total_requests"].(float64)
	if totalRequests != 1 {
		t.Errorf("total_requests = %v, want 1 (only in-range event)", totalRequests)
	}
	totalTokens := row["total_tokens"].(float64)
	if totalTokens != 150 {
		t.Errorf("total_tokens = %v, want 150", totalTokens)
	}
}

func insertUsageEventWithUserAndOrgHTTP(t *testing.T, database *db.DB, id, keyID, userID, modelName string, promptTokens, compTokens, totalTokens int64, createdAt time.Time) {
	t.Helper()
	query := fmt.Sprintf(
		`INSERT INTO usage_events
			(id, key_id, key_type, user_id, model_name,
			 prompt_tokens, completion_tokens, total_tokens, status_code, created_at)
		 VALUES
			('%s', '%s', 'user_key', '%s', '%s',
			 %d, %d, %d, 200, '%s')`,
		id, keyID, userID, modelName,
		promptTokens, compTokens, totalTokens,
		createdAt.UTC().Format(time.RFC3339),
	)
	if _, err := database.SQL().ExecContext(context.Background(), query); err != nil {
		t.Fatalf("insertUsageEventWithUserAndOrgHTTP id=%q: %v", id, err)
	}
}

// ---- GET /api/v1/usage/me ---------------------------------------------------

func TestMyUsage_Me_ReturnsCorrectTotals(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestMyUsage_Me?mode=memory&cache=private")
	userID := "my-user-id"
	testKey := addTestKeyWithUser(t, keyCache, auth.RoleMember, userID)

	now := time.Now().UTC()
	from := now.Add(-2 * time.Hour).Format(time.RFC3339)
	to := now.Add(time.Minute).Format(time.RFC3339)

	// Events matching user's key ID and user ID should be returned.
	insertUsageEventWithUserAndOrgHTTP(t, database, "me-1", "test-key-id-member-user", userID, "gpt-4", 100, 50, 150, now.Add(-90*time.Minute))
	// Events from other keys/users should be excluded.
	insertUsageEventWithUserAndOrgHTTP(t, database, "other-1", "other-key", "other-user", "gpt-4", 500, 500, 1000, now.Add(-30*time.Minute))

	req := httptest.NewRequest("GET", myUsageURL(from, to, ""), nil)
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body: %s", resp.StatusCode, body)
	}

	var envelope struct {
		Data []struct {
			TotalRequests int64 `json:"total_requests"`
			TotalTokens   int64 `json:"total_tokens"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(envelope.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(envelope.Data))
	}
	row := envelope.Data[0]
	if row.TotalRequests != 1 {
		t.Errorf("total_requests = %d, want 1", row.TotalRequests)
	}
	if row.TotalTokens != 150 {
		t.Errorf("total_tokens = %d, want 150", row.TotalTokens)
	}
}
