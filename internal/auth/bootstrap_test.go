package auth

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/jukaza/tavo/internal/cache"
	"github.com/jukaza/tavo/internal/config"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/pkg/keygen"

	_ "modernc.org/sqlite"
)

var bootstrapSecret = []byte("bootstrap-test-secret-32-bytes!!")

func defaultBootstrapCfg(adminKey string) config.SettingsConfig {
	return config.SettingsConfig{
		AdminKey: adminKey,
		Bootstrap: config.BootstrapConfig{
			AdminEmail: "admin@tavo.local",
		},
	}
}

func setupBootstrapDB(t *testing.T, testName string) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=private", testName)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if err := db.RunMigrations(context.Background(), sqlDB, db.SQLiteDialect{}, slog.Default()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return sqlDB
}

func discardLogger() *slog.Logger {
	return slog.Default()
}

func countRows(t *testing.T, sqlDB *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := sqlDB.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM "+table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func TestBootstrap_EmptyAdminKey(t *testing.T) {
	t.Parallel()

	sqlDB := setupBootstrapDB(t, t.Name())
	keyCache := cache.New[string, KeyInfo]()

	_, err := Bootstrap(context.Background(), sqlDB, db.SQLiteDialect{},
		keyCache, defaultBootstrapCfg(""), bootstrapSecret, discardLogger())
	if err != nil {
		t.Fatalf("Bootstrap() with empty key returned error: %v", err)
	}

	if got := keyCache.Len(); got != 0 {
		t.Errorf("cache.Len() = %d, want 0", got)
	}
	if got := countRows(t, sqlDB, "users"); got != 0 {
		t.Errorf("users count = %d, want 0", got)
	}
	if got := countRows(t, sqlDB, "api_keys"); got != 0 {
		t.Errorf("api_keys count = %d, want 0", got)
	}
}

func TestBootstrap_AdminKeyTooShort(t *testing.T) {
	t.Parallel()

	sqlDB := setupBootstrapDB(t, t.Name())
	keyCache := cache.New[string, KeyInfo]()

	tests := []struct {
		name string
		key  string
	}{
		{name: "one char", key: "x"},
		{name: "31 chars", key: strings.Repeat("a", 31)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := Bootstrap(context.Background(), sqlDB, db.SQLiteDialect{},
				keyCache, defaultBootstrapCfg(tc.key), bootstrapSecret, discardLogger())
			if err == nil {
				t.Fatal("Bootstrap() returned nil, want error")
			}
			if !strings.Contains(err.Error(), "at least 32 characters") {
				t.Errorf("error = %q, want message containing %q",
					err.Error(), "at least 32 characters")
			}
		})
	}
}

func TestBootstrap_DBAlreadyHasKeys(t *testing.T) {
	t.Parallel()

	sqlDB := setupBootstrapDB(t, t.Name())
	keyCache := cache.New[string, KeyInfo]()
	cfg := defaultBootstrapCfg(strings.Repeat("b", 32))
	log := discardLogger()

	if _, err := Bootstrap(context.Background(), sqlDB, db.SQLiteDialect{},
		keyCache, cfg, bootstrapSecret, log); err != nil {
		t.Fatalf("first Bootstrap() call: %v", err)
	}

	users := countRows(t, sqlDB, "users")
	keys := countRows(t, sqlDB, "api_keys")

	if _, err := Bootstrap(context.Background(), sqlDB, db.SQLiteDialect{},
		keyCache, cfg, bootstrapSecret, log); err != nil {
		t.Fatalf("second Bootstrap() call: %v", err)
	}

	if got := countRows(t, sqlDB, "users"); got != users {
		t.Errorf("users count after second call = %d, want %d", got, users)
	}
	if got := countRows(t, sqlDB, "api_keys"); got != keys {
		t.Errorf("api_keys count after second call = %d, want %d", got, keys)
	}
}

func TestBootstrap_CreatesEntities(t *testing.T) {
	t.Parallel()

	sqlDB := setupBootstrapDB(t, t.Name())
	keyCache := cache.New[string, KeyInfo]()
	cfg := defaultBootstrapCfg(strings.Repeat("c", 32))

	if _, err := Bootstrap(context.Background(), sqlDB, db.SQLiteDialect{},
		keyCache, cfg, bootstrapSecret, discardLogger()); err != nil {
		t.Fatalf("Bootstrap(): %v", err)
	}

	var email string
	var isAdmin int
	if err := sqlDB.QueryRowContext(context.Background(),
		"SELECT email, is_system_admin FROM users WHERE deleted_at IS NULL").
		Scan(&email, &isAdmin); err != nil {
		t.Fatalf("query user: %v", err)
	}
	if email != "admin@tavo.local" {
		t.Errorf("user email = %q, want %q", email, "admin@tavo.local")
	}
	if isAdmin != 1 {
		t.Errorf("is_system_admin = %d, want 1", isAdmin)
	}
	if got := countRows(t, sqlDB, "users"); got != 1 {
		t.Errorf("users count = %d, want 1", got)
	}

	var keyType, keyName string
	if err := sqlDB.QueryRowContext(context.Background(),
		"SELECT key_type, name FROM api_keys WHERE deleted_at IS NULL").
		Scan(&keyType, &keyName); err != nil {
		t.Fatalf("query api_key: %v", err)
	}
	if keyType != keygen.KeyTypeUser {
		t.Errorf("key_type = %q, want %q", keyType, keygen.KeyTypeUser)
	}
	if keyName != "Bootstrap Admin Key" {
		t.Errorf("key name = %q, want %q", keyName, "Bootstrap Admin Key")
	}
	if got := countRows(t, sqlDB, "api_keys"); got != 1 {
		t.Errorf("api_keys count = %d, want 1", got)
	}
	if got := countRows(t, sqlDB, "wallets"); got != 1 {
		t.Errorf("wallets count = %d, want 1", got)
	}
}

func TestBootstrap_KeyInCache(t *testing.T) {
	t.Parallel()

	sqlDB := setupBootstrapDB(t, t.Name())
	keyCache := cache.New[string, KeyInfo]()
	cfg := defaultBootstrapCfg(strings.Repeat("d", 32))

	if _, err := Bootstrap(context.Background(), sqlDB, db.SQLiteDialect{},
		keyCache, cfg, bootstrapSecret, discardLogger()); err != nil {
		t.Fatalf("Bootstrap(): %v", err)
	}

	var storedHash string
	if err := sqlDB.QueryRowContext(context.Background(),
		"SELECT key_hash FROM api_keys WHERE deleted_at IS NULL").
		Scan(&storedHash); err != nil {
		t.Fatalf("query key_hash: %v", err)
	}

	keyInfo, ok := keyCache.Get(storedHash)
	if !ok {
		t.Fatal("key hash not found in cache")
	}

	if keyInfo.Role != RoleSystemAdmin {
		t.Errorf("cache KeyInfo.Role = %q, want %q", keyInfo.Role, RoleSystemAdmin)
	}
	if keyInfo.KeyType != keygen.KeyTypeUser {
		t.Errorf("cache KeyInfo.KeyType = %q, want %q", keyInfo.KeyType, keygen.KeyTypeUser)
	}
	if keyInfo.Name != "Bootstrap Admin Key" {
		t.Errorf("cache KeyInfo.Name = %q, want %q", keyInfo.Name, "Bootstrap Admin Key")
	}
	if keyInfo.UserID == "" {
		t.Error("cache KeyInfo.UserID is empty")
	}
	if keyInfo.ID == "" {
		t.Error("cache KeyInfo.ID is empty")
	}
}
