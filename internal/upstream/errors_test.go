package upstream

import (
	"testing"
	"time"
)

func TestClassifyError_429Backoff(t *testing.T) {
	r := ClassifyError(429, "rate limit exceeded", 0)
	if !r.ShouldRotate || r.Cooldown < time.Second {
		t.Fatalf("got %+v", r)
	}
	if r.NewBackoffLevel != 1 {
		t.Fatalf("backoff level = %d", r.NewBackoffLevel)
	}
}

func TestClassifyError_401LongCooldown(t *testing.T) {
	r := ClassifyError(401, "", 0)
	if !r.ShouldRotate || r.Cooldown != 2*time.Minute {
		t.Fatalf("got %+v", r)
	}
}

func TestIsModelLockActive(t *testing.T) {
	now := time.Now().UTC()
	locks := map[string]string{
		"gpt-4o": now.Add(2 * time.Minute).Format(time.RFC3339),
	}
	if !IsModelLockActive(locks, "gpt-4o", now) {
		t.Fatal("expected active lock")
	}
	if IsModelLockActive(locks, "gpt-4o-mini", now) {
		t.Fatal("unexpected lock on other model")
	}
}