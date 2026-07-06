import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export type APIKeyStatus = 'active' | 'disabled' | 'expired' | 'quota_exhausted'

export function normalizeKeyStatus(status?: APIKeyStatus): APIKeyStatus {
  return status ?? 'active'
}

export interface KeyLimitsLive {
  daily_tokens?: number
  monthly_tokens?: number
  requests_per_minute?: number
  requests_per_day?: number
}

export interface KeyUsageToday {
  requests: number
  tokens: number
  spend: number
}

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
  status?: APIKeyStatus
  spend_cap?: number
  spend_used?: number
  ip_whitelist?: string
  ip_blacklist?: string
  model_limits_enabled?: boolean
  model_limits?: string[]
  limits_live?: KeyLimitsLive
  usage_today?: KeyUsageToday
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
  spend_cap?: number
  ip_whitelist?: string
  ip_blacklist?: string
  model_limits_enabled?: boolean
  model_limits?: string[]
  status?: APIKeyStatus
}

export interface UpdateAPIKeyParams {
  name?: string
  daily_token_limit?: number
  monthly_token_limit?: number
  requests_per_minute?: number
  requests_per_day?: number
  expires_at?: string | null
  status?: APIKeyStatus
  spend_cap?: number
  ip_whitelist?: string
  ip_blacklist?: string
  model_limits_enabled?: boolean
  model_limits?: string[]
}

export interface KeyUsageResponse {
  from: string
  to: string
  requests: number
  tokens: number
  spend: number
  daily_tokens?: number
  monthly_tokens?: number
  requests_today?: number
}

export interface KeysPolicy {
  max_per_user: number
  auto_create_on_register: boolean
  default_expiry_days: number
  allow_custom_key: boolean
  trust_forwarded_ip: boolean
}

interface RotateKeyResponse {
  new_key: {
    id: string
    key?: string
    hint: string
    expires_at?: string
  }
  old_key: {
    id: string
    hint: string
    expires_at?: string
  }
}

const KEYS_QUERY_KEY = ['api-keys'] as const
const KEYS_POLICY_KEY = ['keys-policy'] as const

export interface UseAPIKeysOptions {
  cursor?: string
  userId?: string
  includeDeleted?: boolean
}

export function useAPIKeys(options?: UseAPIKeysOptions) {
  const { cursor, userId, includeDeleted } = options ?? {}
  const params = new URLSearchParams({ limit: '20' })
  if (cursor) params.set('cursor', cursor)
  if (userId) params.set('user_id', userId)
  if (includeDeleted) params.set('include_deleted', 'true')

  return useQuery({
    queryKey: [...KEYS_QUERY_KEY, cursor, userId, includeDeleted],
    queryFn: () => apiClient<PaginatedKeys>(`/keys?${params}`),
  })
}

export function useKeyUsage(keyId: string | null | undefined) {
  return useQuery({
    queryKey: ['key-usage', keyId],
    queryFn: () => apiClient<KeyUsageResponse>(`/keys/${keyId}/usage`),
    enabled: !!keyId,
    staleTime: 30_000,
  })
}

export function useKeyLimits(keyId: string | null | undefined) {
  return useQuery({
    queryKey: ['key-limits', keyId],
    queryFn: () => apiClient<KeyLimitsLive>(`/keys/${keyId}/limits`),
    enabled: !!keyId,
    staleTime: 15_000,
    refetchInterval: 30_000,
  })
}

export function useKeysPolicy() {
  return useQuery({
    queryKey: [...KEYS_POLICY_KEY, 'admin'] as const,
    queryFn: () => apiClient<KeysPolicy>('/admin/settings/keys'),
    staleTime: 30_000,
  })
}

export function useUpdateKeysPolicy() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: Partial<KeysPolicy>) =>
      apiClient<KeysPolicy>('/admin/settings/keys', {
        method: 'PUT',
        body: JSON.stringify(payload),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData([...KEYS_POLICY_KEY, 'admin'], data)
    },
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
      queryClient.invalidateQueries({ queryKey: KEYS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
    },
  })
}

export function useCreateAPIKeysBatch() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (keys: CreateAPIKeyParams[]) =>
      apiClient<{ data: APIKeyResponse[] }>('/keys/batch', {
        method: 'POST',
        body: JSON.stringify({ keys }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: KEYS_QUERY_KEY })
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
      queryClient.invalidateQueries({ queryKey: KEYS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
    },
  })
}

export function useUpdateAPIKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ keyId, params }: { keyId: string; params: UpdateAPIKeyParams }) =>
      apiClient<APIKeyResponse>(`/keys/${keyId}`, {
        method: 'PATCH',
        body: JSON.stringify(params),
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: KEYS_QUERY_KEY }),
  })
}

export function usePatchAPIKeyStatus() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ keyId, status }: { keyId: string; status: APIKeyStatus }) =>
      apiClient<APIKeyResponse>(`/keys/${keyId}`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: KEYS_QUERY_KEY }),
  })
}

export function useRotateAPIKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (keyId: string) => {
      const resp = await apiClient<RotateKeyResponse>(`/keys/${keyId}/rotate`, { method: 'POST' })
      return {
        id: resp.new_key.id,
        key: resp.new_key.key,
        key_hint: resp.new_key.hint,
        expires_at: resp.new_key.expires_at ?? null,
      }
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: KEYS_QUERY_KEY }),
  })
}