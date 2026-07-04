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
  cost_estimate: number
  avg_duration_ms: number
}

export interface UsageResponse {
  from: string
  to: string
  group_by: string
  data: UsageDataPoint[]
}

export function useMyUsage(from: string, to: string, groupBy: string) {
  return useQuery({
    queryKey: ['usage-me', from, to, groupBy],
    queryFn: () =>
      apiClient<UsageResponse>(
        `/usage/me?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&group_by=${groupBy}`,
      ),
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

