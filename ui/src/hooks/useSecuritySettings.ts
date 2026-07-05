import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface OAuthProviderConfig {
  enabled: boolean
  allow_login: boolean
  allow_signup: boolean
  client_id: string
  client_secret_configured: boolean
}

export interface SecurityConfig {
  turnstile: {
    enabled: boolean
    site_key: string
    secret_key_configured: boolean
  }
  oauth: {
    google: OAuthProviderConfig
    github: OAuthProviderConfig
  }
  two_fa: {
    allow_user_enable: boolean
    require_system_admin: boolean
  }
  session: {
    ttl_hours: number
    allow_multiple: boolean
    max_concurrent: number
  }
  password: {
    min_length: number
    allow_oauth_set_password: boolean
  }
  oauth_callback_urls?: {
    google?: string
    github?: string
  }
}

export type SecurityConfigUpdate = Partial<{
  turnstile: Partial<{
    enabled: boolean
    site_key: string
    secret_key: string
  }>
  oauth: Partial<{
    google: Partial<OAuthProviderConfig & { client_secret?: string }>
    github: Partial<OAuthProviderConfig & { client_secret?: string }>
  }>
  two_fa: Partial<{
    allow_user_enable: boolean
    require_system_admin: boolean
  }>
  session: Partial<{
    ttl_hours: number
    allow_multiple: boolean
    max_concurrent: number
  }>
  password: Partial<{
    min_length: number
    allow_oauth_set_password: boolean
  }>
}>

const SECURITY_QUERY_KEY = ['security-settings'] as const

export function useSecuritySettings() {
  return useQuery({
    queryKey: SECURITY_QUERY_KEY,
    queryFn: () => apiClient<SecurityConfig>('/admin/settings/security'),
    staleTime: 30_000,
  })
}

export function useUpdateSecuritySettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: SecurityConfigUpdate) =>
      apiClient<SecurityConfig>('/admin/settings/security', {
        method: 'PUT',
        body: JSON.stringify(payload),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData(SECURITY_QUERY_KEY, data)
    },
  })
}

export interface PublicAuthConfig {
  local: boolean
  register_enabled: boolean
  turnstile: { enabled: boolean; site_key?: string }
  oauth: {
    google: { enabled: boolean; login: boolean; signup: boolean }
    github: { enabled: boolean; login: boolean; signup: boolean }
  }
  two_fa?: { available: boolean }
}

const AUTH_CONFIG_KEY = ['public-auth-config'] as const

export async function fetchPublicAuthConfig(): Promise<PublicAuthConfig> {
  const res = await fetch('/api/v1/public/auth-config')
  if (!res.ok) throw new Error('Failed to load auth configuration')
  return res.json() as Promise<PublicAuthConfig>
}

export function usePublicAuthConfig() {
  return useQuery({
    queryKey: AUTH_CONFIG_KEY,
    queryFn: fetchPublicAuthConfig,
    staleTime: 60_000,
  })
}

export function oauthStartUrl(
  provider: string,
  mode: 'login' | 'signup' | 'bind',
  opts?: { acceptTerms?: boolean },
) {
  const params = new URLSearchParams({ mode })
  if (mode === 'signup' && opts?.acceptTerms) {
    params.set('accept_terms', '1')
  }
  return `/api/v1/auth/oauth/${provider}?${params.toString()}`
}

export interface ConnectionsResponse {
  google: { linked: boolean; label?: string }
  github: { linked: boolean; label?: string }
}

const CONNECTIONS_QUERY_KEY = ['me-connections'] as const

export function useMeConnections() {
  return useQuery({
    queryKey: CONNECTIONS_QUERY_KEY,
    queryFn: () => apiClient<ConnectionsResponse>('/me/connections'),
    staleTime: 30_000,
  })
}

export function useDeleteConnection() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (provider: string) =>
      apiClient<void>(`/me/connections/${provider}`, { method: 'DELETE' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: CONNECTIONS_QUERY_KEY })
    },
  })
}