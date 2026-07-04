import { useQuery } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface ChannelModelStats {
  model_name: string
  total_requests: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  cached_tokens: number
  revenue: number
}

export interface ChannelUsageRow {
  channel_id: string
  channel_label: string
  provider?: string
  provider_logo?: string
  provider_slug?: string
  provider_protocol?: string
  total_requests: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  cached_tokens: number
  revenue: number
  models: ChannelModelStats[]
}

export interface ChannelUsageTotals {
  total_requests: number
  prompt_tokens: number
  completion_tokens: number
  cached_tokens: number
  total_tokens: number
  revenue: number
}

export interface ChannelTopologyNode {
  channel_id: string
  label: string
  provider?: string
  channel_type?: string
}

export interface ChannelTopologyProduct {
  name: string
  provider?: string
  channels: ChannelTopologyNode[]
}

export interface ChannelUsageResponse {
  from: string
  to: string
  totals: ChannelUsageTotals
  channels: ChannelUsageRow[]
  topology: ChannelTopologyProduct[]
}

export function useChannelUsage(from: string, to: string, enabled: boolean) {
  const query = new URLSearchParams({ from, to })
  return useQuery({
    queryKey: ['usage-channels', from, to],
    queryFn: () => apiClient<ChannelUsageResponse>(`/usage/channels?${query.toString()}`),
    enabled: enabled && !!from && !!to,
    staleTime: 60_000,
  })
}