package provider

// OptionalCostFields splits a CostRef into optional per-field pointers suitable
// for nullable DB columns. Zero-valued cache fields are omitted.
func OptionalCostFields(cost *CostRef) (in, out, cachedIn, cacheWrite *float64) {
	if cost == nil {
		return nil, nil, nil, nil
	}
	in, out = &cost.In, &cost.Out
	if cost.CachedIn > 0 {
		cachedIn = &cost.CachedIn
	}
	if cost.CacheWrite > 0 {
		cacheWrite = &cost.CacheWrite
	}
	return in, out, cachedIn, cacheWrite
}