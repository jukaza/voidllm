package admin_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
)

func mustCreateUser(t *testing.T, database *db.DB, email, displayName string) *db.User {
	t.Helper()
	hash := "$2a$04$testu1testu2testu3testu4testhashXXXXXXXXXXXXXXXXXXXX"
	user, err := database.CreateUser(context.Background(), db.CreateUserParams{
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: &hash,
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("mustCreateUser(%q): %v", email, err)
	}
	return user
}

// ---- POST /api/v1/users ------------------------------------------------------

func TestCreateUser(t *testing.T) {
	t.Parallel()

	t.Run("system_admin creates user returns 201", func(t *testing.T) {
		t.Parallel()

		app, _, keyCache := setupTestApp(t, "file:TestCreateUser_SysAdmin201?mode=memory&cache=private")
		testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

		req := httptest.NewRequest("POST", "/api/v1/users", bodyJSON(t, map[string]any{
			"email":        "newuser@example.com",
			"display_name": "New User",
			"password":     "securepassword",
		}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+testKey)

		resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("status = %d, want 201; body: %s", resp.StatusCode, body)
		}
	})

	t.Run("missing email returns 400", func(t *testing.T) {
		t.Parallel()

		app, _, keyCache := setupTestApp(t, "file:TestCreateUser_MissingEmail?mode=memory&cache=private")
		testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

		req := httptest.NewRequest("POST", "/api/v1/users", bodyJSON(t, map[string]any{
			"display_name": "No Email",
			"password":     "securepassword",
		}))
		req.Header.Set("Content-Type", "application/json")
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
	})

	t.Run("password too short returns 400", func(t *testing.T) {
		t.Parallel()

		app, _, keyCache := setupTestApp(t, "file:TestCreateUser_ShortPassword?mode=memory&cache=private")
		testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

		req := httptest.NewRequest("POST", "/api/v1/users", bodyJSON(t, map[string]any{
			"email":        "shortpwd@example.com",
			"display_name": "Short Pwd",
			"password":     "123",
		}))
		req.Header.Set("Content-Type", "application/json")
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
	})
}

func TestCreateUser_NoAuth(t *testing.T) {
	t.Parallel()

	app, _, _ := setupTestApp(t, "file:TestCreateUser_NoAuth?mode=memory&cache=private")

	req := httptest.NewRequest("POST", "/api/v1/users", bodyJSON(t, map[string]any{
		"email":        "unauth@example.com",
		"display_name": "Unauth User",
		"password":     "securepassword",
	}))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestCreateUser_DupEmail?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)
	mustCreateUser(t, database, "existing@example.com", "Existing User")

	req := httptest.NewRequest("POST", "/api/v1/users", bodyJSON(t, map[string]any{
		"email":        "existing@example.com",
		"display_name": "Duplicate",
		"password":     "securepassword",
	}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want 409; body: %s", resp.StatusCode, body)
	}
}

func TestCreateUser_ResponseHasNoPassword(t *testing.T) {
	t.Parallel()

	app, _, keyCache := setupTestApp(t, "file:TestCreateUser_ResponseNoPwd?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	req := httptest.NewRequest("POST", "/api/v1/users", bodyJSON(t, map[string]any{
		"email":        "nopwdresp@example.com",
		"display_name": "No Pwd Resp User",
		"password":     "supersecretpassword",
	}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 201; body: %s", resp.StatusCode, body)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if _, ok := raw["password"]; ok {
		t.Error("response contains 'password' key")
	}
	if _, ok := raw["password_hash"]; ok {
		t.Error("response contains 'password_hash' key")
	}
}

func TestCreateUser_PasswordTooLong(t *testing.T) {
	t.Parallel()

	app, _, keyCache := setupTestApp(t, "file:TestCreateUser_PwdTooLong?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)

	tooLongPwd := strings.Repeat("a", 73)
	req := httptest.NewRequest("POST", "/api/v1/users", bodyJSON(t, map[string]any{
		"email":        "toolongpwd@example.com",
		"display_name": "Too Long Pwd",
		"password":     tooLongPwd,
	}))
	req.Header.Set("Content-Type", "application/json")
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

// ---- GET /api/v1/users/:user_id ----------------------------------------------

func TestGetUser(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestGetUser?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)
	user := mustCreateUser(t, database, "getuser@example.com", "Get User")

	t.Run("system_admin gets user returns 200", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/"+user.ID, nil)
		req.Header.Set("Authorization", "Bearer "+testKey)

		resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("non-existent user returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/00000000-0000-0000-0000-000000000000", nil)
		req.Header.Set("Authorization", "Bearer "+testKey)

		resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
	})
}

// ---- GET /api/v1/users -------------------------------------------------------

func TestListUsers(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestListUsers?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)
	mustCreateUser(t, database, "listuser1@example.com", "List User 1")
	mustCreateUser(t, database, "listuser2@example.com", "List User 2")

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var envelope struct {
		Data []db.User `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(envelope.Data) < 2 {
		t.Errorf("len(data) = %d, want at least 2", len(envelope.Data))
	}
}

// ---- PATCH /api/v1/users/:user_id --------------------------------------------

func TestUpdateUser(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestUpdateUser?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)
	user := mustCreateUser(t, database, "updateuser@example.com", "Update User")

	t.Run("system_admin updates user returns 200", func(t *testing.T) {
		req := httptest.NewRequest("PATCH", "/api/v1/users/"+user.ID, bodyJSON(t, map[string]any{
			"display_name": "Updated Display Name",
		}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+testKey)

		resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("non-existent user returns 404", func(t *testing.T) {
		req := httptest.NewRequest("PATCH", "/api/v1/users/00000000-0000-0000-0000-000000000000", bodyJSON(t, map[string]any{
			"display_name": "Ghost Update",
		}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+testKey)

		resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
	})
}

// ---- DELETE /api/v1/users/:user_id -------------------------------------------

func TestDeleteUser(t *testing.T) {
	t.Parallel()

	app, database, keyCache := setupTestApp(t, "file:TestDeleteUser?mode=memory&cache=private")
	testKey := addTestKey(t, keyCache, auth.RoleSystemAdmin)
	user := mustCreateUser(t, database, "deleteuser@example.com", "Delete User")

	t.Run("system_admin deletes user returns 204", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/users/"+user.ID, nil)
		req.Header.Set("Authorization", "Bearer "+testKey)

		resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusNoContent {
			t.Errorf("status = %d, want 204", resp.StatusCode)
		}
	})

	t.Run("non-existent user returns 404", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/users/00000000-0000-0000-0000-000000000000", nil)
		req.Header.Set("Authorization", "Bearer "+testKey)

		resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
	})
}
