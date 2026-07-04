import { useQuery } from '@tanstack/react-query'
import apiClient from '../api/client'
import { useMe } from './useMe'

export interface UsageLogEntry {
  id: string
  request_id: string
  created_at: string
  model_name: string
  requested_model_name?: string
  user_id?: string
  user_display_name?: string
  key_id?: string
  key_hint?: string
  deployment_id?: string
  channel_label?: string
  provider_name?: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  cached_tokens: number
  cache_write_tokens?: number
  revenue?: number
  status_code: number
  duration_ms: number
  ttft_ms?: number
  log_type: string
  is_stream: boolean
  meta?: Record<string, unknown>
}

export interface UsageLogsResponse {
  from: string
  to: string
  data: UsageLogEntry[]
  next_cursor?: string
}

export interface UsageLogsFilters {
  from: string
  to: string
  model?: string
  user_id?: string
  key_id?: string
  deployment_id?: string
  request_id?: string
  log_type?: string
  status?: number
  cursor?: string
  limit?: number
}

export interface ProfitTotals {
  total_requests: number
  revenue: number
  prompt_tokens: number
  completion_tokens: number
  cached_tokens: number
}

export interface ProfitDataPoint {
  group_key?: string
  group_label?: string
  total_requests: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  cached_tokens: number
  revenue: number
  avg_duration_ms: number
}

export interface ProfitResponse {
  from: string
  to: string
  group_by?: string
  totals: ProfitTotals
  data: ProfitDataPoint[]
}

function buildLogsQuery(filters: UsageLogsFilters): string {
  const q = new URLSearchParams()
  q.set('from', filters.from)
  q.set('to', filters.to)
  if (filters.model) q.set('model', filters.model)
  if (filters.user_id) q.set('user_id', filters.user_id)
  if (filters.key_id) q.set('key_id', filters.key_id)
  if (filters.deployment_id) q.set('deployment_id', filters.deployment_id)
  if (filters.request_id) q.set('request_id', filters.request_id)
  if (filters.log_type) q.set('log_type', filters.log_type)
  if (filters.status) q.set('status', String(filters.status))
  if (filters.cursor) q.set('cursor', filters.cursor)
  if (filters.limit) q.set('limit', String(filters.limit))
  return q.toString()
}

export function useUsageLogs(filters: UsageLogsFilters) {
  const { data: me } = useMe()
  const isAdmin = me?.is_system_admin ?? false
  const path = isAdmin ? '/usage/logs' : '/usage/me/logs'

  return useQuery({
    queryKey: ['usage-logs', isAdmin, filters],
    queryFn: () =>
      apiClient<UsageLogsResponse>(`${path}?${buildLogsQuery(filters)}`),
    enabled: !!filters.from && !!filters.to,
  })
}

export function useUsageLogDetail(requestId: string | null) {
  const { data: me } = useMe()
  const isAdmin = me?.is_system_admin ?? false
  const path = isAdmin ? '/usage/logs' : '/usage/me/logs'

  return useQuery({
    queryKey: ['usage-log-detail', isAdmin, requestId],
    queryFn: () => apiClient<UsageLogEntry>(`${path}/${requestId}`),
    enabled: !!requestId,
  })
}

export function useProfitReport(from: string, to: string, groupBy: string) {
  const { data: me } = useMe()
  const isAdmin = me?.is_system_admin ?? false

  return useQuery({
    queryKey: ['usage-profit', from, to, groupBy],
    queryFn: () =>
      apiClient<ProfitResponse>(
        `/usage/profit?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&group_by=${groupBy}`,
      ),
    enabled: isAdmin && !!from && !!to,
  })
}