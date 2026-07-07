import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

import type { SiteAnnouncement } from '../lib/announcements'
import { LOCAL_STORAGE_KEY } from '../lib/constants'

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
  site_subtitle: string
  support_zalo: string
  support_telegram: string
  doc_url: string
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
  site_subtitle: string
  support_zalo: string
  support_telegram: string
  doc_url: string
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

function setSiteConfigCache(queryClient: ReturnType<typeof useQueryClient>, data: SiteConfig) {
  queryClient.setQueryData(SITE_QUERY_KEY, data)
  queryClient.setQueryData([...SITE_QUERY_KEY, 'admin'], data)
}

async function siteLogoRequest<T>(endpoint: string, init?: RequestInit): Promise<T> {
  const key = localStorage.getItem(LOCAL_STORAGE_KEY) ?? ''
  const res = await fetch(`/api/v1${endpoint}`, {
    ...init,
    headers: {
      Authorization: `Bearer ${key}`,
      ...init?.headers,
    },
  })

  if (res.status === 401) {
    throw new Error('Session expired — please log in again')
  }

  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: { message?: string } } | null
    throw new Error(body?.error?.message ?? res.statusText)
  }

  return res.json() as Promise<T>
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
      setSiteConfigCache(queryClient, data)
    },
  })
}

export function useUploadSiteLogo() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (file: File) => {
      const form = new FormData()
      form.append('logo', file)
      return siteLogoRequest<SiteConfig>('/admin/settings/site/logo', {
        method: 'POST',
        body: form,
      })
    },
    onSuccess: (data) => {
      setSiteConfigCache(queryClient, data)
    },
  })
}

export function useResetSiteLogo() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () =>
      siteLogoRequest<SiteConfig>('/admin/settings/site/logo', {
        method: 'DELETE',
      }),
    onSuccess: (data) => {
      setSiteConfigCache(queryClient, data)
    },
  })
}