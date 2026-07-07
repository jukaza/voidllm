package auth

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jukaza/tavo/internal/cache"
	"github.com/jukaza/tavo/internal/db"
)

// LoadKeysIntoCache queries all active (non-deleted) API keys from the database
// in a single JOIN query, resolves their effective RBAC role and per-key
// limits inline, and populates the key cache. Existing cache entries are
// replaced atomically via LoadAll. Rows with unparseable data are skipped with
// an error log rather than aborting the entire load.
func LoadKeysIntoCache(ctx context.Context, database *db.DB, keyCache *cache.Cache[string, KeyInfo], log *slog.Logger) error {
	records, skipErrors, err := database.LoadAllActiveKeys(ctx)
	if err != nil {
		return fmt.Errorf("load keys into cache: %w", err)
	}
	for _, skipErr := range skipErrors {
		log.LogAttrs(ctx, slog.LevelWarn, "skipped corrupt key record during cache load",
			slog.String("error", skipErr.Error()),
		)
	}

	subByKey, err := database.LoadAllActiveKeySubscriptionContexts(ctx)
	if err != nil {
		return fmt.Errorf("load key subscription contexts: %w", err)
	}

	entries := make(map[string]KeyInfo, len(records))

	for _, r := range records {
		status := r.Status
		if status == "" {
			status = "active"
		}
		ki := KeyInfo{
			ID:                 r.ID,
			KeyType:            r.KeyType,
			Name:               r.Name,
			DailyTokenLimit:    r.DailyTokenLimit,
			MonthlyTokenLimit:  r.MonthlyTokenLimit,
			RequestsPerMinute:  r.RequestsPerMinute,
			RequestsPerDay:     r.RequestsPerDay,
			ExpiresAt:          r.ExpiresAt,
			Status:             status,
			SpendCap:           r.SpendCap,
			SpendUsed:          r.SpendUsed,
			IPWhitelist:        r.IPWhitelist,
			IPBlacklist:        r.IPBlacklist,
			ModelLimitsEnabled: r.ModelLimitsEnabled,
			ModelLimits:        r.ModelLimits,
		}

		if r.UserID != nil {
			ki.UserID = *r.UserID
		}

		if r.UserRole != "" {
			ki.Role = r.UserRole
		} else {
			ki.Role = RoleMember
		}

		if subCtx, ok := subByKey[r.ID]; ok {
			ki.Subscription = &SubscriptionBinding{
				UserSubscriptionID:  subCtx.UserSubscriptionID,
				PlanID:              subCtx.PlanID,
				AllowedModels:       subCtx.AllowedModels,
				DailyTokenLimit:     subCtx.DailyTokenLimit,
				MonthlyTokenLimit:   subCtx.MonthlyTokenLimit,
				DailyRequestLimit:   subCtx.DailyRequestLimit,
				MonthlyRequestLimit: subCtx.MonthlyRequestLimit,
				RequestsPerMinute:   subCtx.RequestsPerMinute,
				RequestsPerDay:      subCtx.RequestsPerDay,
				QuotaExceededPolicy: subCtx.QuotaExceededPolicy,
				ExpiresAt:           subCtx.ExpiresAt,
			}
		}

		entries[r.KeyHash] = ki
	}

	keyCache.LoadAll(entries)

	log.LogAttrs(ctx, slog.LevelDebug, "key cache loaded",
		slog.Int("keys", len(entries)),
	)

	return nil
}

// StartCacheRefresh starts a background goroutine that reloads the key cache
// from the database every interval. Returns a stop function that blocks until
// the refresh goroutine has exited, ensuring a clean shutdown.
func StartCacheRefresh(database *db.DB, keyCache *cache.Cache[string, KeyInfo], interval time.Duration, log *slog.Logger) func() {
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := LoadKeysIntoCache(ctx, database, keyCache, log); err != nil {
					log.LogAttrs(ctx, slog.LevelError, "key cache refresh failed",
						slog.String("error", err.Error()),
					)
				}
				cancel()
			case <-done:
				return
			}
		}
	}()
	return func() {
		close(done)
		wg.Wait()
	}
}
