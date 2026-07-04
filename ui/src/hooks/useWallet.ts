import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'
import { useMe } from './useMe'

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
  status: 'pending' | 'approved' | 'rejected'
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
    mutationFn: (params: { amount: number; payment_ref: string }) =>
      apiClient<TopupRequestItem>('/me/topups', {
        method: 'POST',
        body: JSON.stringify(params),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['my-topups'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Admin-side hooks
// ---------------------------------------------------------------------------

export function useAdminTopups(status: string, cursor: string, limit = 25) {
  const { data: me } = useMe()
  const isAdmin = me?.is_system_admin ?? false

  return useQuery({
    queryKey: ['admin-topups', status, cursor, limit],
    queryFn: () =>
      apiClient<Paginated<TopupRequestItem>>(
        `/topups?limit=${limit}${status ? `&status=${status}` : ''}${cursor ? `&cursor=${cursor}` : ''}`,
      ),
    enabled: isAdmin,
  })
}

export function useReviewTopup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { topupId: string; status: 'approved' | 'rejected'; note?: string }) =>
      apiClient<{ status: string; balance: number }>(`/topups/${params.topupId}/review`, {
        method: 'POST',
        body: JSON.stringify({ status: params.status, note: params.note ?? '' }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-topups'] })
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
      void queryClient.invalidateQueries({ queryKey: ['admin-topups'] })
    },
  })
}
