import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface ProviderItem {
  id: string
  name: string
  contact_info: string
  status: 'active' | 'paused'
  notes: string
  created_at: string
  updated_at: string
}

interface Paginated<T> {
  data: T[]
  has_more: boolean
  cursor: string
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
// Public storefront price list (no auth)
// ---------------------------------------------------------------------------

export interface PublicModelItem {
  name: string
  type: string
  max_context_tokens?: number
  sell_input_per_1m: number | null
  sell_output_per_1m: number | null
  sell_cached_input_per_1m?: number | null
}

export function usePublicModels() {
  return useQuery({
    queryKey: ['public-models'],
    queryFn: async () => {
      const res = await fetch('/api/v1/public/models')
      if (!res.ok) throw new Error('failed to load price list')
      return (await res.json()) as { data: PublicModelItem[] }
    },
    staleTime: 60_000,
  })
}
