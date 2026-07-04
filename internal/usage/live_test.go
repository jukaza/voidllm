package usage

import (
	"testing"
	"time"
)

func TestLiveStats_BeginEndAndSnapshot(t *testing.T) {
	t.Parallel()

	l := NewLiveStats()
	l.Begin("req-1", "gpt-4o", "openai", "gpt-4o/key-1")
	snap := l.Snapshot()
	if snap.ActiveCount != 1 {
		t.Fatalf("ActiveCount = %d, want 1", snap.ActiveCount)
	}

	l.RecordComplete(LiveEvent{
		RequestID:        "req-1",
		ModelName:        "gpt-4o",
		Provider:         "openai",
		DeploymentID:     "gpt-4o/key-1",
		PromptTokens:     100,
		CompletionTokens: 50,
		StatusCode:       200,
		CompletedAt:      time.Now().UTC(),
	})
	snap = l.Snapshot()
	if snap.ActiveCount != 0 {
		t.Fatalf("ActiveCount after complete = %d, want 0", snap.ActiveCount)
	}
	if snap.RPM != 1 {
		t.Fatalf("RPM = %d, want 1", snap.RPM)
	}
	if snap.TPM != 150 {
		t.Fatalf("TPM = %d, want 150", snap.TPM)
	}
	if len(snap.RecentRequests) != 1 {
		t.Fatalf("RecentRequests len = %d, want 1", len(snap.RecentRequests))
	}
}