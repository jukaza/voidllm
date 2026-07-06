package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// UsageSeeder provides seed data for token counters at startup.
type UsageSeeder interface {
	// QueryUsageSeed returns rows of (key_id, total_tokens) since the given time.
	QueryUsageSeed(ctx context.Context, since time.Time) (RowScanner, error)
}

// RowScanner iterates over database rows. *sql.Rows satisfies this interface.
type RowScanner interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

// ErrTokenBudgetExceeded is returned by CheckTokens when a token budget is
// exceeded.
var ErrTokenBudgetExceeded = errors.New("token budget exceeded")

// tokenEntry holds an atomic token counter and the Unix timestamp of the start
// of the window it belongs to (start of day or start of month). mu is held
// only during window rollover; the common path (same window) is lock-free.
type tokenEntry struct {
	tokens      atomic.Int64
	windowStart atomic.Int64
	mu          sync.Mutex
}

// TokenCounter tracks per-scope token usage in memory using atomic counters.
// Counters are incremented immediately when usage events are logged and checked
// before each proxy request to enforce token budgets. The zero value is not
// usable; construct via NewTokenCounter.
type TokenCounter struct {
	dailyCounters   sync.Map // scope string → *tokenEntry
	monthlyCounters sync.Map // scope string → *tokenEntry
}

// NewTokenCounter constructs a TokenCounter ready for use.
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{}
}

// Add increments token counters for all applicable scopes. It must be called
// immediately when a usage event is recorded so that subsequent CheckTokens
// calls see the up-to-date totals.
func (tc *TokenCounter) Add(keyID string, tokens int64) {
	now := time.Now().UTC()
	dayWindow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()
	monthWindow := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Unix()

	tc.addToScope(&tc.dailyCounters, "key:"+keyID, dayWindow, tokens)
	tc.addToScope(&tc.monthlyCounters, "key:"+keyID, monthWindow, tokens)
}

// TokenUsageSnapshot holds in-memory token totals for a key scope.
type TokenUsageSnapshot struct {
	DailyTokens   int64 `json:"daily_tokens"`
	MonthlyTokens int64 `json:"monthly_tokens"`
}

// Snapshot returns the current daily/monthly token usage for keyID without
// mutating counters.
func (tc *TokenCounter) Snapshot(keyID string) TokenUsageSnapshot {
	now := time.Now().UTC()
	dayWindow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()
	monthWindow := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Unix()
	scope := "key:" + keyID
	return TokenUsageSnapshot{
		DailyTokens:   tc.getCount(&tc.dailyCounters, scope, dayWindow),
		MonthlyTokens: tc.getCount(&tc.monthlyCounters, scope, monthWindow),
	}
}

// CheckTokens verifies that no scope has exceeded its token budget. Each scope
// is checked against its own limit independently. Returns ErrTokenBudgetExceeded
// if any scope is over budget.
func (tc *TokenCounter) CheckTokens(keyID string, keyLimits Limits) error {
	return tc.CheckScopedTokens("key:"+keyID, keyLimits)
}

// CheckScopedTokens verifies token budgets for an arbitrary counter scope.
func (tc *TokenCounter) CheckScopedTokens(scope string, limits Limits) error {
	now := time.Now().UTC()
	dayWindow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()
	monthWindow := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Unix()

	if limits.DailyTokenLimit > 0 {
		if tc.getCount(&tc.dailyCounters, scope, dayWindow) >= limits.DailyTokenLimit {
			return ErrTokenBudgetExceeded
		}
	}
	if limits.MonthlyTokenLimit > 0 {
		if tc.getCount(&tc.monthlyCounters, scope, monthWindow) >= limits.MonthlyTokenLimit {
			return ErrTokenBudgetExceeded
		}
	}
	return nil
}

// AddScoped increments token counters for an arbitrary scope.
func (tc *TokenCounter) AddScoped(scope string, tokens int64) {
	if tokens <= 0 {
		return
	}
	now := time.Now().UTC()
	dayWindow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()
	monthWindow := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Unix()
	tc.addToScope(&tc.dailyCounters, scope, dayWindow, tokens)
	tc.addToScope(&tc.monthlyCounters, scope, monthWindow, tokens)
}

// Seed loads token usage totals from the database into the in-memory counters.
// It should be called once at startup before the proxy begins serving requests.
// Seed reads all usage_events rows from the current day and current month, so
// the in-memory state matches what is already persisted.
func (tc *TokenCounter) Seed(ctx context.Context, seeder UsageSeeder) error {
	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	if err := tc.seedWindow(ctx, seeder, &tc.dailyCounters, dayStart); err != nil {
		return fmt.Errorf("seed daily counters: %w", err)
	}
	if err := tc.seedWindow(ctx, seeder, &tc.monthlyCounters, monthStart); err != nil {
		return fmt.Errorf("seed monthly counters: %w", err)
	}
	return nil
}

// EvictStale removes token counter entries from expired windows to reclaim
// memory. It should be called periodically (e.g. every hour).
func (tc *TokenCounter) EvictStale() {
	now := time.Now().UTC()
	dayWindow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()
	monthWindow := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Unix()

	tc.dailyCounters.Range(func(key, value any) bool {
		if value.(*tokenEntry).windowStart.Load() < dayWindow {
			tc.dailyCounters.Delete(key)
		}
		return true
	})
	tc.monthlyCounters.Range(func(key, value any) bool {
		if value.(*tokenEntry).windowStart.Load() < monthWindow {
			tc.monthlyCounters.Delete(key)
		}
		return true
	})
}

// addToScope atomically adds delta tokens to the counter for scope in m. If
// the stored window differs from windowStart the counter is reset to delta
// (window rollover).
//
// The common path (same window) uses atomic.Add and is lock-free. The rollover
// path takes entry.mu so that windowStart and the reset token count are
// updated as a single visible transition. Without the lock, a goroutine that
// wins the windowStart CAS can have its tokens.Store(delta) overwritten by a
// second goroutine that races in before the Store and increments the stale
// count — producing an incorrect aggregate.
func (tc *TokenCounter) addToScope(m *sync.Map, scope string, windowStart int64, delta int64) {
	entry := tc.loadOrCreateToken(m, scope)
	for {
		stored := entry.windowStart.Load()
		if stored == windowStart {
			// Same window — lock-free add.
			entry.tokens.Add(delta)
			return
		}

		// Window has rolled over — take the lock so the reset is atomic.
		entry.mu.Lock()
		if entry.windowStart.Load() != windowStart {
			// We are the first inside the lock to handle this rollover.
			entry.windowStart.Store(windowStart)
			entry.tokens.Store(delta)
			entry.mu.Unlock()
			return
		}
		// Another goroutine already rolled the window; fall through to the
		// same-window Add path.
		entry.mu.Unlock()
	}
}

// getCount returns the current token count for scope in m, or 0 if the stored
// window does not match currentWindow (i.e. the entry is from a prior window).
func (tc *TokenCounter) getCount(m *sync.Map, scope string, currentWindow int64) int64 {
	v, ok := m.Load(scope)
	if !ok {
		return 0
	}
	entry := v.(*tokenEntry)
	if entry.windowStart.Load() != currentWindow {
		return 0
	}
	return entry.tokens.Load()
}

// loadOrCreateToken retrieves an existing tokenEntry for the given scope, or
// stores and returns a newly allocated one.
func (tc *TokenCounter) loadOrCreateToken(m *sync.Map, scope string) *tokenEntry {
	if v, ok := m.Load(scope); ok {
		return v.(*tokenEntry)
	}
	entry := &tokenEntry{}
	actual, _ := m.LoadOrStore(scope, entry)
	return actual.(*tokenEntry)
}

// seedWindow reads all usage_events rows since the given time and accumulates
// token totals into counters. The windowStart for all seeded entries is set to
// since.Unix().
func (tc *TokenCounter) seedWindow(ctx context.Context, seeder UsageSeeder, counters *sync.Map, since time.Time) error {
	rows, err := seeder.QueryUsageSeed(ctx, since)
	if err != nil {
		return fmt.Errorf("query usage_events: %w", err)
	}
	defer rows.Close()

	windowStart := since.Unix()
	for rows.Next() {
		var keyID string
		var tokens int64
		if err := rows.Scan(&keyID, &tokens); err != nil {
			return fmt.Errorf("scan usage_events row: %w", err)
		}
		tc.addToScope(counters, "key:"+keyID, windowStart, tokens)
	}
	return rows.Err()
}
