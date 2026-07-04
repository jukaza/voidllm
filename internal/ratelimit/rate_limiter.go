package ratelimit

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrRateLimitExceeded is returned by CheckRate when a rate limit is exceeded.
var ErrRateLimitExceeded = errors.New("rate limit exceeded")

type counterEntry struct {
	count       atomic.Int64
	windowStart atomic.Int64
	mu          sync.Mutex
}

var _ Checker = (*RateLimiter)(nil)

// RateLimiter enforces per-key RPM/RPD limits using in-memory atomic counters.
type RateLimiter struct {
	minuteCounters sync.Map
	dayCounters    sync.Map
}

// NewRateLimiter constructs a RateLimiter ready for use.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{}
}

// CheckRate verifies rate limits for the API key scope.
func (r *RateLimiter) CheckRate(keyID string, keyLimits Limits) error {
	now := time.Now().UTC()
	minuteWindow := now.Truncate(time.Minute).Unix()
	dayWindow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()

	scope := "key:" + keyID

	if keyLimits.RequestsPerMinute > 0 {
		entry := r.loadOrCreate(&r.minuteCounters, scope)
		if !r.tryIncrement(entry, minuteWindow, int64(keyLimits.RequestsPerMinute)) {
			return ErrRateLimitExceeded
		}
	}
	if keyLimits.RequestsPerDay > 0 {
		entry := r.loadOrCreate(&r.dayCounters, scope)
		if !r.tryIncrement(entry, dayWindow, int64(keyLimits.RequestsPerDay)) {
			return ErrRateLimitExceeded
		}
	}
	return nil
}

func (r *RateLimiter) loadOrCreate(m *sync.Map, scope string) *counterEntry {
	if v, ok := m.Load(scope); ok {
		return v.(*counterEntry)
	}
	entry := &counterEntry{}
	actual, _ := m.LoadOrStore(scope, entry)
	return actual.(*counterEntry)
}

// EvictStale removes counter entries whose window has expired.
func (r *RateLimiter) EvictStale() {
	now := time.Now().UTC()
	minuteWindow := now.Truncate(time.Minute).Unix()
	dayWindow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()

	r.minuteCounters.Range(func(key, value any) bool {
		if value.(*counterEntry).windowStart.Load() < minuteWindow {
			r.minuteCounters.Delete(key)
		}
		return true
	})
	r.dayCounters.Range(func(key, value any) bool {
		if value.(*counterEntry).windowStart.Load() < dayWindow {
			r.dayCounters.Delete(key)
		}
		return true
	})
}

func (r *RateLimiter) tryIncrement(entry *counterEntry, currentWindow int64, limit int64) bool {
	for {
		stored := entry.windowStart.Load()
		if stored == currentWindow {
			cur := entry.count.Load()
			if cur >= limit {
				return false
			}
			if entry.count.CompareAndSwap(cur, cur+1) {
				return true
			}
			continue
		}

		entry.mu.Lock()
		if entry.windowStart.Load() != currentWindow {
			entry.windowStart.Store(currentWindow)
			entry.count.Store(1)
			entry.mu.Unlock()
			return true
		}
		entry.mu.Unlock()
	}
}