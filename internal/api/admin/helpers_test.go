package admin_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"

	"github.com/jukaza/tavo/internal/api/admin"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/cache"
	"github.com/jukaza/tavo/internal/config"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/pkg/keygen"
)

const testTimeout = 5 * time.Second
var testHMACSecret = []byte("test-hmac-secret-for-admin-api-tests")

func setupTestApp(t *testing.T, dsn string) (*fiber.App, *db.DB, *cache.Cache[string, auth.KeyInfo]) {
	t.Helper()

	ctx := context.Background()
	database, err := db.Open(ctx, config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             dsn,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("open test DB: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := db.RunMigrations(ctx, database.SQL(), db.SQLiteDialect{}, slog.Default()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	keyCache := cache.New[string, auth.KeyInfo]()

	handler := &admin.Handler{
		DB:         database,
		HMACSecret: testHMACSecret,
		KeyCache:   keyCache,
		Log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	app := fiber.New()
	admin.RegisterRoutes(app, handler, keyCache, testHMACSecret, nil)

	return app, database, keyCache
}

func addTestKey(t *testing.T, keyCache *cache.Cache[string, auth.KeyInfo], role string) string {
	t.Helper()

	plaintext, err := keygen.Generate(keygen.KeyTypeUser)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	hash := keygen.Hash(plaintext, testHMACSecret)
	keyCache.Set(hash, auth.KeyInfo{
		ID:      "test-key-id-" + role,
		KeyType: keygen.KeyTypeUser,
		Role:    role,
		Name:    "test key " + role,
	})

	return plaintext
}

func addTestKeyWithUser(t *testing.T, keyCache *cache.Cache[string, auth.KeyInfo], role, userID string) string {
	t.Helper()

	plaintext, err := keygen.Generate(keygen.KeyTypeUser)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	hash := keygen.Hash(plaintext, testHMACSecret)
	keyCache.Set(hash, auth.KeyInfo{
		ID:      "test-key-id-" + role + "-user",
		KeyType: keygen.KeyTypeUser,
		Role:    role,
		UserID:  userID,
		Name:    "test key " + role,
	})

	return plaintext
}

func mustCreateUserWithPassword(t *testing.T, database *db.DB, email, displayName, password string, isSystemAdmin bool) *db.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword: %v", err)
	}
	h := string(hash)
	user, err := database.CreateUser(context.Background(), db.CreateUserParams{
		Email:         email,
		DisplayName:   displayName,
		PasswordHash:  &h,
		IsSystemAdmin: isSystemAdmin,
	})
	if err != nil {
		t.Fatalf("mustCreateUserWithPassword(%q): %v", email, err)
	}
	return user
}

func bodyJSON(t *testing.T, v any) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	return strings.NewReader(string(b))
}

func decodeBody(t *testing.T, r io.Reader, v any) {
	t.Helper()
	if err := json.NewDecoder(r).Decode(v); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

func keysURL() string {
	return "/api/v1/keys"
}

func keyItemURL(keyID string) string {
	return "/api/v1/keys/" + keyID
}