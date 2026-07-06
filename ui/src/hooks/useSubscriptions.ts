import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface SubscriptionPackage {
  id: string
  name: string
  description: string
  cover_type: 'upload' | 'default' | 'url'
  cover_value: string
  enabled: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

export interface SubscriptionPlan {
  id: string
  package_id: string
  name: string
  description: string
  price: number
  validity_days: number
  max_concurrent_bindings: number
  active_bindings: number
  slots_remaining?: number
  daily_token_limit: number
  monthly_token_limit: number
  daily_request_limit: number
  monthly_request_limit: number
  requests_per_minute: number
  requests_per_day: number
  allowed_models: string[]
  quota_exceeded_policy: 'block' | 'fallback_wallet'
  for_sale: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

export interface SubscriptionPlanQuota {
  validity_days: number
  daily_token_limit: number
  monthly_token_limit: number
  daily_request_limit: number
  monthly_request_limit: number
  requests_per_minute: number
  requests_per_day: number
  allowed_models: string[]
  quota_exceeded_policy: 'block' | 'fallback_wallet'
}

export interface SubscriptionUsageSnapshot {
  daily_tokens: number
  monthly_tokens: number
  requests_per_minute: number
  requests_per_day: number
  monthly_requests: number
}

export interface UserSubscription {
  id: string
  user_id: string
  plan_id: string
  plan_name?: string
  package_id?: string
  package_name?: string
  status: string
  starts_at: string
  expires_at: string
  days_remaining?: number
  order_id?: string
  plan?: SubscriptionPlanQuota
  usage?: SubscriptionUsageSnapshot
  created_at: string
  updated_at: string
}

export interface KeySubscriptionBinding {
  key_id: string
  user_subscription_id: string
  plan_id?: string
  plan_name?: string
  package_name?: string
  expires_at?: string
  status: string
}

export function useSubscriptionPackages(includeDisabled = true) {
  return useQuery({
    queryKey: ['subscription-packages', includeDisabled],
    queryFn: () =>
      apiClient<{ data: SubscriptionPackage[] }>(
        `/admin/subscription-packages${includeDisabled ? '?include_disabled=1' : ''}`,
      ),
  })
}

export function useSubscriptionPlans(packageId?: string) {
  return useQuery({
    queryKey: ['subscription-plans', packageId ?? 'all'],
    queryFn: () =>
      apiClient<{ data: SubscriptionPlan[] }>(
        `/admin/subscription-plans${packageId ? `?package_id=${encodeURIComponent(packageId)}` : ''}`,
      ),
  })
}

export function useMySubscriptions() {
  return useQuery({
    queryKey: ['my-subscriptions'],
    queryFn: () => apiClient<{ data: UserSubscription[] }>('/my-subscriptions'),
  })
}

export function useKeySubscriptionBinding(keyId: string | null) {
  return useQuery({
    queryKey: ['key-subscription-binding', keyId],
    enabled: !!keyId,
    queryFn: () =>
      apiClient<{ data: KeySubscriptionBinding | null }>(`/keys/${keyId}/subscription-binding`),
  })
}

export function useCreateSubscriptionPackage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Partial<SubscriptionPackage>) =>
      apiClient<SubscriptionPackage>('/admin/subscription-packages', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['subscription-packages'] })
    },
  })
}

export function useUpdateSubscriptionPackage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: Partial<SubscriptionPackage> & { id: string }) =>
      apiClient<SubscriptionPackage>(`/admin/subscription-packages/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['subscription-packages'] })
    },
  })
}

export function useDeleteSubscriptionPackage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiClient(`/admin/subscription-packages/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['subscription-packages'] })
      qc.invalidateQueries({ queryKey: ['subscription-plans'] })
    },
  })
}

export function useUploadPackageCover() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, file }: { id: string; file: File }) => {
      const form = new FormData()
      form.append('cover', file)
      return apiClient<SubscriptionPackage>(`/admin/subscription-packages/${id}/cover`, {
        method: 'POST',
        body: form,
      })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['subscription-packages'] })
    },
  })
}

export function useCreateSubscriptionPlan() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Partial<SubscriptionPlan> & { package_id: string; name: string }) =>
      apiClient<SubscriptionPlan>('/admin/subscription-plans', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['subscription-plans'] })
    },
  })
}

export function useUpdateSubscriptionPlan() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: Partial<SubscriptionPlan> & { id: string }) =>
      apiClient<SubscriptionPlan>(`/admin/subscription-plans/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['subscription-plans'] })
    },
  })
}

export function useDeleteSubscriptionPlan() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiClient(`/admin/subscription-plans/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['subscription-plans'] })
    },
  })
}

export function useGrantUserSubscription() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { user_id: string; plan_id: string; days?: number }) =>
      apiClient<UserSubscription>('/admin/user-subscriptions', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['my-subscriptions'] })
    },
  })
}

export function useBindKeySubscription() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ keyId, userSubscriptionId }: { keyId: string; userSubscriptionId: string }) =>
      apiClient<{ data: KeySubscriptionBinding }>(`/keys/${keyId}/subscription-binding`, {
        method: 'POST',
        body: JSON.stringify({ user_subscription_id: userSubscriptionId }),
      }),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['key-subscription-binding', vars.keyId] })
      qc.invalidateQueries({ queryKey: ['subscription-plans'] })
    },
  })
}

export interface PublicSubscriptionPlan extends SubscriptionPlan {
  slots_remaining?: number
}

export interface PublicSubscriptionPackage extends SubscriptionPackage {
  min_price?: number
  model_count?: number
  subscriber_count?: number
  plans: PublicSubscriptionPlan[]
}

export function computeDaysRemaining(expiresAt: string): number | undefined {
  const ms = Date.parse(expiresAt)
  if (Number.isNaN(ms)) return undefined
  const days = Math.floor((ms - Date.now()) / (24 * 60 * 60 * 1000))
  return Math.max(0, days)
}

export function enrichPublicPackage(pkg: PublicSubscriptionPackage): PublicSubscriptionPackage {
  const modelSet = new Set<string>()
  let minPrice = pkg.min_price
  for (const plan of pkg.plans) {
    if (minPrice === undefined || plan.price < minPrice) minPrice = plan.price
    for (const m of plan.allowed_models ?? []) {
      if (m) modelSet.add(m)
    }
  }
  return {
    ...pkg,
    min_price: minPrice ?? pkg.plans[0]?.price ?? 0,
    model_count: pkg.model_count ?? modelSet.size,
    subscriber_count: pkg.subscriber_count ?? 0,
  }
}

export function buildPublicPlanLookup(
  catalog: PublicSubscriptionPackage[],
): Map<string, PublicSubscriptionPlan> {
  const map = new Map<string, PublicSubscriptionPlan>()
  for (const pkg of catalog) {
    for (const plan of pkg.plans) {
      map.set(plan.id, plan)
    }
  }
  return map
}

export function enrichUserSubscription(
  sub: UserSubscription,
  planLookup?: Map<string, PublicSubscriptionPlan>,
): UserSubscription {
  const out: UserSubscription = { ...sub }
  if (out.days_remaining === undefined) {
    const days = computeDaysRemaining(out.expires_at)
    if (days !== undefined) out.days_remaining = days
  }
  if (!out.plan && planLookup) {
    const pub = planLookup.get(out.plan_id)
    if (pub) {
      out.plan = {
        validity_days: pub.validity_days,
        daily_token_limit: pub.daily_token_limit,
        monthly_token_limit: pub.monthly_token_limit,
        daily_request_limit: pub.daily_request_limit,
        monthly_request_limit: pub.monthly_request_limit,
        requests_per_minute: pub.requests_per_minute,
        requests_per_day: pub.requests_per_day,
        allowed_models: pub.allowed_models ?? [],
        quota_exceeded_policy: pub.quota_exceeded_policy,
      }
    }
  }
  return out
}

export function usePublicSubscriptionCatalog() {
  return useQuery({
    queryKey: ['public-subscription-packages'],
    queryFn: async () => {
      const res = await fetch('/api/v1/public/subscription-packages')
      if (!res.ok) throw new Error('Failed to load subscription catalog')
      const raw = (await res.json()) as { data: PublicSubscriptionPackage[] }
      return { data: raw.data.map(enrichPublicPackage) }
    },
    staleTime: 60_000,
  })
}

export function useCreateSubscriptionOrder() {
  return useMutation({
    mutationFn: (planId: string) =>
      apiClient<SepayOrderLike>('/me/subscription-orders', {
        method: 'POST',
        body: JSON.stringify({ plan_id: planId }),
      }),
  })
}

export interface SepayOrderLike {
  payment_method?: 'wallet' | 'sepay'
  trade_no?: string
  pay_amount: number
  credit_amount?: number
  bonus_amount?: number
  bank_code?: string
  bank_name?: string
  account_number?: string
  account_name?: string
  qr_url?: string
  expires_at?: string
  status: string
  subscription?: UserSubscription
}

export function useReleaseKeySubscription() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (keyId: string) =>
      apiClient(`/keys/${keyId}/subscription-binding`, { method: 'DELETE' }),
    onSuccess: (_data, keyId) => {
      qc.invalidateQueries({ queryKey: ['key-subscription-binding', keyId] })
      qc.invalidateQueries({ queryKey: ['subscription-plans'] })
    },
  })
}