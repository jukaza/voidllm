import { describe, expect, it } from 'vitest'
import type { CatalogModelItem } from '../hooks/useProviders'
import {
  anyModelHasCache,
  applyCatalogFilters,
  detectFamily,
  filterChatModels,
  formatContextTokens,
  hasCachePrice,
  formatCatalogLatency,
  formatCatalogRequestCount,
  inferCapabilities,
  isChatModel,
  sortCatalogModels,
} from './catalog-utils'

function model(partial: Partial<CatalogModelItem> & Pick<CatalogModelItem, 'name'>): CatalogModelItem {
  return {
    type: 'chat',
    bill_per_token: true,
    bill_per_request: false,
    ...partial,
  }
}

describe('catalog-utils', () => {
  it('isChatModel accepts chat and completion', () => {
    expect(isChatModel('chat')).toBe(true)
    expect(isChatModel('completion')).toBe(true)
    expect(isChatModel('image')).toBe(false)
  })

  it('hasCachePrice requires positive sell_cached_input_per_1m', () => {
    expect(hasCachePrice(model({ name: 'a', sell_cached_input_per_1m: 500 }))).toBe(true)
    expect(hasCachePrice(model({ name: 'b', sell_cached_input_per_1m: 0 }))).toBe(false)
    expect(hasCachePrice(model({ name: 'c' }))).toBe(false)
  })

  it('detectFamily maps model name prefixes', () => {
    expect(detectFamily('claude-sonnet-4')).toBe('claude')
    expect(detectFamily('gpt-4o')).toBe('gpt')
    expect(detectFamily('gemini-2.5-pro')).toBe('gemini')
    expect(detectFamily('glm-4-flash')).toBe('glm')
    expect(detectFamily('kimi-k2')).toBe('kimi')
    expect(detectFamily('minimax-m2')).toBe('minimax')
    expect(detectFamily('unknown-model')).toBe('other')
  })

  it('formatContextTokens abbreviates large values', () => {
    expect(formatContextTokens(200_000)).toBe('200K ctx')
    expect(formatContextTokens(1_000_000)).toBe('1M ctx')
    expect(formatContextTokens(0)).toBeNull()
  })

  it('filterChatModels drops non-chat types', () => {
    const models = [
      model({ name: 'gpt-4o' }),
      model({ name: 'dall-e', type: 'image' }),
    ]
    expect(filterChatModels(models)).toHaveLength(1)
  })

  it('applyCatalogFilters supports cache-only toggle', () => {
    const models = [
      model({ name: 'gpt-4o', sell_cached_input_per_1m: 100 }),
      model({ name: 'claude-haiku' }),
    ]
    const result = applyCatalogFilters(models, {
      search: '',
      family: 'all',
      billing: 'all',
      sort: 'name',
      cacheOnly: true,
    })
    expect(result.map((m) => m.name)).toEqual(['gpt-4o'])
  })

  it('sortCatalogModels orders by input price', () => {
    const models = [
      model({ name: 'b', sell_input_per_1m: 3000 }),
      model({ name: 'a', sell_input_per_1m: 1000 }),
    ]
    expect(sortCatalogModels(models, 'price_asc').map((m) => m.name)).toEqual(['a', 'b'])
  })

  it('anyModelHasCache reflects catalog contents', () => {
    expect(anyModelHasCache([model({ name: 'a' })])).toBe(false)
    expect(anyModelHasCache([model({ name: 'a', sell_cached_input_per_1m: 1 })])).toBe(true)
  })

  it('inferCapabilities reads API capability flags', () => {
    expect(
      inferCapabilities(
        model({
          name: 'premium-chat',
          supports_tools: true,
          supports_vision: true,
          sell_cached_input_per_1m: 100,
        }),
      ),
    ).toEqual({
      tools: true,
      vision: true,
      cache: true,
    })
    expect(inferCapabilities(model({ name: 'basic-chat' }))).toEqual({
      tools: false,
      vision: false,
      cache: false,
    })
  })

  it('formatCatalogRequestCount abbreviates large counts', () => {
    expect(formatCatalogRequestCount(1500)).toBe('1.5K')
    expect(formatCatalogRequestCount(0)).toBeNull()
  })

  it('formatCatalogLatency formats ms and seconds', () => {
    expect(formatCatalogLatency(450)).toBe('450ms')
    expect(formatCatalogLatency(2400)).toBe('2.4s')
  })
})