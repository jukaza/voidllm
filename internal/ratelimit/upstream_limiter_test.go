package ratelimit

import (
	"testing"
)

func TestUpstreamLimiter_Unlimited(t *testing.T) {
	t.Parallel()
	u := NewUpstreamLimiter()

	if !u.Allow("dep-1", 0, 0, 0) {
		t.Error("Allow with all-zero limits = false, want true")
	}
	if !u.Allow("", 1, 1, 1) {
		t.Error("Allow with empty depID = false, want true")
	}
}

func TestUpstreamLimiter_RPMCap(t *testing.T) {
	t.Parallel()
	u := NewUpstreamLimiter()

	for i := 0; i < 3; i++ {
		if !u.Allow("dep-rpm", 3, 0, 0) {
			t.Fatalf("Allow #%d = false, want true", i)
		}
		u.RecordRequest("dep-rpm")
	}
	if u.Allow("dep-rpm", 3, 0, 0) {
		t.Error("Allow after 3 requests with rpm=3 = true, want false")
	}
	// A different deployment must be unaffected.
	if !u.Allow("dep-other", 3, 0, 0) {
		t.Error("Allow other deployment = false, want true")
	}
}

func TestUpstreamLimiter_TPMCap(t *testing.T) {
	t.Parallel()
	u := NewUpstreamLimiter()

	u.RecordTokens("dep-tpm", 900)
	if !u.Allow("dep-tpm", 0, 1000, 0) {
		t.Error("Allow under tpm cap = false, want true")
	}
	u.RecordTokens("dep-tpm", 200)
	if u.Allow("dep-tpm", 0, 1000, 0) {
		t.Error("Allow at 1100/1000 tokens = true, want false")
	}
}

func TestUpstreamLimiter_DailyCap(t *testing.T) {
	t.Parallel()
	u := NewUpstreamLimiter()

	u.RecordRequest("dep-daily")
	u.RecordRequest("dep-daily")
	if u.Allow("dep-daily", 0, 0, 2) {
		t.Error("Allow at 2/2 daily = true, want false")
	}
	// RPM-only limit should still pass (2 < 100).
	if !u.Allow("dep-daily", 100, 0, 0) {
		t.Error("Allow rpm=100 = false, want true")
	}
}
