import { useQuery } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface UsageDataPoint {
  group_key: string
  group_label?: string
  total_requests: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  cached_tokens: number
  revenue: number
  avg_duration_ms: number
}

export interface UsageResponse {
  from: string
  to: string
  group_by: string
  data: UsageDataPoint[]
}

export interface MyUsageFilters {
  keyId?: string
  model?: string
}

function buildMyUsageQuery(
  from: string,
  to: string,
  groupBy: string,
  filters?: MyUsageFilters,
): string {
  const q = new URLSearchParams({
    from,
    to,
    group_by: groupBy,
  })
  if (filters?.keyId) q.set('key_id', filters.keyId)
  if (filters?.model) q.set('model', filters.model)
  return q.toString()
}

export function useMyUsage(
  from: string,
  to: string,
  groupBy: string,
  filters?: MyUsageFilters,
) {
  return useQuery({
    queryKey: ['usage-me', from, to, groupBy, filters],
    queryFn: () => apiClient<UsageResponse>(`/usage/me?${buildMyUsageQuery(from, to, groupBy, filters)}`),
    enabled: !!from && !!to,
    staleTime: 60_000,
  })
}

export function useSystemUsage(
  params: { from: string; to: string; groupBy: string },
  enabled: boolean,
) {
  const { from, to, groupBy } = params
  const query = new URLSearchParams({ from, to, group_by: groupBy })

  return useQuery({
    queryKey: ['system-usage', from, to, groupBy],
    queryFn: () => apiClient<UsageResponse>(`/usage?${query.toString()}`),
    enabled: enabled && !!from && !!to,
    staleTime: 60_000,
  })
}

