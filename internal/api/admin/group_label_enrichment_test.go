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
	"github.com/voidmind-io/voidllm/internal/cache"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/pkg/keygen"
)

// systemUsageURL constructs the GET /api/v1/usage URL with the given query params.
func systemUsageURL(from, to, groupBy string) string {
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

// myUsageURL constructs the GET /api/v1/usage/me URL with the given query params.
func myUsageURL(from, to, groupBy string) string {
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
	u := "/api/v1/usage/me"
	if len(params) > 0 {
		u += "?" + strings.Join(params, "&")
	}
	return u
}

// addTestKeyWithIDAndUser generates a key with a specific key ID and user ID
// in the cache. Used to seed MyUsage tests.
func addTestKeyWithIDAndUser(t *testing.T, keyCache *cache.Cache[string, auth.KeyInfo], role, keyID, userID string) string {
	t.Helper()

	plaintext, err := keygen.Generate(keygen.KeyTypeUser)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	hash := keygen.Hash(plaintext, testHMACSecret)
	keyCache.Set(hash, auth.KeyInfo{
		ID:      keyID,
		KeyType: keygen.KeyTypeUser,
		Role:    role,
		UserID:  userID,
		Name:    "test key " + role,
	})

	return plaintext
}

// ---- SystemAdminUsage group_label enrichment tests ---------------------------

// TestSystemAdminUsage_GroupByUser_HasGroupLabel verifies that the system-wide
// GET /api/v1/usage endpoint sets group_label to the user's display_name when
// group_by=user and a matching user row exists.
func TestSystemAdminUsage_GroupByUser_HasGroupLabel(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestSysUsage_GBUser_Label?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	// Create a real user whose ID will appear in usage_events.user_id.
	user, err := database.CreateUser(context.Background(), db.CreateUserParams{
		Email:       "sys-usage-user@example.com",
		DisplayName: "System Usage User",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	now := time.Now().UTC()
	from := now.Add(-2 * time.Hour).Format(time.RFC3339)
	to := now.Add(time.Minute).Format(time.RFC3339)

	// Seed a usage event referencing the created user's ID.
	insertUsageEventWithUserHTTP(t, database, "sys-gl-user-1", "key-sys-user", user.ID, "gpt-4",
		300, now.Add(-30*time.Minute))

	req := httptest.NewRequest("GET", systemUsageURL(from, to, "user"), nil)
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
			GroupKey   string `json:"group_key"`
			GroupLabel string `json:"group_label"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(envelope.Data) == 0 {
		t.Fatal("data is empty, want at least one row")
	}

	// Find the data point whose group_key matches our user's ID.
	var found bool
	for _, row := range envelope.Data {
		if row.GroupKey == user.ID {
			found = true
			if row.GroupLabel != "System Usage User" {
				t.Errorf("group_label = %q, want %q", row.GroupLabel, "System Usage User")
			}
			break
		}
	}
	if !found {
		t.Errorf("no data point with group_key = %q; all keys: %v", user.ID, envelope.Data)
	}
}

// TestMyUsage_GroupByKey_HasGroupLabel verifies that GET /api/v1/usage/me with
// group_by=key returns a data point whose group_label equals the key's Name when
// the calling key has a user_id and the key exists in api_keys.
func TestMyUsage_GroupByKey_HasGroupLabel(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestMyUsage_GBKey_Label?mode=memory&cache=private")

	// Create a real user and a real api_key row so the label resolver finds them.
	creator, err := database.CreateUser(context.Background(), db.CreateUserParams{
		Email:       "my-usage-creator@example.com",
		DisplayName: "My Usage Creator",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	apiKey, err := database.CreateAPIKey(context.Background(), db.CreateAPIKeyParams{
		KeyHash:   "dummy-hash-my-usage-key",
		KeyHint:   "vl_uk_...mu",
		KeyType:   "user_key",
		Name:      "My Labeled Key",
		CreatedBy: creator.ID,
	})
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	testKey := addTestKeyWithIDAndUser(t, keyCache, auth.RoleMember, apiKey.ID, creator.ID)

	now := time.Now().UTC()
	from := now.Add(-2 * time.Hour).Format(time.RFC3339)
	to := now.Add(time.Minute).Format(time.RFC3339)

	// Seed usage events with the user_id so that the scoped-by-user filter picks them up.
	insertUsageEventWithUserHTTP(t, database, "my-gl-key-1", apiKey.ID, creator.ID, "gpt-4",
		150, now.Add(-30*time.Minute))

	req := httptest.NewRequest("GET", myUsageURL(from, to, "key"), nil)
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
			GroupKey   string `json:"group_key"`
			GroupLabel string `json:"group_label"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(envelope.Data) == 0 {
		t.Fatal("data is empty, want at least one row")
	}

	row := envelope.Data[0]
	if row.GroupKey != apiKey.ID {
		t.Errorf("group_key = %q, want %q", row.GroupKey, apiKey.ID)
	}
	if row.GroupLabel != "My Labeled Key" {
		t.Errorf("group_label = %q, want %q", row.GroupLabel, "My Labeled Key")
	}
}

func insertUsageEventWithUserHTTP(t *testing.T, database *db.DB, id, keyID, userID, modelName string, totalTokens int64, createdAt time.Time) {
	t.Helper()
	query := fmt.Sprintf(
		`INSERT INTO usage_events
			(id, key_id, key_type, user_id, model_name,
			 prompt_tokens, completion_tokens, total_tokens, status_code, created_at)
		 VALUES
			('%s', '%s', 'user_key', '%s', '%s',
			 0, %d, %d, 200, '%s')`,
		id, keyID, userID, modelName,
		totalTokens, totalTokens,
		createdAt.UTC().Format(time.RFC3339),
	)
	if _, err := database.SQL().ExecContext(context.Background(), query); err != nil {
		t.Fatalf("insertUsageEventWithUserHTTP id=%q: %v", id, err)
	}
}

