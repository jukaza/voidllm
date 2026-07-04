package usage

import (
	"sync"
	"time"
)

const (
	liveRecentCap     = 10
	liveWindow        = 60 * time.Second
	liveActiveTimeout = 45 * time.Second
)

// LiveEvent is a lightweight completion record for the live stats dashboard.
type LiveEvent struct {
	RequestID        string
	ModelName        string
	Provider         string
	DeploymentID     string
	PromptTokens     int
	CompletionTokens int
	StatusCode       int
	CompletedAt      time.Time
}

// ActiveRequestGroup aggregates in-flight requests by model and provider.
type ActiveRequestGroup struct {
	Model      string `json:"model"`
	Provider   string `json:"provider"`
	Deployment string `json:"deployment,omitempty"`
	Count      int    `json:"count"`
}

// RecentRequestSnapshot is one row in the live recent-requests list.
type RecentRequestSnapshot struct {
	Timestamp        string `json:"timestamp"`
	Model            string `json:"model"`
	Provider         string `json:"provider"`
	DeploymentID     string `json:"deployment_id,omitempty"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	StatusCode       int    `json:"status_code"`
}

// LiveSnapshot is the SSE/REST payload for volatile usage stats.
type LiveSnapshot struct {
	RPM            int                     `json:"rpm"`
	TPM            int                     `json:"tpm"`
	ActiveCount    int                     `json:"active_count"`
	ActiveRequests []ActiveRequestGroup    `json:"active_requests"`
	RecentRequests []RecentRequestSnapshot `json:"recent_requests"`
}

type activeEntry struct {
	model      string
	provider   string
	deployment string
	started    time.Time
}

type completionSample struct {
	at     time.Time
	tokens int
}

// LiveStats tracks in-flight and recently completed proxy requests in memory.
type LiveStats struct {
	mu sync.RWMutex

	active      map[string]*activeEntry
	activeTimer map[string]*time.Timer
	recent      []RecentRequestSnapshot
	completions []completionSample
}

// NewLiveStats constructs an empty LiveStats tracker.
func NewLiveStats() *LiveStats {
	return &LiveStats{
		active:      make(map[string]*activeEntry),
		activeTimer: make(map[string]*time.Timer),
	}
}

// Begin marks a request as in-flight. Duplicate IDs are ignored.
func (l *LiveStats) Begin(requestID, model, provider, deployment string) {
	if l == nil || requestID == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.active[requestID]; exists {
		return
	}
	l.active[requestID] = &activeEntry{
		model:      model,
		provider:   provider,
		deployment: deployment,
		started:    time.Now().UTC(),
	}
	if old, ok := l.activeTimer[requestID]; ok {
		old.Stop()
	}
	l.activeTimer[requestID] = time.AfterFunc(liveActiveTimeout, func() {
		l.End(requestID)
	})
}

// End removes a request from the in-flight set.
func (l *LiveStats) End(requestID string) {
	if l == nil || requestID == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.active, requestID)
	if t, ok := l.activeTimer[requestID]; ok {
		t.Stop()
		delete(l.activeTimer, requestID)
	}
}

// RecordComplete appends a completion sample and updates recent requests.
func (l *LiveStats) RecordComplete(ev LiveEvent) {
	if l == nil {
		return
	}
	if ev.CompletedAt.IsZero() {
		ev.CompletedAt = time.Now().UTC()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.EndLocked(ev.RequestID)

	totalTokens := ev.PromptTokens + ev.CompletionTokens
	l.completions = append(l.completions, completionSample{
		at:     ev.CompletedAt,
		tokens: totalTokens,
	})
	cutoff := ev.CompletedAt.Add(-liveWindow)
	pruned := l.completions[:0]
	for _, s := range l.completions {
		if s.at.After(cutoff) {
			pruned = append(pruned, s)
		}
	}
	l.completions = pruned

	row := RecentRequestSnapshot{
		Timestamp:        ev.CompletedAt.UTC().Format(time.RFC3339),
		Model:            ev.ModelName,
		Provider:         ev.Provider,
		DeploymentID:     ev.DeploymentID,
		PromptTokens:     ev.PromptTokens,
		CompletionTokens: ev.CompletionTokens,
		StatusCode:       ev.StatusCode,
	}
	l.recent = append([]RecentRequestSnapshot{row}, l.recent...)
	if len(l.recent) > liveRecentCap {
		l.recent = l.recent[:liveRecentCap]
	}
}

// Snapshot returns the current live stats for dashboards.
func (l *LiveStats) Snapshot() LiveSnapshot {
	if l == nil {
		return LiveSnapshot{}
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	now := time.Now().UTC()
	cutoff := now.Add(-liveWindow)

	requests := 0
	tokens := 0
	pruned := l.completions[:0]
	for _, s := range l.completions {
		if s.at.After(cutoff) {
			pruned = append(pruned, s)
			requests++
			tokens += s.tokens
		}
	}

	type groupKey struct {
		model, provider, deployment string
	}
	groups := make(map[groupKey]int)
	for _, e := range l.active {
		k := groupKey{model: e.model, provider: e.provider, deployment: e.deployment}
		groups[k]++
	}
	active := make([]ActiveRequestGroup, 0, len(groups))
	for k, count := range groups {
		active = append(active, ActiveRequestGroup{
			Model:      k.model,
			Provider:   k.provider,
			Deployment: k.deployment,
			Count:      count,
		})
	}

	recent := make([]RecentRequestSnapshot, len(l.recent))
	copy(recent, l.recent)

	return LiveSnapshot{
		RPM:            requests,
		TPM:            tokens,
		ActiveCount:    len(l.active),
		ActiveRequests: active,
		RecentRequests: recent,
	}
}

func (l *LiveStats) EndLocked(requestID string) {
	if requestID == "" {
		return
	}
	delete(l.active, requestID)
	if t, ok := l.activeTimer[requestID]; ok {
		t.Stop()
		delete(l.activeTimer, requestID)
	}
}