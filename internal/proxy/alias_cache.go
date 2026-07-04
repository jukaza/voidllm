package proxy

import "sync"

// AliasCache is a concurrency-safe global model-alias map used on the proxy hot path.
type AliasCache struct {
	mu      sync.RWMutex
	aliases map[string]string // alias → canonical model name
}

// NewAliasCache returns an empty, ready-to-use AliasCache.
func NewAliasCache() *AliasCache {
	return &AliasCache{aliases: make(map[string]string)}
}

// Resolve looks up alias and returns the canonical model name.
func (c *AliasCache) Resolve(alias string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	canonical, ok := c.aliases[alias]
	return canonical, ok
}

// Len returns the number of cached alias mappings.
func (c *AliasCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.aliases)
}

// Load atomically replaces all cached aliases.
func (c *AliasCache) Load(aliases map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliases = aliases
}