package site_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/voidmind-io/voidllm/internal/config"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/internal/site"
)

func TestEnsureDefaultsAndUpdate(t *testing.T) {
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

	cfg, err := site.Load(ctx, database)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.SystemName != site.DefaultSystemName {
		t.Fatalf("system_name = %q", cfg.SystemName)
	}

	name := "My AI Portal"
	updated, err := site.Update(ctx, database, site.UpdateInput{
		SystemName: &name,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.SystemName != name {
		t.Fatalf("updated system_name = %q", updated.SystemName)
	}
}