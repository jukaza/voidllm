import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface APIKeyResponse {
  id: string
  key?: string
  key_hint: string
  key_type: string
  name: string
  user_id: string | null
  daily_token_limit: number
  monthly_token_limit: number
  requests_per_minute: number
  requests_per_day: number
  expires_at: string | null
  last_used_at: string | null
  created_by: string
  created_at: string
  updated_at: string
}

export interface PaginatedKeys {
  data: APIKeyResponse[]
  has_more: boolean
  next_cursor?: string
}

export interface CreateAPIKeyParams {
  name: string
  key_type?: string
  user_id?: string
  expires_at?: string
  daily_token_limit?: number
  monthly_token_limit?: number
  requests_per_minute?: number
  requests_per_day?: number
}

export function useAPIKeys(cursor?: string) {
  const params = new URLSearchParams({ limit: '20' })
  if (cursor) params.set('cursor', cursor)
  return useQuery({
    queryKey: ['api-keys', cursor],
    queryFn: () => apiClient<PaginatedKeys>(`/keys?${params}`),
  })
}

export function useCreateAPIKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: CreateAPIKeyParams) =>
      apiClient<APIKeyResponse>('/keys', {
        method: 'POST',
        body: JSON.stringify({ ...params, key_type: params.key_type ?? 'user_key' }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
    },
  })
}

export function useDeleteAPIKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (keyId: string) =>
      apiClient<void>(`/keys/${keyId}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
    },
  })
}

export function useUpdateAPIKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ keyId, params }: { keyId: string; params: Record<string, unknown> }) =>
      apiClient<APIKeyResponse>(`/keys/${keyId}`, {
        method: 'PATCH',
        body: JSON.stringify(params),
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['api-keys'] }),
  })
}

export function useRotateAPIKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (keyId: string) =>
      apiClient<APIKeyResponse>(`/keys/${keyId}/rotate`, { method: 'POST' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['api-keys'] }),
  })
}