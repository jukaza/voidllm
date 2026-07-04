package db

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/voidmind-io/voidllm/internal/config"
)

func openMigratedDB(t *testing.T) *DB {
	t.Helper()

	safeName := strings.NewReplacer("/", "_", " ", "_", "#", "_").Replace(t.Name())

	cfg := config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             "file:" + safeName + "?mode=memory&cache=private",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	}
	ctx := context.Background()
	d, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	if err := RunMigrations(ctx, d.SQL(), d.Dialect(), slog.Default()); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}

	return d
}

func ptr[T any](v T) *T {
	return &v
}