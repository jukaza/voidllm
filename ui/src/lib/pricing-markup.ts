import type { RouteStepDraft } from '../components/models/ComboRouteEditor'
import type { UpstreamModelItem } from '../hooks/useUpstreamModels'
import { stepToKey } from '../components/models/upstream-model-utils'

export const DEFAULT_SELL_MARKUP = 1.5

export function inventoryByStepKey(items: UpstreamModelItem[]): Map<string, UpstreamModelItem> {
  const map = new Map<string, UpstreamModelItem>()
  for (const item of items) {
    map.set(`${item.provider_id}\0${item.upstream_id}`, item)
  }
  return map
}

/** Derive sell cached / cache-write prices from upstream inventory costs × markup. */
export function deriveCacheSellPrices(
  steps: RouteStepDraft[],
  inventory: UpstreamModelItem[],
  markup = DEFAULT_SELL_MARKUP,
): { sellCached?: number; sellCacheWrite?: number } {
  const byKey = inventoryByStepKey(inventory)
  let maxCachedIn = 0
  let maxCacheWrite = 0

  for (const step of steps) {
    const item = byKey.get(stepToKey(step))
    if (!item) continue
    if (item.cost_cached_input_per_1m != null && item.cost_cached_input_per_1m > maxCachedIn) {
      maxCachedIn = item.cost_cached_input_per_1m
    }
    if (item.cost_cache_write_per_1m != null && item.cost_cache_write_per_1m > maxCacheWrite) {
      maxCacheWrite = item.cost_cache_write_per_1m
    }
  }

  const out: { sellCached?: number; sellCacheWrite?: number } = {}
  if (maxCachedIn > 0) out.sellCached = roundPrice(maxCachedIn * markup)
  if (maxCacheWrite > 0) out.sellCacheWrite = roundPrice(maxCacheWrite * markup)
  return out
}

function roundPrice(v: number): number {
  return Math.round(v * 10000) / 10000
}