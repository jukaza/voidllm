package ratelimit

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNewRateLimiter(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter()
	if rl == nil {
		t.Fatal("NewRateLimiter() returned nil")
	}
}

func TestCheckRate_ZeroLimitsAlwaysAllowed(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter()
	unlimited := Limits{}

	for i := range 100 {
		if err := rl.CheckRate("key1", unlimited); err != nil {
			t.Fatalf("iteration %d: CheckRate() with zero limits error = %v, want nil", i, err)
		}
	}
}

func TestCheckRate_WithinRPMLimit(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter()
	keyLimits := Limits{RequestsPerMinute: 10}

	for i := range 10 {
		if err := rl.CheckRate("key-rpm-ok", keyLimits); err != nil {
			t.Fatalf("request %d: CheckRate() error = %v, want nil", i+1, err)
		}
	}
}

func TestCheckRate_ExceedingRPMLimit(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter()
	keyLimits := Limits{RequestsPerMinute: 3}

	for i := range 3 {
		if err := rl.CheckRate("key-rpm-exceed", keyLimits); err != nil {
			t.Fatalf("request %d: CheckRate() error = %v, want nil", i+1, err)
		}
	}

	err := rl.CheckRate("key-rpm-exceed", keyLimits)
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("4th CheckRate() error = %v, want ErrRateLimitExceeded", err)
	}
}

func TestCheckRate_WithinRPDLimit(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter()
	keyLimits := Limits{RequestsPerDay: 5}

	for i := range 5 {
		if err := rl.CheckRate("key-rpd-ok", keyLimits); err != nil {
			t.Fatalf("request %d: CheckRate() error = %v, want nil", i+1, err)
		}
	}
}

func TestCheckRate_ExceedingRPDLimit(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter()
	keyLimits := Limits{RequestsPerDay: 2}

	for i := range 2 {
		if err := rl.CheckRate("key-rpd-exceed", keyLimits); err != nil {
			t.Fatalf("request %d: CheckRate() error = %v, want nil", i+1, err)
		}
	}

	err := rl.CheckRate("key-rpd-exceed", keyLimits)
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("3rd CheckRate() error = %v, want ErrRateLimitExceeded", err)
	}
}

func TestCheckRate_CountersNotIncrementedOnFailure(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter()
	keyLimits := Limits{RequestsPerMinute: 2}

	_ = rl.CheckRate("key-no-incr", keyLimits)
	_ = rl.CheckRate("key-no-incr", keyLimits)

	for i := range 5 {
		err := rl.CheckRate("key-no-incr", keyLimits)
		if !errors.Is(err, ErrRateLimitExceeded) {
			t.Errorf("rejected call %d: error = %v, want ErrRateLimitExceeded", i+1, err)
		}
	}
}

func TestCheckRate_RPMAndRPDIndependent(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter()
	keyLimits := Limits{RequestsPerMinute: 3, RequestsPerDay: 10}

	for i := range 3 {
		if err := rl.CheckRate("key-both-limits", keyLimits); err != nil {
			t.Fatalf("request %d: CheckRate() error = %v, want nil", i+1, err)
		}
	}

	err := rl.CheckRate("key-both-limits", keyLimits)
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("4th CheckRate() error = %v, want ErrRateLimitExceeded (expected RPM block)", err)
	}
}

func TestCheckRate_UniqueKeysSeparateCounters(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter()
	keyLimits := Limits{RequestsPerMinute: 2}

	for i := range 2 {
		key := fmt.Sprintf("key-unique-%d", i)
		if err := rl.CheckRate(key, keyLimits); err != nil {
			t.Fatalf("key %d request 1: %v", i, err)
		}
		if err := rl.CheckRate(key, keyLimits); err != nil {
			t.Fatalf("key %d request 2: %v", i, err)
		}
	}

	err := rl.CheckRate("key-unique-0", keyLimits)
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("key-unique-0 3rd request: error = %v, want ErrRateLimitExceeded", err)
	}
	err = rl.CheckRate("key-unique-1", keyLimits)
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("key-unique-1 3rd request: error = %v, want ErrRateLimitExceeded", err)
	}
}

func TestCheckRate_ConcurrentCASCorrectness(t *testing.T) {
	t.Parallel()

	const (
		goroutines = 100
		limit      = 50
	)

	rl := NewRateLimiter()
	keyLimits := Limits{RequestsPerMinute: limit}

	var (
		wg        sync.WaitGroup
		successes atomic.Int64
		failures  atomic.Int64
	)

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			err := rl.CheckRate("cas-key", keyLimits)
			if err == nil {
				successes.Add(1)
			} else {
				failures.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := successes.Load(); got != limit {
		t.Errorf("successes = %d, want %d", got, limit)
	}
	if got := failures.Load(); got != goroutines-limit {
		t.Errorf("failures = %d, want %d", got, goroutines-limit)
	}
}

func TestEvictStale_RemovesExpiredEntries(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter()
	keyLimits := Limits{RequestsPerMinute: 10, RequestsPerDay: 10}

	for i := range 5 {
		key := fmt.Sprintf("evict-key-%d", i)
		if err := rl.CheckRate(key, keyLimits); err != nil {
			t.Fatalf("setup CheckRate for %s: %v", key, err)
		}
	}

	staleEntry := &counterEntry{}
	staleEntry.windowStart.Store(0)
	staleEntry.count.Store(3)
	rl.minuteCounters.Store("stale-minute-scope", staleEntry)
	rl.dayCounters.Store("stale-day-scope", staleEntry)

	if _, ok := rl.minuteCounters.Load("stale-minute-scope"); !ok {
		t.Fatal("stale minute entry not present before EvictStale")
	}
	if _, ok := rl.dayCounters.Load("stale-day-scope"); !ok {
		t.Fatal("stale day entry not present before EvictStale")
	}

	rl.EvictStale()

	if _, ok := rl.minuteCounters.Load("stale-minute-scope"); ok {
		t.Error("stale minute entry still present after EvictStale")
	}
	if _, ok := rl.dayCounters.Load("stale-day-scope"); ok {
		t.Error("stale day entry still present after EvictStale")
	}

	for i := range 5 {
		key := fmt.Sprintf("evict-key-%d", i)
		if _, ok := rl.minuteCounters.Load("key:" + key); !ok {
			t.Errorf("current-window minute entry for %s was wrongly evicted", key)
		}
	}
}

func TestCheckRate_KeyLimitDoesNotConsumeOtherKeys(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter()
	keyLimits := Limits{RequestsPerMinute: 3}

	for i := range 3 {
		if err := rl.CheckRate("key-isolation-a", keyLimits); err != nil {
			t.Fatalf("key A request %d: CheckRate() error = %v, want nil", i+1, err)
		}
	}
	err := rl.CheckRate("key-isolation-a", keyLimits)
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("key A 4th CheckRate() error = %v, want ErrRateLimitExceeded", err)
	}

	for i := range 3 {
		if err := rl.CheckRate("key-isolation-b", keyLimits); err != nil {
			t.Fatalf("key B request %d: CheckRate() error = %v, want nil", i+1, err)
		}
	}
}

func TestRateLimiter_Snapshot(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter()
	keyLimits := Limits{RequestsPerMinute: 10, RequestsPerDay: 20}

	if err := rl.CheckRate("snap-key", keyLimits); err != nil {
		t.Fatalf("setup CheckRate: %v", err)
	}
	if err := rl.CheckRate("snap-key", keyLimits); err != nil {
		t.Fatalf("setup CheckRate: %v", err)
	}

	snap := rl.Snapshot("snap-key")
	if snap.RequestsPerMinute != 2 {
		t.Errorf("RequestsPerMinute = %d, want 2", snap.RequestsPerMinute)
	}
	if snap.RequestsPerDay != 2 {
		t.Errorf("RequestsPerDay = %d, want 2", snap.RequestsPerDay)
	}
}

func BenchmarkCheckRate(b *testing.B) {
	rl := NewRateLimiter()
	keyLimits := Limits{RequestsPerMinute: 1_000_000}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := rl.CheckRate("bench-key", keyLimits); err != nil {
				b.Fatal(err)
			}
		}
	})
}