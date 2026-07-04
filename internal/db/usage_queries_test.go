package db

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"
)

func insertTestUsageEvent(t *testing.T, d *DB, id, keyID, userID string, totalTokens int64, createdAt time.Time) {
	t.Helper()
	userVal := "NULL"
	if userID != "" {
		userVal = fmt.Sprintf("'%s'", userID)
	}
	query := fmt.Sprintf(
		`INSERT INTO usage_events
			(id, key_id, key_type, user_id, model_name,
			 prompt_tokens, completion_tokens, total_tokens, status_code, created_at)
		 VALUES
			('%s', '%s', 'user_key', %s, 'test-model',
			 0, %d, %d, 200, '%s')`,
		id, keyID, userVal,
		totalTokens, totalTokens,
		createdAt.UTC().Format(time.RFC3339),
	)
	if _, err := d.sql.ExecContext(context.Background(), query); err != nil {
		t.Fatalf("insertTestUsageEvent id=%q: %v", id, err)
	}
}

type usageEventParams struct {
	id           string
	keyID        string
	userID       string
	modelName    string
	promptTokens int64
	compTokens   int64
	totalTokens  int64
	costEstimate *float64
	durationMS   *int64
	createdAt    time.Time
}

func insertUsageEvent(t *testing.T, d *DB, p usageEventParams) {
	t.Helper()

	userVal := "NULL"
	if p.userID != "" {
		userVal = fmt.Sprintf("'%s'", p.userID)
	}
	costVal := "NULL"
	if p.costEstimate != nil {
		costVal = fmt.Sprintf("%f", *p.costEstimate)
	}
	durVal := "NULL"
	if p.durationMS != nil {
		durVal = fmt.Sprintf("%d", *p.durationMS)
	}
	if p.modelName == "" {
		p.modelName = "test-model"
	}

	query := fmt.Sprintf(
		`INSERT INTO usage_events
			(id, key_id, key_type, user_id, model_name,
			 prompt_tokens, completion_tokens, total_tokens,
			 cost_estimate, request_duration_ms, status_code, created_at)
		 VALUES
			('%s', '%s', 'user_key', %s, '%s',
			 %d, %d, %d,
			 %s, %s, 200, '%s')`,
		p.id, p.keyID, userVal, p.modelName,
		p.promptTokens, p.compTokens, p.totalTokens,
		costVal, durVal,
		p.createdAt.UTC().Format(time.RFC3339),
	)
	if _, err := d.sql.ExecContext(context.Background(), query); err != nil {
		t.Fatalf("insertUsageEvent id=%q: %v", p.id, err)
	}
}

func float64Ptr(v float64) *float64 { return &v }
func int64Ptr(v int64) *int64       { return &v }

func TestGetTokenUsageSince_NoEvents(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	since := time.Now().UTC().Add(-24 * time.Hour)

	total, err := d.GetTokenUsageSince(context.Background(), ScopeKey, "key-no-events", since)
	if err != nil {
		t.Fatalf("GetTokenUsageSince() error = %v, want nil", err)
	}
	if total != 0 {
		t.Errorf("GetTokenUsageSince() = %d, want 0", total)
	}
}

func TestGetTokenUsageSince_ScopeKey_ReturnsOnlyMatchingKey(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	since := now.Add(-time.Hour)

	insertTestUsageEvent(t, d, "evt-k1-a", "key-scope-a", "user-1", 300, now.Add(-30*time.Minute))
	insertTestUsageEvent(t, d, "evt-k1-b", "key-scope-a", "user-1", 200, now.Add(-10*time.Minute))
	insertTestUsageEvent(t, d, "evt-k2-a", "key-scope-b", "user-2", 999, now.Add(-5*time.Minute))

	total, err := d.GetTokenUsageSince(ctx, ScopeKey, "key-scope-a", since)
	if err != nil {
		t.Fatalf("GetTokenUsageSince(ScopeKey) error = %v", err)
	}
	if total != 500 {
		t.Errorf("GetTokenUsageSince(ScopeKey) = %d, want 500", total)
	}
}

func TestGetTokenUsageSince_ScopeUser_ReturnsOnlyMatchingUser(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	since := now.Add(-time.Hour)

	insertTestUsageEvent(t, d, "evt-u1-a", "key-u1", "user-scope-a", 400, now.Add(-40*time.Minute))
	insertTestUsageEvent(t, d, "evt-u1-b", "key-u2", "user-scope-a", 100, now.Add(-20*time.Minute))
	insertTestUsageEvent(t, d, "evt-u2-a", "key-u3", "user-scope-b", 777, now.Add(-15*time.Minute))

	total, err := d.GetTokenUsageSince(ctx, ScopeUser, "user-scope-a", since)
	if err != nil {
		t.Fatalf("GetTokenUsageSince(ScopeUser) error = %v", err)
	}
	if total != 500 {
		t.Errorf("GetTokenUsageSince(ScopeUser) = %d, want 500", total)
	}
}

func TestGetTokenUsageSince_EventsBeforeSinceNotCounted(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	since := now.Add(-time.Hour)

	insertTestUsageEvent(t, d, "evt-before-a", "key-before", "user-before", 500, now.Add(-2*time.Hour))

	total, err := d.GetTokenUsageSince(ctx, ScopeKey, "key-before", since)
	if err != nil {
		t.Fatalf("GetTokenUsageSince() error = %v", err)
	}
	if total != 0 {
		t.Errorf("GetTokenUsageSince() = %d, want 0", total)
	}
}

func TestGetSystemUsageAggregates_NoEvents_NoGrouping(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	rows, err := d.GetSystemUsageAggregates(ctx, now.Add(-time.Hour), now, "")
	if err != nil {
		t.Fatalf("GetSystemUsageAggregates() error = %v", err)
	}
	if len(rows) == 0 {
		return
	}
	if len(rows) != 1 || rows[0].TotalRequests != 0 {
		t.Errorf("unexpected rows for empty usage: %+v", rows)
	}
}

func TestGetSystemUsageAggregates_NoGrouping_CorrectTotals(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	from := now.Add(-2 * time.Hour)
	to := now.Add(time.Minute)

	insertUsageEvent(t, d, usageEventParams{
		id: "agg-tot-1", keyID: "key-agg-1", userID: "user-agg",
		promptTokens: 100, compTokens: 50, totalTokens: 150,
		createdAt: now.Add(-90 * time.Minute),
	})
	insertUsageEvent(t, d, usageEventParams{
		id: "agg-tot-2", keyID: "key-agg-1", userID: "user-agg",
		promptTokens: 200, compTokens: 80, totalTokens: 280,
		createdAt: now.Add(-60 * time.Minute),
	})
	insertUsageEvent(t, d, usageEventParams{
		id: "agg-tot-3", keyID: "key-agg-1", userID: "user-agg",
		promptTokens: 50, compTokens: 20, totalTokens: 70,
		createdAt: now.Add(-30 * time.Minute),
	})

	rows, err := d.GetSystemUsageAggregates(ctx, from, to, "")
	if err != nil {
		t.Fatalf("GetSystemUsageAggregates() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	r := rows[0]
	if r.TotalRequests != 3 || r.TotalTokens != 500 {
		t.Errorf("totals = requests:%d tokens:%d, want 3/500", r.TotalRequests, r.TotalTokens)
	}
}

func TestGetSystemUsageAggregates_GroupByModel(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	from := now.Add(-2 * time.Hour)
	to := now.Add(time.Minute)

	insertUsageEvent(t, d, usageEventParams{
		id: "agg-m1-a", keyID: "key-m1", userID: "user-m",
		modelName: "gpt-4", totalTokens: 100, promptTokens: 80, compTokens: 20,
		createdAt: now.Add(-90 * time.Minute),
	})
	insertUsageEvent(t, d, usageEventParams{
		id: "agg-m1-b", keyID: "key-m1", userID: "user-m",
		modelName: "gpt-4", totalTokens: 200, promptTokens: 160, compTokens: 40,
		createdAt: now.Add(-60 * time.Minute),
	})
	insertUsageEvent(t, d, usageEventParams{
		id: "agg-m2-a", keyID: "key-m1", userID: "user-m",
		modelName: "claude-3", totalTokens: 50, promptTokens: 40, compTokens: 10,
		createdAt: now.Add(-30 * time.Minute),
	})

	rows, err := d.GetSystemUsageAggregates(ctx, from, to, "model")
	if err != nil {
		t.Fatalf("GetSystemUsageAggregates(model) error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].GroupKey != "claude-3" || rows[1].GroupKey != "gpt-4" {
		t.Errorf("unexpected group keys: %+v", rows)
	}
}

func TestGetScopedUsageAggregates_UserFilter(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	from := now.Add(-2 * time.Hour)
	to := now.Add(time.Minute)

	insertUsageEvent(t, d, usageEventParams{
		id: "agg-user-mine", keyID: "key-user-filter", userID: "user-target",
		totalTokens: 100, promptTokens: 80, compTokens: 20,
		createdAt: now.Add(-30 * time.Minute),
	})
	insertUsageEvent(t, d, usageEventParams{
		id: "agg-user-other", keyID: "key-user-other", userID: "user-other",
		totalTokens: 9999, promptTokens: 9999, compTokens: 0,
		createdAt: now.Add(-20 * time.Minute),
	})

	rows, err := d.GetScopedUsageAggregates(ctx, UsageFilter{UserID: "user-target"}, from, to, "")
	if err != nil {
		t.Fatalf("GetScopedUsageAggregates() error = %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("GetScopedUsageAggregates() returned 0 rows, want 1")
	}
	if rows[0].TotalTokens != 100 {
		t.Errorf("TotalTokens = %d, want 100", rows[0].TotalTokens)
	}
}

func TestGetSystemUsageAggregates_InvalidGroupBy_ReturnsError(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_, err := d.GetSystemUsageAggregates(ctx, now.Add(-time.Hour), now, "invalid_column")
	if err == nil {
		t.Fatal("GetSystemUsageAggregates() with invalid groupBy error = nil, want error")
	}
}

func TestGetSystemUsageAggregates_CostEstimate_NullTreatedAsZero(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	from := now.Add(-2 * time.Hour)
	to := now.Add(time.Minute)

	insertUsageEvent(t, d, usageEventParams{
		id: "agg-cost-1", keyID: "key-cost", userID: "user-cost",
		totalTokens: 100, costEstimate: float64Ptr(0.005),
		createdAt: now.Add(-90 * time.Minute),
	})
	insertUsageEvent(t, d, usageEventParams{
		id: "agg-cost-2", keyID: "key-cost", userID: "user-cost",
		totalTokens: 200, costEstimate: nil,
		createdAt: now.Add(-60 * time.Minute),
	})
	insertUsageEvent(t, d, usageEventParams{
		id: "agg-cost-3", keyID: "key-cost", userID: "user-cost",
		totalTokens: 50, costEstimate: float64Ptr(0.002),
		createdAt: now.Add(-30 * time.Minute),
	})

	rows, err := d.GetSystemUsageAggregates(ctx, from, to, "")
	if err != nil {
		t.Fatalf("GetSystemUsageAggregates() error = %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("GetSystemUsageAggregates() returned 0 rows, want 1")
	}
	const wantCost = 0.007
	if math.Abs(rows[0].CostEstimate-wantCost) > 1e-9 {
		t.Errorf("CostEstimate = %f, want %f", rows[0].CostEstimate, wantCost)
	}
}