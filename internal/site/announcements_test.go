package site_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jukaza/tavo/internal/config"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/internal/site"
)

func TestAnnouncementsRoundTrip(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, config.DatabaseConfig{
		Driver: "sqlite", DSN: ":memory:", MaxOpenConns: 1, MaxIdleConns: 1, ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.RunMigrations(ctx, database.SQL(), db.SQLiteDialect{}, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	items := []site.Announcement{{
		ID:          "ann-1",
		Content:     "**Bold** and *italic*",
		PublishDate: time.Now().UTC().Format(time.RFC3339),
		Type:        "warning",
		Extra:       "Subtitle here",
	}}
	_, err = site.Update(ctx, database, site.UpdateInput{Announcements: &items})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	cfg, err := site.Load(ctx, database)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.Announcements) != 1 {
		t.Fatalf("announcements len = %d", len(cfg.Announcements))
	}
	if cfg.Announcements[0].Content != "**Bold** and *italic*" {
		t.Fatalf("content = %q", cfg.Announcements[0].Content)
	}
}