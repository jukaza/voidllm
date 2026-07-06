package keys

import "sync"

// Runtime holds hot-reloaded API key policy settings.
type Runtime struct {
	mu     sync.RWMutex
	policy Policy
}

// NewRuntime constructs a Runtime with default policy.
func NewRuntime() *Runtime {
	return &Runtime{policy: DefaultPolicy()}
}

// Get returns the current policy snapshot.
func (r *Runtime) Get() Policy {
	if r == nil {
		return DefaultPolicy()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.policy
}

// Set replaces the current policy.
func (r *Runtime) Set(p Policy) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.policy = p
	r.mu.Unlock()
}