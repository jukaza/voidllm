import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'
import type { SepayOrder } from './usePaymentSettings'

export interface WalletResponse {
  balance: number
  currency: string
}

export interface TransactionItem {
  id: string
  type: 'topup' | 'usage' | 'adjustment' | 'refund'
  amount: number
  balance_after: number
  ref_id: string
  description: string
  created_at: string
}

export interface TopupRequestItem {
  id: string
  user_id: string
  amount: number
  payment_ref: string
  status: 'pending' | 'completed' | 'expired' | 'failed'
  trade_no?: string
  pay_amount?: number
  credit_amount?: number
  bonus_amount?: number
  reviewed_by?: string | null
  reviewed_at?: string | null
  note: string
  created_at: string
}

interface Paginated<T> {
  data: T[]
  has_more: boolean
  cursor: string
}

export function useMyWallet() {
  return useQuery({
    queryKey: ['my-wallet'],
    queryFn: () => apiClient<WalletResponse>('/me/wallet'),
    staleTime: 15_000,
  })
}

export function useMyTransactions(cursor: string, limit = 25) {
  return useQuery({
    queryKey: ['my-transactions', cursor, limit],
    queryFn: () =>
      apiClient<Paginated<TransactionItem>>(
        `/me/transactions?limit=${limit}${cursor ? `&cursor=${cursor}` : ''}`,
      ),
  })
}

export function useMyTopups(cursor: string, limit = 25) {
  return useQuery({
    queryKey: ['my-topups', cursor, limit],
    queryFn: () =>
      apiClient<Paginated<TopupRequestItem>>(
        `/me/topups?limit=${limit}${cursor ? `&cursor=${cursor}` : ''}`,
      ),
  })
}

export function useCreateTopup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { amount: number }) =>
      apiClient<SepayOrder>('/me/topups', {
        method: 'POST',
        body: JSON.stringify(params),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['my-topups'] })
      void queryClient.invalidateQueries({ queryKey: ['my-wallet'] })
    },
  })
}

export function useAdjustWallet() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { userId: string; amount: number; description: string }) =>
      apiClient<{ balance: number }>(`/users/${params.userId}/wallet/adjust`, {
        method: 'POST',
        body: JSON.stringify({ amount: params.amount, description: params.description }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['finance-topups'] })
      void queryClient.invalidateQueries({ queryKey: ['finance-summary'] })
      void queryClient.invalidateQueries({ queryKey: ['finance-transactions'] })
    },
  })
}
