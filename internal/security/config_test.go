package security_test

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jukaza/tavo/internal/config"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/internal/security"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	ctx := context.Background()
	database, err := db.Open(ctx, config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             ":memory:",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.RunMigrations(ctx, database.SQL(), db.SQLiteDialect{}, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func TestLoadPolicyDefaults(t *testing.T) {
	ctx := context.Background()
	database := openTestDB(t)

	cfg, err := security.Load(ctx, database)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Session.TTLHours != security.DefaultSessionTTLHours {
		t.Fatalf("session ttl_hours = %d, want %d", cfg.Session.TTLHours, security.DefaultSessionTTLHours)
	}
	if cfg.Session.AllowMultiple {
		t.Fatal("expected session.allow_multiple default false")
	}
	if cfg.Session.MaxConcurrent != security.DefaultSessionMaxConcurrent {
		t.Fatalf("session max_concurrent = %d, want %d", cfg.Session.MaxConcurrent, security.DefaultSessionMaxConcurrent)
	}
	if cfg.Password.MinLength != security.DefaultPasswordMinLength {
		t.Fatalf("password min_length = %d, want %d", cfg.Password.MinLength, security.DefaultPasswordMinLength)
	}
	if !cfg.Password.AllowOAuthSetPassword {
		t.Fatal("expected password.allow_oauth_set_password default true")
	}
}

func TestUpdatePolicyRoundTrip(t *testing.T) {
	ctx := context.Background()
	database := openTestDB(t)

	ttl := 48
	allowMultiple := true
	maxConcurrent := 3
	minLength := 12
	allowOAuthSet := false

	updated, err := security.Update(ctx, database, security.UpdateInput{
		Session: &security.SessionPolicyUpdate{
			TTLHours:      &ttl,
			AllowMultiple: &allowMultiple,
			MaxConcurrent: &maxConcurrent,
		},
		Password: &security.PasswordPolicyUpdate{
			MinLength:             &minLength,
			AllowOAuthSetPassword: &allowOAuthSet,
		},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Session.TTLHours != 48 || !updated.Session.AllowMultiple || updated.Session.MaxConcurrent != 3 {
		t.Fatalf("session = %+v", updated.Session)
	}
	if updated.Password.MinLength != 12 || updated.Password.AllowOAuthSetPassword {
		t.Fatalf("password = %+v", updated.Password)
	}

	reloaded, err := security.Load(ctx, database)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Session != updated.Session || reloaded.Password != updated.Password {
		t.Fatalf("reload mismatch:\n got  session=%+v password=%+v\n want session=%+v password=%+v",
			reloaded.Session, reloaded.Password,
			updated.Session, updated.Password)
	}
}

func TestUpdatePolicyValidation(t *testing.T) {
	ctx := context.Background()
	database := openTestDB(t)

	badTTL := 0
	_, err := security.Update(ctx, database, security.UpdateInput{
		Session: &security.SessionPolicyUpdate{TTLHours: &badTTL},
	})
	if err != security.ErrSessionTTLOutOfRange {
		t.Fatalf("ttl error = %v, want ErrSessionTTLOutOfRange", err)
	}

	badMax := 100
	_, err = security.Update(ctx, database, security.UpdateInput{
		Session: &security.SessionPolicyUpdate{MaxConcurrent: &badMax},
	})
	if err != security.ErrSessionMaxConcurrentOutOfRange {
		t.Fatalf("max_concurrent error = %v, want ErrSessionMaxConcurrentOutOfRange", err)
	}

	badMin := 4
	_, err = security.Update(ctx, database, security.UpdateInput{
		Password: &security.PasswordPolicyUpdate{MinLength: &badMin},
	})
	if err != security.ErrPasswordMinLengthOutOfRange {
		t.Fatalf("min_length error = %v, want ErrPasswordMinLengthOutOfRange", err)
	}
}

func TestMigration0036RemovesTOTP(t *testing.T) {
	ctx := context.Background()
	database := openTestDB(t)

	var name string
	err := database.SQL().QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='user_totp_backup_codes'",
	).Scan(&name)
	if err != sql.ErrNoRows {
		t.Fatalf("expected user_totp_backup_codes table removed, got name=%q err=%v", name, err)
	}

	for _, col := range []string{"totp_secret_encrypted", "totp_enabled_at"} {
		var count int
		err := database.SQL().QueryRowContext(ctx,
			"SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = ?", col,
		).Scan(&count)
		if err != nil {
			t.Fatalf("pragma users.%s: %v", col, err)
		}
		if count != 0 {
			t.Fatalf("users.%s column should be removed", col)
		}
	}

	for _, col := range []string{"login_ip", "user_agent"} {
		var count int
		err := database.SQL().QueryRowContext(ctx,
			"SELECT COUNT(*) FROM pragma_table_info('api_keys') WHERE name = ?", col,
		).Scan(&count)
		if err != nil {
			t.Fatalf("pragma api_keys.%s: %v", col, err)
		}
		if count != 1 {
			t.Fatalf("api_keys.%s column missing", col)
		}
	}
}