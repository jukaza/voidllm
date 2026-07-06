import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface ProviderItem {
  id: string
  name: string
  contact_info: string
  status: 'active' | 'paused'
  notes: string
  slug?: string
  protocol?: string
  logo?: string
  base_url?: string
  has_api_key?: boolean
  rpm_limit?: number
  created_at: string
  updated_at: string
}

interface Paginated<T> {
  data: T[]
  has_more: boolean
  cursor: string
}

export interface ProviderPreset {
  id: string
  name: string
  logo: string
  protocol: string
  base_url: string
  key_hint?: string
  docs_url?: string
}

export function useProviderPresets() {
  return useQuery({
    queryKey: ['provider-presets'],
    queryFn: () => apiClient<{ data: ProviderPreset[] }>('/providers/presets'),
    staleTime: 300_000,
  })
}

export interface ProviderUsageEntry {
  revenue: number
  total_requests: number
}

export interface ProviderUsageSlice {
  from: string
  to: string
  totals: {
    total_requests: number
    revenue: number
  }
  by_provider: Record<string, ProviderUsageEntry>
}

export interface ProviderUsageResponse {
  today: ProviderUsageSlice
  all_time: ProviderUsageSlice
}

export function useProviderUsage() {
  return useQuery({
    queryKey: ['provider-usage'],
    queryFn: () => apiClient<ProviderUsageResponse>('/providers/usage'),
    staleTime: 60_000,
  })
}

export function useProviders(cursor = '', limit = 50) {
  return useQuery({
    queryKey: ['providers', cursor, limit],
    queryFn: () =>
      apiClient<Paginated<ProviderItem>>(
        `/providers?limit=${limit}${cursor ? `&cursor=${cursor}` : ''}`,
      ),
  })
}

export function useProvider(id: string) {
  return useQuery({
    queryKey: ['provider', id],
    queryFn: () => apiClient<ProviderItem>(`/providers/${id}`),
    enabled: Boolean(id),
  })
}

export interface ProviderParams {
  name?: string
  contact_info?: string
  status?: string
  notes?: string
  slug?: string
  protocol?: string
  logo?: string
  base_url?: string
  api_key?: string
  rpm_limit?: number
}

export function useCreateProvider() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: ProviderParams) =>
      apiClient<ProviderItem>('/providers', {
        method: 'POST',
        body: JSON.stringify(params),
      }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['providers'] }),
  })
}

export function useUpdateProvider() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...params }: ProviderParams & { id: string }) =>
      apiClient<ProviderItem>(`/providers/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(params),
      }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['providers'] }),
  })
}

export function useDeleteProvider() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/providers/${id}`, { method: 'DELETE' }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['providers'] }),
  })
}

// ---------------------------------------------------------------------------
// Provider wizard (discover + import)
// ---------------------------------------------------------------------------

export interface DiscoveredModel {
  id: string
  known_cost?: { in: number; out: number; cached_in?: number; cache_write?: number }
  exists: boolean
}

export interface DiscoverModelsParams {
  preset_id?: string
  provider_id?: string
  base_url?: string
  protocol?: string
  api_key?: string
}

export interface DiscoverModelsResponse {
  success: boolean
  message: string
  data: DiscoveredModel[]
}

export function useDiscoverProviderModels() {
  return useMutation({
    mutationFn: (params: DiscoverModelsParams) =>
      apiClient<DiscoverModelsResponse>('/providers/discover-models', {
        method: 'POST',
        body: JSON.stringify(params),
      }),
  })
}

export interface ImportModelSpec {
  upstream_id: string
  product_name?: string
}

export interface ImportProviderParams {
  preset_id?: string
  provider_id?: string
  name?: string
  slug?: string
  base_url?: string
  protocol?: string
  api_key?: string
  models: ImportModelSpec[]
  markup?: number
  make_public?: boolean
}

export interface ImportProviderResult {
  upstream_id: string
  inventory_id?: string
  error?: string
}

export function useImportProvider() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: ImportProviderParams) =>
      apiClient<{ provider: ProviderItem; results: ImportProviderResult[] }>('/providers/import', {
        method: 'POST',
        body: JSON.stringify(params),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['providers'] })
      void queryClient.invalidateQueries({ queryKey: ['provider'] })
      void queryClient.invalidateQueries({ queryKey: ['provider-upstream-models'] })
      void queryClient.invalidateQueries({ queryKey: ['upstream-models'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Public storefront price list (no auth)
// ---------------------------------------------------------------------------

export interface CatalogModelStats {
  window_days?: number
  request_count?: number
  success_rate?: number
  avg_latency_ms?: number
  avg_tps?: number
}

export interface CatalogModelItem {
  name: string
  type: string
  logo?: string
  max_context_tokens?: number
  bill_per_token: boolean
  bill_per_request: boolean
  bill_min_per_request?: boolean
  supports_tools?: boolean
  supports_vision?: boolean
  sell_input_per_1m?: number | null
  sell_output_per_1m?: number | null
  sell_cached_input_per_1m?: number | null
  sell_per_request?: number | null
  sell_min_per_request?: number | null
  stats?: CatalogModelStats | null
}

export type CatalogScope = 'landing' | 'member'

/** @deprecated Use CatalogModelItem */
export type PublicModelItem = CatalogModelItem

export function usePublicCatalog(scope: CatalogScope = 'member') {
  const queryScope = scope === 'landing' ? 'landing' : 'member'
  return useQuery({
    queryKey: ['public-catalog', queryScope],
    queryFn: async () => {
      const res = await fetch(`/api/v1/public/catalog?scope=${queryScope}`)
      if (res.status === 404) {
        return { data: [] as CatalogModelItem[] }
      }
      if (!res.ok) throw new Error('failed to load catalog')
      return (await res.json()) as { data: CatalogModelItem[] }
    },
    staleTime: 60_000,
  })
}

/** Full member catalog (all priced active models). */
export function useMemberCatalog() {
  return usePublicCatalog('member')
}

/** @deprecated Use usePublicCatalog */
export function usePublicModels() {
  return usePublicCatalog()
}
