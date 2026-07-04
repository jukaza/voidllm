package ratelimit

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewTokenCounter(t *testing.T) {
	t.Parallel()

	tc := NewTokenCounter()
	if tc == nil {
		t.Fatal("NewTokenCounter() returned nil")
	}
}

func TestTokenCounter_AddAndCheckTokens(t *testing.T) {
	t.Parallel()

	tc := NewTokenCounter()
	tc.Add("key1", 500)

	keyLimits := Limits{DailyTokenLimit: 1000}

	if err := tc.CheckTokens("key1", keyLimits); err != nil {
		t.Errorf("CheckTokens() = %v, want nil (500 tokens < 1000 limit)", err)
	}
}

func TestTokenCounter_CheckTokens_ExceedsKeyDailyLimit(t *testing.T) {
	t.Parallel()

	tc := NewTokenCounter()
	tc.Add("key-exceed-daily", 1000)

	keyLimits := Limits{DailyTokenLimit: 1000}

	err := tc.CheckTokens("key-exceed-daily", keyLimits)
	if !errors.Is(err, ErrTokenBudgetExceeded) {
		t.Errorf("CheckTokens() = %v, want ErrTokenBudgetExceeded", err)
	}
}

func TestTokenCounter_CheckTokens_ExceedsKeyMonthlyLimit(t *testing.T) {
	t.Parallel()

	tc := NewTokenCounter()
	tc.Add("key-exceed-monthly", 5000)

	keyLimits := Limits{MonthlyTokenLimit: 5000}

	err := tc.CheckTokens("key-exceed-monthly", keyLimits)
	if !errors.Is(err, ErrTokenBudgetExceeded) {
		t.Errorf("CheckTokens() = %v, want ErrTokenBudgetExceeded", err)
	}
}

func TestTokenCounter_ZeroLimit_Unlimited(t *testing.T) {
	t.Parallel()

	tc := NewTokenCounter()
	tc.Add("key-unlimited", 1_000_000)

	if err := tc.CheckTokens("key-unlimited", Limits{}); err != nil {
		t.Errorf("CheckTokens() = %v, want nil (all limits zero = unlimited)", err)
	}
}

func TestTokenCounter_DailyWindowReset(t *testing.T) {
	t.Parallel()

	tc := NewTokenCounter()

	now := time.Now().UTC()
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC).Unix()

	stale := &tokenEntry{}
	stale.windowStart.Store(yesterday)
	stale.tokens.Store(900)
	tc.dailyCounters.Store("key:key-stale-daily", stale)

	keyLimits := Limits{DailyTokenLimit: 1000}

	if err := tc.CheckTokens("key-stale-daily", keyLimits); err != nil {
		t.Errorf("CheckTokens() = %v, want nil (stale window should read as 0 tokens)", err)
	}
}

func TestTokenCounter_MonthlyWindowReset(t *testing.T) {
	t.Parallel()

	tc := NewTokenCounter()

	now := time.Now().UTC()
	lastMonth := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC).Unix()

	stale := &tokenEntry{}
	stale.windowStart.Store(lastMonth)
	stale.tokens.Store(4500)
	tc.monthlyCounters.Store("key:key-stale-monthly", stale)

	keyLimits := Limits{MonthlyTokenLimit: 5000}

	if err := tc.CheckTokens("key-stale-monthly", keyLimits); err != nil {
		t.Errorf("CheckTokens() = %v, want nil (stale monthly window should read as 0 tokens)", err)
	}
}

func TestTokenCounter_AddResetsOnWindowRollover(t *testing.T) {
	t.Parallel()

	tc := NewTokenCounter()

	now := time.Now().UTC()
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC).Unix()
	stale := &tokenEntry{}
	stale.windowStart.Store(yesterday)
	stale.tokens.Store(9999)
	tc.dailyCounters.Store("key:key-rollover", stale)

	tc.Add("key-rollover", 100)

	keyLimits := Limits{DailyTokenLimit: 500}

	if err := tc.CheckTokens("key-rollover", keyLimits); err != nil {
		t.Errorf("CheckTokens() = %v, want nil (100 tokens after rollover, limit 500)", err)
	}
}

func TestTokenCounter_EvictStale(t *testing.T) {
	t.Parallel()

	tc := NewTokenCounter()

	now := time.Now().UTC()
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC).Unix()
	lastMonth := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC).Unix()

	staleDaily := &tokenEntry{}
	staleDaily.windowStart.Store(yesterday)
	staleDaily.tokens.Store(100)
	tc.dailyCounters.Store("key:stale-daily-key", staleDaily)

	staleMonthly := &tokenEntry{}
	staleMonthly.windowStart.Store(lastMonth)
	staleMonthly.tokens.Store(200)
	tc.monthlyCounters.Store("key:stale-monthly-key", staleMonthly)

	tc.Add("key-current", 50)
	tc.EvictStale()

	if _, ok := tc.dailyCounters.Load("key:stale-daily-key"); ok {
		t.Error("stale daily entry still present after EvictStale")
	}
	if _, ok := tc.monthlyCounters.Load("key:stale-monthly-key"); ok {
		t.Error("stale monthly entry still present after EvictStale")
	}
	if _, ok := tc.dailyCounters.Load("key:key-current"); !ok {
		t.Error("current daily entry was wrongly evicted")
	}
	if _, ok := tc.monthlyCounters.Load("key:key-current"); !ok {
		t.Error("current monthly entry was wrongly evicted")
	}
}

func TestTokenCounter_UniqueKeysSeparateCounters(t *testing.T) {
	t.Parallel()

	tc := NewTokenCounter()
	tc.Add("key-a", 400)
	tc.Add("key-b", 100)

	keyLimits := Limits{DailyTokenLimit: 300}

	if err := tc.CheckTokens("key-b", keyLimits); err != nil {
		t.Errorf("CheckTokens(key-b) = %v, want nil", err)
	}
	if err := tc.CheckTokens("key-a", keyLimits); !errors.Is(err, ErrTokenBudgetExceeded) {
		t.Errorf("CheckTokens(key-a) = %v, want ErrTokenBudgetExceeded", err)
	}
}

func TestTokenCounter_ConcurrentAdd(t *testing.T) {
	t.Parallel()

	const (
		goroutines = 100
		tokensEach = 10
		wantTotal  = goroutines * tokensEach
	)

	tc := NewTokenCounter()

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			tc.Add("concurrent-key", tokensEach)
		}()
	}
	wg.Wait()

	now := time.Now().UTC()
	dayWindow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()

	got := tc.getCount(&tc.dailyCounters, "key:concurrent-key", dayWindow)
	if got != wantTotal {
		t.Errorf("concurrent Add total = %d, want %d", got, wantTotal)
	}
}

func TestTokenCounter_ConcurrentAddAndCheck(t *testing.T) {
	t.Parallel()

	tc := NewTokenCounter()
	keyLimits := Limits{DailyTokenLimit: 100_000}

	var wg sync.WaitGroup
	var checkErrors atomic.Int64

	for i := range 50 {
		wg.Add(2)
		key := fmt.Sprintf("race-key-%d", i%5)
		go func() {
			defer wg.Done()
			tc.Add(key, 10)
		}()
		go func() {
			defer wg.Done()
			if err := tc.CheckTokens(key, keyLimits); err != nil {
				checkErrors.Add(1)
			}
		}()
	}
	wg.Wait()

	if n := checkErrors.Load(); n > 0 {
		t.Errorf("%d CheckTokens calls returned unexpected errors under low load", n)
	}
}

func BenchmarkTokenCounter_Add(b *testing.B) {
	tc := NewTokenCounter()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tc.Add("bench-key", 100)
		}
	})
}

func BenchmarkTokenCounter_CheckTokens(b *testing.B) {
	tc := NewTokenCounter()
	tc.Add("bench-key", 100)
	keyLimits := Limits{DailyTokenLimit: 1_000_000, MonthlyTokenLimit: 10_000_000}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := tc.CheckTokens("bench-key", keyLimits); err != nil {
				b.Fatal(err)
			}
		}
	})
}