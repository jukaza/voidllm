import { useQuery } from '@tanstack/react-query'
import apiClient from '../api/client'
import type { UsageResponse } from './useUsage'

export function useTopModels(enabled = true) {
  const to = new Date().toISOString()
  const from = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString()

  return useQuery({
    queryKey: ['top-models', from, to],
    queryFn: () =>
      apiClient<UsageResponse>(
        `/usage/me?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&group_by=model`,
      ),
    enabled,
    staleTime: 60_000,
    select: (data) => data.data.slice(0, 5),
  })
}