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

export function useProviders(cursor = '', limit = 50) {
  return useQuery({
    queryKey: ['providers', cursor, limit],
    queryFn: () =>
      apiClient<Paginated<ProviderItem>>(
        `/providers?limit=${limit}${cursor ? `&cursor=${cursor}` : ''}`,
      ),
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
  product_name: string
  model_id?: string
  created_model: boolean
  route_id?: string
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
      void queryClient.invalidateQueries({ queryKey: ['models'] })
      void queryClient.invalidateQueries({ queryKey: ['public-catalog'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Public storefront price list (no auth)
// ---------------------------------------------------------------------------

export interface CatalogModelItem {
  name: string
  type: string
  logo?: string
  max_context_tokens?: number
  bill_per_token: boolean
  bill_per_request: boolean
  sell_input_per_1m?: number | null
  sell_output_per_1m?: number | null
  sell_cached_input_per_1m?: number | null
  sell_per_request?: number | null
}

/** @deprecated Use CatalogModelItem */
export type PublicModelItem = CatalogModelItem

export function usePublicCatalog() {
  return useQuery({
    queryKey: ['public-catalog'],
    queryFn: async () => {
      const res = await fetch('/api/v1/public/catalog')
      if (!res.ok) throw new Error('failed to load catalog')
      return (await res.json()) as { data: CatalogModelItem[] }
    },
    staleTime: 60_000,
  })
}

/** @deprecated Use usePublicCatalog */
export function usePublicModels() {
  return usePublicCatalog()
}
