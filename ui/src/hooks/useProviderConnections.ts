import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface ProviderConnection {
  id: string
  provider_id: string
  name: string
  auth_type: string
  priority: number
  is_active: boolean
  has_api_key: boolean
  test_status: string
  last_error: string
  error_code: number | null
  last_error_at: string | null
  backoff_level: number
  locked_until: string | null
  model_locks: Record<string, string>
  earliest_lock_until: string | null
  last_used_at: string | null
  consecutive_use_count: number
  created_at: string
  updated_at: string
}

export interface CreateConnectionParams {
  name?: string
  api_key?: string
  auth_type?: string
  priority?: number
  bulk?: string
}

export interface BulkCreateConnectionResult {
  name: string
  id?: string
  error?: string
}

export interface UpdateConnectionParams {
  name?: string
  api_key?: string
  priority?: number
  is_active?: boolean
}

export interface TestConnectionResult {
  ok: boolean
  status?: number
  error?: string
  connection?: ProviderConnection
}

function connectionsKey(providerId: string) {
  return ['provider-connections', providerId] as const
}

export function useProviderConnections(providerId: string, activeOnly = false) {
  const q = activeOnly ? '?active_only=1' : ''
  return useQuery({
    queryKey: [...connectionsKey(providerId), activeOnly],
    queryFn: () =>
      apiClient<{ data: ProviderConnection[] }>(
        `/providers/${providerId}/connections${q}`,
      ),
    enabled: Boolean(providerId),
  })
}

export function useCreateProviderConnection(providerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: Omit<CreateConnectionParams, 'bulk'>) =>
      apiClient<ProviderConnection>(`/providers/${providerId}/connections`, {
        method: 'POST',
        body: JSON.stringify(params),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: connectionsKey(providerId) })
    },
  })
}

export function useBulkCreateProviderConnections(providerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (bulk: string) =>
      apiClient<{ results: BulkCreateConnectionResult[] }>(
        `/providers/${providerId}/connections`,
        { method: 'POST', body: JSON.stringify({ bulk }) },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: connectionsKey(providerId) })
    },
  })
}

export function useUpdateProviderConnection(providerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      connectionId,
      ...params
    }: UpdateConnectionParams & { connectionId: string }) =>
      apiClient<ProviderConnection>(
        `/providers/${providerId}/connections/${connectionId}`,
        { method: 'PATCH', body: JSON.stringify(params) },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: connectionsKey(providerId) })
    },
  })
}

export function useDeleteProviderConnection(providerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (connectionId: string) =>
      apiClient<void>(`/providers/${providerId}/connections/${connectionId}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: connectionsKey(providerId) })
    },
  })
}

export function useUnlockProviderConnection(providerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (connectionId: string) =>
      apiClient<ProviderConnection>(
        `/providers/${providerId}/connections/${connectionId}/unlock`,
        { method: 'POST' },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: connectionsKey(providerId) })
    },
  })
}

export function useReorderProviderConnections(providerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (orderedIds: string[]) =>
      apiClient<void>(`/providers/${providerId}/connections/reorder`, {
        method: 'POST',
        body: JSON.stringify({ ordered_ids: orderedIds }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: connectionsKey(providerId) })
    },
  })
}

export function useTestProviderConnection(providerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (connectionId: string) =>
      apiClient<TestConnectionResult>(
        `/providers/${providerId}/connections/${connectionId}/test`,
        { method: 'POST' },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: connectionsKey(providerId) })
    },
  })
}