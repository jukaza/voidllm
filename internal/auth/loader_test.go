package auth

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/voidmind-io/voidllm/internal/cache"
	"github.com/voidmind-io/voidllm/internal/config"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/pkg/keygen"
)

var loaderHMACSecret = []byte("loader-test-hmac-secret-32bytes!")

func openLoaderDB(t *testing.T) *db.DB {
	t.Helper()
	safeName := strings.NewReplacer("/", "_", " ", "_", "#", "_", "=", "_").Replace(t.Name())
	cfg := config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             "file:" + safeName + "?mode=memory&cache=shared",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	}
	ctx := context.Background()
	d, err := db.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	if err := db.RunMigrations(ctx, d.SQL(), d.Dialect(), slog.Default()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return d
}

func createUser(t *testing.T, d *db.DB, email string, isSystemAdmin bool) string {
	t.Helper()
	user, err := d.CreateUser(context.Background(), db.CreateUserParams{
		Email:         email,
		DisplayName:   "Test User",
		AuthProvider:  "local",
		IsSystemAdmin: isSystemAdmin,
	})
	if err != nil {
		t.Fatalf("CreateUser(%q): %v", email, err)
	}
	return user.ID
}

func insertKey(t *testing.T, d *db.DB, params db.CreateAPIKeyParams) string {
	t.Helper()
	plaintext, err := keygen.Generate(params.KeyType)
	if err != nil {
		t.Fatalf("keygen.Generate(%s): %v", params.KeyType, err)
	}
	params.KeyHash = keygen.Hash(plaintext, loaderHMACSecret)
	params.KeyHint = keygen.Hint(plaintext)
	_, err = d.CreateAPIKey(context.Background(), params)
	if err != nil {
		t.Fatalf("CreateAPIKey(%q): %v", params.Name, err)
	}
	return plaintext
}

func softDeleteKey(t *testing.T, d *db.DB, plaintext string) {
	t.Helper()
	hash := keygen.Hash(plaintext, loaderHMACSecret)
	_, err := d.SQL().ExecContext(context.Background(),
		"UPDATE api_keys SET deleted_at = CURRENT_TIMESTAMP WHERE key_hash = ?", hash)
	if err != nil {
		t.Fatalf("softDeleteKey: %v", err)
	}
}

func TestLoadKeysIntoCache(t *testing.T) {
	t.Parallel()
	d := openLoaderDB(t)
	kc := cache.New[string, KeyInfo]()
	log := slog.Default()

	userID := createUser(t, d, "u1@example.com", false)
	pt := insertKey(t, d, db.CreateAPIKeyParams{
		KeyType:   keygen.KeyTypeUser,
		Name:      "uk1",
		UserID:    &userID,
		CreatedBy: userID,
	})

	if err := LoadKeysIntoCache(context.Background(), d, kc, log); err != nil {
		t.Fatalf("LoadKeysIntoCache: %v", err)
	}

	ki, ok := kc.Get(keygen.Hash(pt, loaderHMACSecret))
	if !ok {
		t.Fatal("key not found in cache")
	}
	if ki.KeyType != keygen.KeyTypeUser {
		t.Errorf("KeyType = %q, want %q", ki.KeyType, keygen.KeyTypeUser)
	}
}

func TestLoadKeysIntoCache_LoadAllReplacesCache(t *testing.T) {
	t.Parallel()

	d := openLoaderDB(t)
	kc := cache.New[string, KeyInfo]()
	log := slog.Default()
	ctx := context.Background()

	u1 := createUser(t, d, "reload1@example.com", false)

	pt1 := insertKey(t, d, db.CreateAPIKeyParams{
		KeyType:   keygen.KeyTypeUser,
		Name:      "reload-key-1",
		UserID:    &u1,
		CreatedBy: u1,
	})
	pt2 := insertKey(t, d, db.CreateAPIKeyParams{
		KeyType:   keygen.KeyTypeUser,
		Name:      "reload-key-2",
		UserID:    &u1,
		CreatedBy: u1,
	})

	if err := LoadKeysIntoCache(ctx, d, kc, log); err != nil {
		t.Fatalf("first LoadKeysIntoCache() error = %v", err)
	}
	if kc.Len() != 2 {
		t.Fatalf("cache.Len() after first load = %d, want 2", kc.Len())
	}

	softDeleteKey(t, d, pt2)

	if err := LoadKeysIntoCache(ctx, d, kc, log); err != nil {
		t.Fatalf("second LoadKeysIntoCache() error = %v", err)
	}
	if kc.Len() != 1 {
		t.Errorf("cache.Len() after reload = %d, want 1", kc.Len())
	}
	if _, ok := kc.Get(keygen.Hash(pt1, loaderHMACSecret)); !ok {
		t.Error("pt1 not found in cache after reload, want present")
	}
	if _, ok := kc.Get(keygen.Hash(pt2, loaderHMACSecret)); ok {
		t.Error("pt2 still in cache after reload, want absent")
	}
}

func TestStartCacheRefresh(t *testing.T) {
	t.Parallel()

	d := openLoaderDB(t)
	kc := cache.New[string, KeyInfo]()
	log := slog.Default()

	stop := StartCacheRefresh(d, kc, 50*time.Millisecond, log)
	t.Cleanup(stop)

	userID := createUser(t, d, "refresh@example.com", false)
	plaintext := insertKey(t, d, db.CreateAPIKeyParams{
		KeyType:   keygen.KeyTypeUser,
		Name:      "refresh-key",
		UserID:    &userID,
		CreatedBy: userID,
	})
	hash := keygen.Hash(plaintext, loaderHMACSecret)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, ok := kc.Get(hash); ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Error("key did not appear in cache within 500ms after StartCacheRefresh")
}
