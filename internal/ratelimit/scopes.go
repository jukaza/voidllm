package ratelimit

// Scope prefixes isolate RPM counters from deployment IDs in UpstreamLimiter.
const (
	scopeProductModelPrefix = "pm:"
	scopeProviderPrefix     = "prov:"
)

// ScopeProductModel returns the limiter key for a customer-facing product model.
func ScopeProductModel(name string) string {
	return scopeProductModelPrefix + name
}

// ScopeProvider returns the limiter key for an upstream provider pool.
func ScopeProvider(providerID string) string {
	return scopeProviderPrefix + providerID
}