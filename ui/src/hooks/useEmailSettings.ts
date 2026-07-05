import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface SMTPConfig {
  enabled: boolean
  host: string
  port: number
  username: string
  password?: string
  from: string
  ssl_enabled: boolean
}

export interface EmailSettings {
  enabled: boolean
  host: string
  port: number
  username: string
  password?: string
  from: string
  ssl_enabled: boolean
  password_configured: boolean
}

const EMAIL_QUERY_KEY = ['email-settings'] as const

export function useAdminEmailSettings() {
  return useQuery({
    queryKey: [...EMAIL_QUERY_KEY, 'admin'] as const,
    queryFn: () => apiClient<EmailSettings>('/admin/settings/email'),
    staleTime: 30_000,
  })
}

export function useUpdateEmailSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: { smtp: Partial<SMTPConfig> }) =>
      apiClient<EmailSettings>('/admin/settings/email', {
        method: 'PUT',
        body: JSON.stringify(payload),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData([...EMAIL_QUERY_KEY, 'admin'], data)
    },
  })
}