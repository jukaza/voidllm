package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"os"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/voidmind-io/voidllm/internal/cache"
	"github.com/voidmind-io/voidllm/internal/config"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/pkg/keygen"
)

// BootstrapResult holds the plaintext credentials generated during a successful
// bootstrap. These values must be presented to the operator exactly once and
// are never persisted or logged through structured logging.
type BootstrapResult struct {
	// APIKey is the plaintext API key for the initial system-admin user.
	APIKey string
	// Email is the email address of the initial system-admin user.
	Email string
	// Password is the plaintext password for the initial system-admin user.
	Password string
}

// Bootstrap performs first-time setup when the database is empty.
// If cfg.AdminKey is non-empty and no API keys exist in the database, it
// creates a default organization, system admin user with a random password,
// and an API key. The returned BootstrapResult holds the plaintext credentials
// that the caller must present to the operator; it is nil when bootstrap is
// skipped (database already has keys or cfg.AdminKey is empty).
func Bootstrap(ctx context.Context, sqlDB *sql.DB, dialect db.Dialect,
	keyCache *cache.Cache[string, KeyInfo], cfg config.SettingsConfig,
	hmacSecret []byte, log *slog.Logger) (*BootstrapResult, error) {
	if cfg.AdminKey == "" {
		return nil, nil
	}

	if len(cfg.AdminKey) < 32 {
		return nil, errors.New("admin key must be at least 32 characters")
	}

	adminEmail := cfg.Bootstrap.AdminEmail

	userID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("bootstrap: generate user id: %w", err)
	}

	keyID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("bootstrap: generate key id: %w", err)
	}

	password, err := generatePassword(16)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: %w", err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: hash password: %w", err)
	}

	plaintextKey, err := keygen.Generate(keygen.KeyTypeUser)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: generate api key: %w", err)
	}

	keyHash := keygen.Hash(plaintextKey, hmacSecret)
	keyHint := keygen.Hint(plaintextKey)

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after successful Commit

	var count int
	row := tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM api_keys WHERE deleted_at IS NULL")
	if err = row.Scan(&count); err != nil {
		return nil, fmt.Errorf("bootstrap: count api_keys: %w", err)
	}

	if count > 0 {
		log.Warn("VOIDLLM_ADMIN_KEY is set but database already has keys, ignoring")
		return nil, nil
	}

	insertUser := "INSERT INTO users (id, email, display_name, password_hash, auth_provider, is_system_admin, created_at, updated_at) VALUES (" +
		dialect.Placeholder(1) + ", " +
		dialect.Placeholder(2) + ", " +
		dialect.Placeholder(3) + ", " +
		dialect.Placeholder(4) + ", " +
		dialect.Placeholder(5) + ", " +
		dialect.Placeholder(6) + ", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"
	if _, err = tx.ExecContext(ctx, insertUser,
		userID.String(),
		adminEmail,
		"Admin",
		string(passwordHash),
		"local",
		1,
	); err != nil {
		return nil, fmt.Errorf("bootstrap: insert user: %w", err)
	}

	insertKey := "INSERT INTO api_keys (id, key_hash, key_hint, key_type, name, user_id, created_by, created_at, updated_at) VALUES (" +
		dialect.Placeholder(1) + ", " +
		dialect.Placeholder(2) + ", " +
		dialect.Placeholder(3) + ", " +
		dialect.Placeholder(4) + ", " +
		dialect.Placeholder(5) + ", " +
		dialect.Placeholder(6) + ", " +
		dialect.Placeholder(7) + ", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"
	if _, err = tx.ExecContext(ctx, insertKey,
		keyID.String(),
		keyHash,
		keyHint,
		keygen.KeyTypeUser,
		"Bootstrap Admin Key",
		userID.String(),
		userID.String(),
	); err != nil {
		return nil, fmt.Errorf("bootstrap: insert api key: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("bootstrap: commit transaction: %w", err)
	}

	keyCache.Set(keyHash, KeyInfo{
		ID:      keyID.String(),
		KeyType: keygen.KeyTypeUser,
		Role:    RoleSystemAdmin,
		UserID:  userID.String(),
		Name:    "Bootstrap Admin Key",
	})

	os.Unsetenv("VOIDLLM_ADMIN_KEY")

	log.Warn("bootstrap complete, default organization and system admin created",
		slog.String("key_hint", keyHint))

	return &BootstrapResult{
		APIKey:   plaintextKey,
		Email:    adminEmail,
		Password: password,
	}, nil
}

// generatePassword creates a random alphanumeric password of the given length.
func generatePassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("generate password: %w", err)
		}
		b[i] = charset[idx.Int64()]
	}
	return string(b), nil
}
