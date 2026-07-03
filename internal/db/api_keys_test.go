package db

import (
	"context"
	"errors"
	"testing"

	"github.com/voidmind-io/voidllm/pkg/keygen"
)

var testHMACSecret = []byte("test-hmac-secret-for-db-layer-tests")

func mustCreateAPIKey(t *testing.T, d *DB, params CreateAPIKeyParams) *APIKey {
	t.Helper()
	key, err := d.CreateAPIKey(context.Background(), params)
	if err != nil {
		t.Fatalf("mustCreateAPIKey(%q): %v", params.Name, err)
	}
	return key
}

func testKeyParams(userID string) CreateAPIKeyParams {
	plaintext := "vl_uk_deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	return CreateAPIKeyParams{
		KeyHash:   keygen.Hash(plaintext, testHMACSecret),
		KeyHint:   keygen.Hint(plaintext),
		KeyType:   keygen.KeyTypeUser,
		Name:      "test key",
		UserID:    &userID,
		CreatedBy: userID,
	}
}

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	user := mustCreateUser(t, d, CreateUserParams{Email: "u@example.com", DisplayName: "U"})

	params := testKeyParams(user.ID)
	got, err := d.CreateAPIKey(context.Background(), params)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	if got.ID == "" {
		t.Error("ID is empty, want non-empty UUID")
	}
	if got.KeyHash != params.KeyHash {
		t.Errorf("KeyHash = %q, want %q", got.KeyHash, params.KeyHash)
	}
	if got.KeyHint != params.KeyHint {
		t.Errorf("KeyHint = %q, want %q", got.KeyHint, params.KeyHint)
	}
	if got.KeyType != keygen.KeyTypeUser {
		t.Errorf("KeyType = %q, want %q", got.KeyType, keygen.KeyTypeUser)
	}
	if got.Name != params.Name {
		t.Errorf("Name = %q, want %q", got.Name, params.Name)
	}
	if got.UserID == nil || *got.UserID != *params.UserID {
		t.Errorf("UserID = %v, want %q", got.UserID, *params.UserID)
	}
}

func TestGetAPIKey(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	user := mustCreateUser(t, d, CreateUserParams{Email: "g1@example.com", DisplayName: "G1"})
	k := mustCreateAPIKey(t, d, testKeyParams(user.ID))

	got, err := d.GetAPIKey(context.Background(), k.ID)
	if err != nil {
		t.Fatalf("GetAPIKey failed: %v", err)
	}
	if got.ID != k.ID {
		t.Errorf("GetAPIKey ID = %q, want %q", got.ID, k.ID)
	}

	_, err = d.GetAPIKey(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetAPIKey non-existent error = %v, want ErrNotFound", err)
	}
}

func TestListAPIKeys(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	user := mustCreateUser(t, d, CreateUserParams{Email: "list@example.com", DisplayName: "L"})

	plain1 := "vl_uk_listtesta0000000000000000000000000"
	mustCreateAPIKey(t, d, CreateAPIKeyParams{
		KeyHash:   keygen.Hash(plain1, testHMACSecret),
		KeyHint:   keygen.Hint(plain1),
		KeyType:   keygen.KeyTypeUser,
		Name:      "key-a",
		UserID:    &user.ID,
		CreatedBy: user.ID,
	})

	keys, err := d.ListAPIKeys(context.Background(), user.ID, "", 100, false)
	if err != nil {
		t.Fatalf("ListAPIKeys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("ListAPIKeys length = %d, want 1", len(keys))
	}
}

func TestUpdateAPIKey(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	user := mustCreateUser(t, d, CreateUserParams{Email: "upd1@example.com", DisplayName: "U"})
	k := mustCreateAPIKey(t, d, testKeyParams(user.ID))

	newName := "renamed key"
	updated, err := d.UpdateAPIKey(context.Background(), k.ID, UpdateAPIKeyParams{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("UpdateAPIKey failed: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("updated Name = %q, want %q", updated.Name, newName)
	}
}

func TestDeleteAPIKey(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	user := mustCreateUser(t, d, CreateUserParams{Email: "dk@example.com", DisplayName: "DK"})
	k := mustCreateAPIKey(t, d, testKeyParams(user.ID))

	err := d.DeleteAPIKey(context.Background(), k.ID)
	if err != nil {
		t.Fatalf("DeleteAPIKey failed: %v", err)
	}

	_, err = d.GetAPIKey(context.Background(), k.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetAPIKey after delete error = %v, want ErrNotFound", err)
	}
}

func TestGetUserOrgRole(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	user := mustCreateUser(t, d, CreateUserParams{Email: "role@example.com", DisplayName: "R"})

	role, err := d.GetUserOrgRole(context.Background(), user.ID, "")
	if err != nil {
		t.Fatalf("GetUserOrgRole failed: %v", err)
	}
	if role != "member" {
		t.Errorf("GetUserOrgRole = %q, want %q", role, "member")
	}

	adminUser := mustCreateUser(t, d, CreateUserParams{Email: "roleadmin@example.com", DisplayName: "RA", IsSystemAdmin: true})
	adminRole, err := d.GetUserOrgRole(context.Background(), adminUser.ID, "")
	if err != nil {
		t.Fatalf("GetUserOrgRole failed for admin: %v", err)
	}
	if adminRole != "system_admin" {
		t.Errorf("GetUserOrgRole admin = %q, want %q", adminRole, "system_admin")
	}
}
