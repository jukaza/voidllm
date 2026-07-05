import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'
import { useMe } from './useMe'

export type FinanceRangeDays = 7 | 30 | 90

export function financeRangeISO(days: FinanceRangeDays): { from: string; to: string } {
  const to = new Date()
  const from = new Date(to.getTime() - days * 24 * 3_600_000)
  return { from: from.toISOString(), to: to.toISOString() }
}

export interface FinanceSummaryTotals {
  wallet_liability: number
  topup_inflow: number
  topup_pay_amount: number
  topup_bonus: number
  usage_outflow: number
  adjustment_net: number
  refund_total: number
  pending_topup_count: number
  completed_topup_count: number
}

export interface FinanceDailyBucket {
  day: string
  topup_inflow: number
  usage_outflow: number
  adjustment_net: number
  refund_total: number
  net_flow: number
}

export interface FinanceSummaryResponse {
  from: string
  to: string
  timezone: string
  currency: string
  totals: FinanceSummaryTotals
  daily: FinanceDailyBucket[]
}

export interface FinanceTopupItem {
  id: string
  user_id: string
  user_email?: string
  user_display_name?: string
  amount: number
  payment_ref: string
  status: 'pending' | 'completed' | 'expired' | 'failed'
  trade_no?: string
  pay_amount?: number
  credit_amount?: number
  bonus_amount?: number
  bonus_detail?: Record<string, unknown> | null
  sepay_tx_id?: string
  expires_at?: string | null
  completed_at?: string | null
  note: string
  created_at: string
}

export interface FinanceTransactionItem {
  id: string
  user_id: string
  user_email?: string
  user_display_name?: string
  type: 'topup' | 'usage' | 'adjustment' | 'refund'
  amount: number
  balance_after: number
  ref_id: string
  description: string
  created_at: string
  usage_log_url?: string
}

interface Paginated<T> {
  data: T[]
  has_more: boolean
  cursor: string
}

export function useFinanceSummary(from: string, to: string) {
  const { data: me } = useMe()
  const isAdmin = me?.is_system_admin ?? false
  return useQuery({
    queryKey: ['finance-summary', from, to],
    queryFn: () =>
      apiClient<FinanceSummaryResponse>(
        `/admin/finance/summary?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
      ),
    enabled: isAdmin && !!from && !!to,
    staleTime: 30_000,
  })
}

export function useAdminFinanceTopups(params: {
  from?: string
  to?: string
  status?: string
  user_id?: string
  cursor?: string
  limit?: number
}) {
  const { data: me } = useMe()
  const isAdmin = me?.is_system_admin ?? false
  const qs = new URLSearchParams()
  if (params.from) qs.set('from', params.from)
  if (params.to) qs.set('to', params.to)
  if (params.status) qs.set('status', params.status)
  if (params.user_id) qs.set('user_id', params.user_id)
  if (params.cursor) qs.set('cursor', params.cursor)
  if (params.limit) qs.set('limit', String(params.limit))
  const query = qs.toString()
  return useQuery({
    queryKey: ['finance-topups', params],
    queryFn: () =>
      apiClient<Paginated<FinanceTopupItem>>(`/admin/finance/topups${query ? `?${query}` : ''}`),
    enabled: isAdmin,
    staleTime: 15_000,
  })
}

export function useReviewFinanceTopup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { topupId: string; action: 'approve' | 'reject'; note?: string }) =>
      apiClient<{ status: string; balance: number }>(
        `/admin/finance/topups/${params.topupId}/review`,
        {
          method: 'POST',
          body: JSON.stringify({ action: params.action, note: params.note ?? '' }),
        },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['finance-topups'] })
      void queryClient.invalidateQueries({ queryKey: ['finance-summary'] })
      void queryClient.invalidateQueries({ queryKey: ['finance-transactions'] })
    },
  })
}

export function useAdminFinanceTransactions(params: {
  from: string
  to: string
  type?: string
  user_id?: string
  cursor?: string
  limit?: number
}) {
  const { data: me } = useMe()
  const isAdmin = me?.is_system_admin ?? false
  const qs = new URLSearchParams({
    from: params.from,
    to: params.to,
  })
  if (params.type) qs.set('type', params.type)
  if (params.user_id) qs.set('user_id', params.user_id)
  if (params.cursor) qs.set('cursor', params.cursor)
  if (params.limit) qs.set('limit', String(params.limit))
  return useQuery({
    queryKey: ['finance-transactions', params],
    queryFn: () =>
      apiClient<Paginated<FinanceTransactionItem>>(`/admin/finance/transactions?${qs.toString()}`),
    enabled: isAdmin && !!params.from && !!params.to,
    staleTime: 15_000,
  })
}