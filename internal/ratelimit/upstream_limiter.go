package ratelimit

import (
	"sync"
	"sync/atomic"
	"time"
)

// upstreamEntry holds atomic counters for one deployment within the current
// minute and day windows. The same window-rollover pattern as tokenEntry.
type upstreamEntry struct {
	mu sync.Mutex

	minuteStart    atomic.Int64
	minuteRequests atomic.Int64
	minuteTokens   atomic.Int64

	dayStart    atomic.Int64
	dayRequests atomic.Int64
}

// UpstreamLimiter tracks per-deployment (upstream channel) request and token
// throughput in memory so the router can skip channels that have hit their
// configured RPM/TPM/daily caps instead of sending requests that will be
// rejected upstream with 429.
//
// Counters are process-local. In multi-instance deployments each instance
// enforces its own share; for exact cluster-wide caps a Redis-backed
// implementation can replace this (same pattern as ratelimit.RedisChecker).
type UpstreamLimiter struct {
	entries sync.Map // deployment ID → *upstreamEntry
}

// NewUpstreamLimiter constructs an UpstreamLimiter ready for use.
func NewUpstreamLimiter() *UpstreamLimiter {
	return &UpstreamLimiter{}
}

func (u *UpstreamLimiter) entry(depID string) *upstreamEntry {
	if e, ok := u.entries.Load(depID); ok {
		return e.(*upstreamEntry)
	}
	e, _ := u.entries.LoadOrStore(depID, &upstreamEntry{})
	return e.(*upstreamEntry)
}

// rollover resets the counters when the wall clock has moved past the
// entry's window. Lock is held only during the reset.
func (e *upstreamEntry) rollover(now time.Time) {
	minuteWindow := now.Truncate(time.Minute).Unix()
	dayWindow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()

	if e.minuteStart.Load() != minuteWindow || e.dayStart.Load() != dayWindow {
		e.mu.Lock()
		if e.minuteStart.Load() != minuteWindow {
			e.minuteStart.Store(minuteWindow)
			e.minuteRequests.Store(0)
			e.minuteTokens.Store(0)
		}
		if e.dayStart.Load() != dayWindow {
			e.dayStart.Store(dayWindow)
			e.dayRequests.Store(0)
		}
		e.mu.Unlock()
	}
}

// Allow reports whether the deployment identified by depID is under all of
// its configured caps. A zero limit means unlimited for that dimension.
// Deployments with an empty ID (synthesized single-deployment candidates)
// are always allowed.
func (u *UpstreamLimiter) Allow(depID string, rpmLimit, tpmLimit, dailyRequestLimit int) bool {
	if depID == "" || (rpmLimit <= 0 && tpmLimit <= 0 && dailyRequestLimit <= 0) {
		return true
	}
	e := u.entry(depID)
	e.rollover(time.Now().UTC())

	if rpmLimit > 0 && e.minuteRequests.Load() >= int64(rpmLimit) {
		return false
	}
	if tpmLimit > 0 && e.minuteTokens.Load() >= int64(tpmLimit) {
		return false
	}
	if dailyRequestLimit > 0 && e.dayRequests.Load() >= int64(dailyRequestLimit) {
		return false
	}
	return true
}

// RecordRequest counts one request against the deployment's minute and day
// windows. Call when a request is actually sent upstream.
func (u *UpstreamLimiter) RecordRequest(depID string) {
	if depID == "" {
		return
	}
	e := u.entry(depID)
	e.rollover(time.Now().UTC())
	e.minuteRequests.Add(1)
	e.dayRequests.Add(1)
}

// RecordTokens counts tokens against the deployment's minute window. Call
// after the response's usage is known.
func (u *UpstreamLimiter) RecordTokens(depID string, tokens int64) {
	if depID == "" || tokens <= 0 {
		return
	}
	e := u.entry(depID)
	e.rollover(time.Now().UTC())
	e.minuteTokens.Add(tokens)
}

// EvictStale removes entries whose day window is older than the current day,
// reclaiming memory for deleted deployments. Call periodically (e.g. every
// 5 minutes) from a maintenance ticker.
func (u *UpstreamLimiter) EvictStale() {
	today := time.Now().UTC()
	dayWindow := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC).Unix()
	u.entries.Range(func(key, value any) bool {
		e := value.(*upstreamEntry)
		if e.dayStart.Load() < dayWindow {
			u.entries.Delete(key)
		}
		return true
	})
}
