import type { CatalogModelItem } from '../hooks/useProviders'

export type ModelFamily =
  | 'all'
  | 'claude'
  | 'gpt'
  | 'gemini'
  | 'deepseek'
  | 'qwen'
  | 'glm'
  | 'kimi'
  | 'minimax'
  | 'grok'
  | 'llama'
  | 'mistral'
  | 'cohere'
  | 'other'

export type BillingFilter = 'all' | 'token' | 'request'

export type CatalogSort = 'name' | 'price_asc' | 'price_desc'

export interface CatalogFilterState {
  search: string
  family: ModelFamily
  billing: BillingFilter
  sort: CatalogSort
  cacheOnly: boolean
}

export const CATALOG_PAGE_SIZE = 24

const CHAT_TYPES = new Set(['chat', 'completion'])

/** Toolbar chip order — popular families first. */
export const FAMILY_FILTER_ORDER: ModelFamily[] = [
  'all',
  'claude',
  'gpt',
  'gemini',
  'deepseek',
  'qwen',
  'glm',
  'kimi',
  'minimax',
  'grok',
  'llama',
  'mistral',
  'cohere',
  'other',
]

export const FAMILY_LOBE_ICON: Record<Exclude<ModelFamily, 'all'>, string> = {
  claude: 'Claude.Color',
  gpt: 'OpenAI.Color',
  gemini: 'Gemini.Color',
  deepseek: 'DeepSeek.Color',
  qwen: 'Qwen.Color',
  glm: 'Zhipu.Color',
  kimi: 'Kimi.Color',
  minimax: 'Minimax.Color',
  grok: 'XAI',
  llama: 'Ollama',
  mistral: 'Mistral.Color',
  cohere: 'Cohere.Color',
  other: 'LobeHub',
}

const FAMILY_RULES: Array<{ family: Exclude<ModelFamily, 'all' | 'other'>; test: RegExp }> = [
  { family: 'claude', test: /^claude/i },
  { family: 'gpt', test: /^(gpt-|o[134]-|chatgpt)/i },
  { family: 'gemini', test: /^gemini/i },
  { family: 'deepseek', test: /^deepseek/i },
  { family: 'qwen', test: /^qwen/i },
  { family: 'glm', test: /^glm-/i },
  { family: 'kimi', test: /^(kimi|moonshot)/i },
  { family: 'minimax', test: /^minimax/i },
  { family: 'grok', test: /^grok/i },
  { family: 'llama', test: /^(llama|meta-)/i },
  { family: 'mistral', test: /^(mistral|mixtral|codestral)/i },
  { family: 'cohere', test: /^command/i },
]

export function isChatModel(type: string): boolean {
  return CHAT_TYPES.has(type.trim().toLowerCase())
}

export function hasCachePrice(model: CatalogModelItem): boolean {
  const price = model.sell_cached_input_per_1m
  return price != null && price > 0
}

export function anyModelHasCache(models: CatalogModelItem[]): boolean {
  return models.some(hasCachePrice)
}

export function detectFamily(name: string): Exclude<ModelFamily, 'all'> {
  const trimmed = name.trim()
  for (const rule of FAMILY_RULES) {
    if (rule.test.test(trimmed)) return rule.family
  }
  return 'other'
}

export function formatContextTokens(tokens?: number): string | null {
  if (tokens == null || tokens <= 0) return null
  if (tokens >= 1_000_000) {
    const millions = tokens / 1_000_000
    const label = Number.isInteger(millions) ? `${millions}M` : `${millions.toFixed(1)}M`
    return `${label} ctx`
  }
  if (tokens >= 1_000) {
    const thousands = tokens / 1_000
    const label = Number.isInteger(thousands) ? `${thousands}K` : `${thousands.toFixed(1)}K`
    return `${label} ctx`
  }
  return `${tokens} ctx`
}

export function filterChatModels(models: CatalogModelItem[]): CatalogModelItem[] {
  return models.filter((m) => isChatModel(m.type))
}

function matchesSearch(model: CatalogModelItem, search: string): boolean {
  const q = search.trim().toLowerCase()
  if (!q) return true
  return model.name.toLowerCase().includes(q)
}

function matchesFamily(model: CatalogModelItem, family: ModelFamily): boolean {
  if (family === 'all') return true
  return detectFamily(model.name) === family
}

function matchesBilling(model: CatalogModelItem, billing: BillingFilter): boolean {
  if (billing === 'all') return true
  if (billing === 'token') return model.bill_per_token
  return model.bill_per_request
}

function sortKeyPrice(model: CatalogModelItem): number {
  if (model.sell_input_per_1m != null && model.sell_input_per_1m > 0) {
    return model.sell_input_per_1m
  }
  if (model.sell_per_request != null && model.sell_per_request > 0) {
    return model.sell_per_request
  }
  return Number.POSITIVE_INFINITY
}

export function sortCatalogModels(
  models: CatalogModelItem[],
  sort: CatalogSort,
): CatalogModelItem[] {
  const sorted = [...models]
  if (sort === 'name') {
    sorted.sort((a, b) => a.name.localeCompare(b.name))
    return sorted
  }
  if (sort === 'price_asc') {
    sorted.sort((a, b) => sortKeyPrice(a) - sortKeyPrice(b) || a.name.localeCompare(b.name))
    return sorted
  }
  sorted.sort((a, b) => sortKeyPrice(b) - sortKeyPrice(a) || a.name.localeCompare(b.name))
  return sorted
}

export function applyCatalogFilters(
  models: CatalogModelItem[],
  filters: CatalogFilterState,
): CatalogModelItem[] {
  const filtered = models.filter((model) => {
    if (!matchesSearch(model, filters.search)) return false
    if (!matchesFamily(model, filters.family)) return false
    if (!matchesBilling(model, filters.billing)) return false
    if (filters.cacheOnly && !hasCachePrice(model)) return false
    return true
  })
  return sortCatalogModels(filtered, filters.sort)
}

export const DEFAULT_CATALOG_FILTERS: CatalogFilterState = {
  search: '',
  family: 'all',
  billing: 'all',
  sort: 'name',
  cacheOnly: false,
}

export interface ModelCapabilities {
  tools: boolean
  vision: boolean
  cache: boolean
}

/** Reads catalog capability flags from the API (admin-configured). */
export function inferCapabilities(model: CatalogModelItem): ModelCapabilities {
  return {
    tools: model.supports_tools === true,
    vision: model.supports_vision === true,
    cache: hasCachePrice(model),
  }
}

export function formatCatalogRequestCount(count?: number): string | null {
  if (count == null || count <= 0) return null
  if (count >= 1_000_000) return `${(count / 1_000_000).toFixed(1)}M`
  if (count >= 1_000) return `${(count / 1_000).toFixed(1)}K`
  return String(count)
}

export function formatCatalogSuccessRate(rate?: number): string | null {
  if (rate == null || Number.isNaN(rate)) return null
  return `${Math.round(rate)}%`
}

export function formatCatalogLatency(ms?: number): string | null {
  if (ms == null || ms <= 0 || Number.isNaN(ms)) return null
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`
  return `${Math.round(ms)}ms`
}

export function formatCatalogTps(tps?: number): string | null {
  if (tps == null || tps <= 0 || Number.isNaN(tps)) return null
  return tps >= 100 ? String(Math.round(tps)) : tps.toFixed(1)
}