import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface UpstreamModelItem {
  id: string
  provider_id: string
  upstream_id: string
  display_name: string
  is_enabled: boolean
  cost_input_per_1m: number | null
  cost_output_per_1m: number | null
  cost_cached_input_per_1m: number | null
  cost_cache_write_per_1m: number | null
  metadata: string
  created_at: string
  updated_at: string
  provider_name?: string
  provider_slug?: string
  provider_protocol?: string
}

export interface ImportUpstreamModelSpec {
  upstream_id: string
  cost?: { in: number; out: number; cached_in?: number; cache_write?: number }
}

export interface ImportUpstreamModelResult {
  upstream_id: string
  id?: string
  error?: string
}

export interface UpdateUpstreamModelParams {
  display_name?: string
  is_enabled?: boolean
  cost_input_per_1m?: number
  cost_output_per_1m?: number
  cost_cached_input_per_1m?: number
  cost_cache_write_per_1m?: number
}

function providerModelsKey(providerId: string) {
  return ['provider-upstream-models', providerId] as const
}

export function useProviderUpstreamModels(providerId: string, enabledOnly = false) {
  const q = enabledOnly ? '?enabled_only=1' : ''
  return useQuery({
    queryKey: [...providerModelsKey(providerId), enabledOnly],
    queryFn: () =>
      apiClient<{ data: UpstreamModelItem[] }>(
        `/providers/${providerId}/upstream-models${q}`,
      ),
    enabled: Boolean(providerId),
  })
}

export function useAllUpstreamModels(enabledOnly = true, queryEnabled = true) {
  const q = enabledOnly ? '' : '?enabled_only=0'
  return useQuery({
    queryKey: ['upstream-models', enabledOnly],
    queryFn: () => apiClient<{ data: UpstreamModelItem[] }>(`/upstream-models${q}`),
    staleTime: 30_000,
    enabled: queryEnabled,
  })
}

export function useImportProviderUpstreamModels(providerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (models: ImportUpstreamModelSpec[]) =>
      apiClient<{ results: ImportUpstreamModelResult[] }>(
        `/providers/${providerId}/upstream-models/import`,
        { method: 'POST', body: JSON.stringify({ models }) },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: providerModelsKey(providerId) })
      void queryClient.invalidateQueries({ queryKey: ['upstream-models'] })
    },
  })
}

export function useUpdateProviderUpstreamModel(providerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      modelId,
      ...params
    }: UpdateUpstreamModelParams & { modelId: string }) =>
      apiClient<UpstreamModelItem>(
        `/providers/${providerId}/upstream-models/${modelId}`,
        { method: 'PATCH', body: JSON.stringify(params) },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: providerModelsKey(providerId) })
      void queryClient.invalidateQueries({ queryKey: ['upstream-models'] })
    },
  })
}

export function useDeleteProviderUpstreamModel(providerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (modelId: string) =>
      apiClient<void>(`/providers/${providerId}/upstream-models/${modelId}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: providerModelsKey(providerId) })
      void queryClient.invalidateQueries({ queryKey: ['upstream-models'] })
    },
  })
}