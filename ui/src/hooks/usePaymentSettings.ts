import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export type BonusType = 'percent' | 'fixed'

export interface TierBonus {
  min_amount: number
  bonus_type: BonusType
  bonus_percent: number
  bonus_fixed: number
  label?: string
}

export interface Campaign {
  id: string
  name: string
  enabled: boolean
  start_at: string
  end_at: string
  bonus_type: BonusType
  bonus_percent: number
  bonus_fixed: number
  min_amount: number
  max_bonus: number
  first_topup_only: boolean
}

export interface FirstTopupBonus {
  enabled: boolean
  bonus_type: BonusType
  bonus_percent: number
  bonus_fixed: number
}

export type WebhookAuthMode = 'api_key' | 'hmac'

export interface SepayConfig {
  enabled: boolean
  bank_code: string
  account_number: string
  account_name: string
  webhook_auth_mode: WebhookAuthMode
  webhook_token?: string
  webhook_secret?: string
  webhook_ip_check: boolean
  min_amount: number
  max_amount: number
  order_ttl_minutes: number
}

export interface PaymentSettings {
  sepay: SepayConfig
  amount_presets: number[]
  tier_bonuses: TierBonus[]
  campaigns: Campaign[]
  first_topup: FirstTopupBonus
  bonus_stack_mode: 'stack' | 'max'
  webhook_token_configured: boolean
  webhook_secret_configured: boolean
  webhook_auth_mode: WebhookAuthMode
  webhook_url: string
}

export interface PublicTopupConfig {
  enabled: boolean
  min_amount: number
  max_amount: number
  amount_presets: number[]
  tier_bonuses: TierBonus[]
  active_campaigns: Array<{
    id: string
    name: string
    bonus_type: BonusType
    bonus_percent: number
    bonus_fixed: number
    min_amount: number
    max_bonus: number
    end_at: string
  }>
  first_topup: FirstTopupBonus
  bonus_stack_mode: 'stack' | 'max'
  banks: Array<{ code: string; name: string }>
}

export interface TopupQuote {
  pay_amount: number
  credit_amount: number
  bonus_amount: number
}

export interface SepayOrder {
  trade_no: string
  pay_amount: number
  credit_amount: number
  bonus_amount: number
  bank_code: string
  bank_name: string
  account_number: string
  account_name: string
  qr_url: string
  expires_at: string
  status: string
}

const PAYMENT_QUERY_KEY = ['payment-settings'] as const

export function useAdminPaymentSettings() {
  return useQuery({
    queryKey: [...PAYMENT_QUERY_KEY, 'admin'] as const,
    queryFn: () => apiClient<PaymentSettings>('/admin/settings/payment'),
    staleTime: 30_000,
  })
}

export function useUpdatePaymentSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: Partial<PaymentSettings>) =>
      apiClient<PaymentSettings>('/admin/settings/payment', {
        method: 'PUT',
        body: JSON.stringify(payload),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData([...PAYMENT_QUERY_KEY, 'admin'], data)
    },
  })
}

export function usePublicTopupConfig() {
  return useQuery({
    queryKey: [...PAYMENT_QUERY_KEY, 'public'] as const,
    queryFn: async () => {
      const res = await fetch('/api/v1/public/topup-config')
      if (!res.ok) throw new Error('Failed to load top-up config')
      return res.json() as Promise<PublicTopupConfig>
    },
    staleTime: 60_000,
  })
}

export function useTopupQuote(amount: number | null) {
  return useQuery({
    queryKey: ['topup-quote', amount],
    queryFn: () =>
      apiClient<TopupQuote>('/me/topups/quote', {
        method: 'POST',
        body: JSON.stringify({ amount }),
      }),
    enabled: amount != null && amount > 0,
    staleTime: 10_000,
  })
}