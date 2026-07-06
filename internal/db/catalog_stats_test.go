package db_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/voidmind-io/voidllm/internal/config"
	"github.com/voidmind-io/voidllm/internal/db"
)

func TestGetCatalogModelStats_AggregatesUsage(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             "file:TestGetCatalogModelStats?mode=memory&cache=private",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(ctx, database.SQL(), db.SQLiteDialect{}, slog.Default()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	id, _ := uuid.NewV7()
	_, err = database.SQL().ExecContext(ctx,
		`INSERT INTO usage_events
		 (id, key_id, key_type, model_name, prompt_tokens, completion_tokens, total_tokens,
		  request_duration_ms, tokens_per_second, status_code, created_at)
		 VALUES (?, 'k1', 'user', 'gpt-test', 10, 20, 30, 500, 42.5, 200, ?)`,
		id.String(), now,
	)
	if err != nil {
		t.Fatalf("insert usage: %v", err)
	}

	stats, err := database.GetCatalogModelStats(ctx, time.Now().UTC().Add(-24*time.Hour), []string{"gpt-test"})
	if err != nil {
		t.Fatalf("GetCatalogModelStats: %v", err)
	}
	s, ok := stats["gpt-test"]
	if !ok {
		t.Fatal("missing gpt-test stats")
	}
	if s.RequestCount != 1 {
		t.Fatalf("request_count = %d want 1", s.RequestCount)
	}
	if s.SuccessRate != 100 {
		t.Fatalf("success_rate = %v want 100", s.SuccessRate)
	}
	if s.AvgLatencyMs != 500 {
		t.Fatalf("avg_latency_ms = %v want 500", s.AvgLatencyMs)
	}
	if s.AvgTps != 42.5 {
		t.Fatalf("avg_tps = %v want 42.5", s.AvgTps)
	}
}

func TestGetCatalogModelStats_UsesRequestedModelName(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             "file:TestGetCatalogModelStats_Requested?mode=memory&cache=private",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(ctx, database.SQL(), db.SQLiteDialect{}, slog.Default()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	id, _ := uuid.NewV7()
	_, err = database.SQL().ExecContext(ctx,
		`INSERT INTO usage_events
		 (id, key_id, key_type, model_name, requested_model_name, prompt_tokens, completion_tokens, total_tokens,
		  request_duration_ms, tokens_per_second, status_code, created_at)
		 VALUES (?, 'k1', 'user', 'fallback-model', 'premium-chat', 10, 20, 30, 500, 42.5, 200, ?)`,
		id.String(), now,
	)
	if err != nil {
		t.Fatalf("insert usage: %v", err)
	}

	stats, err := database.GetCatalogModelStats(ctx, time.Now().UTC().Add(-24*time.Hour), []string{"premium-chat"})
	if err != nil {
		t.Fatalf("GetCatalogModelStats: %v", err)
	}
	s, ok := stats["premium-chat"]
	if !ok {
		t.Fatal("expected stats keyed by requested product name premium-chat")
	}
	if s.RequestCount != 1 {
		t.Fatalf("request_count = %d want 1", s.RequestCount)
	}
}