package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// DailyBucketDim holds per-dimension counters inside usage_daily JSON blobs.
type DailyBucketDim struct {
	Requests          int     `json:"requests"`
	PromptTokens      int     `json:"prompt_tokens"`
	CompletionTokens  int     `json:"completion_tokens"`
	CachedTokens      int     `json:"cached_tokens"`
	CostSum           float64 `json:"cost_sum"`
	RevenueSum        float64 `json:"revenue_sum"`
}

// DailyRollupDelta is one increment applied to a (day, user_id) bucket.
type DailyRollupDelta struct {
	Day              string
	UserID           string // empty = system-wide
	Requests         int
	PromptTokens     int
	CompletionTokens int
	CachedTokens     int
	CostSum          float64
	RevenueSum       float64
	ModelName        string
	DeploymentID     string
}

type dailyJSONMaps struct {
	ByModel      map[string]DailyBucketDim `json:"by_model"`
	ByDeployment map[string]DailyBucketDim `json:"by_deployment"`
}

func mergeDim(m map[string]DailyBucketDim, key string, d DailyRollupDelta) {
	if key == "" {
		return
	}
	cur := m[key]
	cur.Requests += d.Requests
	cur.PromptTokens += d.PromptTokens
	cur.CompletionTokens += d.CompletionTokens
	cur.CachedTokens += d.CachedTokens
	cur.CostSum += d.CostSum
	cur.RevenueSum += d.RevenueSum
	m[key] = cur
}

// UpsertUsageDaily applies deltas to usage_daily buckets inside a transaction.
func (d *DB) UpsertUsageDaily(ctx context.Context, q Querier, deltas []DailyRollupDelta) error {
	if len(deltas) == 0 {
		return nil
	}

	merged := make(map[string]*dailyJSONMaps)
	totals := make(map[string]DailyRollupDelta)

	for _, delta := range deltas {
		key := delta.Day + "\x00" + delta.UserID
		t := totals[key]
		t.Day = delta.Day
		t.UserID = delta.UserID
		t.Requests += delta.Requests
		t.PromptTokens += delta.PromptTokens
		t.CompletionTokens += delta.CompletionTokens
		t.CachedTokens += delta.CachedTokens
		t.CostSum += delta.CostSum
		t.RevenueSum += delta.RevenueSum
		totals[key] = t

		m, ok := merged[key]
		if !ok {
			m = &dailyJSONMaps{
				ByModel:      map[string]DailyBucketDim{},
				ByDeployment: map[string]DailyBucketDim{},
			}
			merged[key] = m
		}
		mergeDim(m.ByModel, delta.ModelName, delta)
		mergeDim(m.ByDeployment, delta.DeploymentID, delta)
	}

	p := d.dialect.Placeholder
	selectQuery := "SELECT by_model, by_deployment FROM usage_daily WHERE day = " + p(1) + " AND user_id = " + p(2)
	insertQuery := "INSERT INTO usage_daily (day, user_id, requests, prompt_tokens, completion_tokens, cached_tokens, cost_sum, revenue_sum, by_model, by_deployment) " +
		"VALUES (" + p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", " + p(5) + ", " + p(6) + ", " + p(7) + ", " + p(8) + ", " + p(9) + ", " + p(10) + ")"
	updateQuery := "UPDATE usage_daily SET requests = requests + " + p(1) +
		", prompt_tokens = prompt_tokens + " + p(2) +
		", completion_tokens = completion_tokens + " + p(3) +
		", cached_tokens = cached_tokens + " + p(4) +
		", cost_sum = cost_sum + " + p(5) +
		", revenue_sum = revenue_sum + " + p(6) +
		", by_model = " + p(7) +
		", by_deployment = " + p(8) +
		" WHERE day = " + p(9) + " AND user_id = " + p(10)

	for key, total := range totals {
		maps := merged[key]
		byModelJSON, err := json.Marshal(maps.ByModel)
		if err != nil {
			return fmt.Errorf("marshal by_model: %w", err)
		}
		byDepJSON, err := json.Marshal(maps.ByDeployment)
		if err != nil {
			return fmt.Errorf("marshal by_deployment: %w", err)
		}

		var existingModel, existingDep string
		err = q.QueryRowContext(ctx, selectQuery, total.Day, total.UserID).Scan(&existingModel, &existingDep)

		switch {
		case err == nil:
			var existing dailyJSONMaps
			_ = json.Unmarshal([]byte(existingModel), &existing.ByModel)
			_ = json.Unmarshal([]byte(existingDep), &existing.ByDeployment)
			if existing.ByModel == nil {
				existing.ByModel = map[string]DailyBucketDim{}
			}
			if existing.ByDeployment == nil {
				existing.ByDeployment = map[string]DailyBucketDim{}
			}
			for k, v := range maps.ByModel {
				cur := existing.ByModel[k]
				cur.Requests += v.Requests
				cur.PromptTokens += v.PromptTokens
				cur.CompletionTokens += v.CompletionTokens
				cur.CachedTokens += v.CachedTokens
				cur.CostSum += v.CostSum
				cur.RevenueSum += v.RevenueSum
				existing.ByModel[k] = cur
			}
			for k, v := range maps.ByDeployment {
				cur := existing.ByDeployment[k]
				cur.Requests += v.Requests
				cur.PromptTokens += v.PromptTokens
				cur.CompletionTokens += v.CompletionTokens
				cur.CachedTokens += v.CachedTokens
				cur.CostSum += v.CostSum
				cur.RevenueSum += v.RevenueSum
				existing.ByDeployment[k] = cur
			}
			byModelJSON, _ = json.Marshal(existing.ByModel)
			byDepJSON, _ = json.Marshal(existing.ByDeployment)
			_, err = q.ExecContext(ctx, updateQuery,
				total.Requests, total.PromptTokens, total.CompletionTokens, total.CachedTokens,
				total.CostSum, total.RevenueSum, string(byModelJSON), string(byDepJSON),
				total.Day, total.UserID,
			)
		case errors.Is(err, sql.ErrNoRows):
			_, err = q.ExecContext(ctx, insertQuery,
				total.Day, total.UserID,
				total.Requests, total.PromptTokens, total.CompletionTokens, total.CachedTokens,
				total.CostSum, total.RevenueSum, string(byModelJSON), string(byDepJSON),
			)
		default:
			return fmt.Errorf("select usage_daily day=%s user=%s: %w", total.Day, total.UserID, err)
		}
		if err != nil {
			return fmt.Errorf("upsert usage_daily day=%s user=%s: %w", total.Day, total.UserID, translateError(err))
		}
	}
	return nil
}