import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'
import type { SiteAnnouncement } from '../lib/announcements'

export interface SiteConfig {
  system_name: string
  logo: string
  server_address: string
  footer: string
  about: string
  home_page_content: string
  user_agreement: string
  privacy_policy: string
  announcements: SiteAnnouncement[]
  notice_enabled: boolean
  register_enabled: boolean
  user_agreement_enabled: boolean
  privacy_policy_enabled: boolean
}

export type SiteConfigUpdate = Partial<{
  system_name: string
  logo: string
  server_address: string
  footer: string
  about: string
  home_page_content: string
  user_agreement: string
  privacy_policy: string
  announcements: SiteAnnouncement[]
  notice_enabled: boolean
  register_enabled: boolean
}>

const SITE_QUERY_KEY = ['site-config'] as const

export async function fetchPublicSiteConfig(): Promise<SiteConfig> {
  const res = await fetch('/api/v1/public/site')
  if (!res.ok) {
    throw new Error('Failed to load site configuration')
  }
  return res.json() as Promise<SiteConfig>
}

export function useSiteConfig() {
  return useQuery({
    queryKey: SITE_QUERY_KEY,
    queryFn: fetchPublicSiteConfig,
    staleTime: 60_000,
    retry: 2,
  })
}

/** Admin settings page — uses the authenticated endpoint (requires system_admin). */
export function useAdminSiteSettings() {
  return useQuery({
    queryKey: [...SITE_QUERY_KEY, 'admin'] as const,
    queryFn: () => apiClient<SiteConfig>('/admin/settings/site'),
    staleTime: 30_000,
    retry: 1,
  })
}

export function useUpdateSiteConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: SiteConfigUpdate) =>
      apiClient<SiteConfig>('/admin/settings/site', {
        method: 'PUT',
        body: JSON.stringify(payload),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData(SITE_QUERY_KEY, data)
    },
  })
}