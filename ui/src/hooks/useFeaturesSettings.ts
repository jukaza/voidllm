import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface WalletFeatures {
  enforce_balance: boolean
  initial_balance_vnd: number
}

export interface ModuleFeatures {
  public_catalog: boolean
  playground: boolean
}

export interface FeaturesSettings {
  wallet: WalletFeatures
  modules: ModuleFeatures
}

export interface PublicFeatures {
  wallet: WalletFeatures
  modules: ModuleFeatures
}

const FEATURES_QUERY_KEY = ['features-settings'] as const

export function useAdminFeaturesSettings() {
  return useQuery({
    queryKey: [...FEATURES_QUERY_KEY, 'admin'] as const,
    queryFn: () => apiClient<FeaturesSettings>('/admin/settings/features'),
    staleTime: 30_000,
  })
}

export function useUpdateFeaturesSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: Partial<FeaturesSettings>) =>
      apiClient<FeaturesSettings>('/admin/settings/features', {
        method: 'PUT',
        body: JSON.stringify(payload),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData([...FEATURES_QUERY_KEY, 'admin'], data)
      queryClient.setQueryData([...FEATURES_QUERY_KEY, 'public'], {
        wallet: data.wallet,
        modules: data.modules,
      } satisfies PublicFeatures)
    },
  })
}

export function usePublicFeatures() {
  return useQuery({
    queryKey: [...FEATURES_QUERY_KEY, 'public'] as const,
    queryFn: async () => {
      const res = await fetch('/api/v1/public/features')
      if (!res.ok) throw new Error('Failed to load feature settings')
      return res.json() as Promise<PublicFeatures>
    },
    staleTime: 60_000,
  })
}