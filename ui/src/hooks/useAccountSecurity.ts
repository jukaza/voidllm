import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface SessionItem {
  id: string
  ip: string
  user_agent?: string
  device_label: string
  created_at: string
  last_seen_at?: string
  current: boolean
}

const SESSIONS_KEY = ['me-sessions'] as const

export function useSessions() {
  return useQuery({
    queryKey: SESSIONS_KEY,
    queryFn: async () => {
      const res = await apiClient<{ sessions: SessionItem[] }>('/me/sessions')
      return res.sessions
    },
    staleTime: 30_000,
  })
}

export function useRevokeSession() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (sessionId: string) =>
      apiClient<void>(`/me/sessions/${sessionId}`, { method: 'DELETE' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: SESSIONS_KEY })
      void queryClient.invalidateQueries({ queryKey: ['me'] })
    },
  })
}

export function useRevokeOtherSessions() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => apiClient<void>('/me/sessions', { method: 'DELETE' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: SESSIONS_KEY })
      void queryClient.invalidateQueries({ queryKey: ['me'] })
    },
  })
}

export function useTwoFASetup() {
  return useMutation({
    mutationFn: () =>
      apiClient<{ secret: string; otpauth_url: string }>('/me/2fa/setup', { method: 'POST' }),
  })
}

export function useTwoFAVerify() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (code: string) =>
      apiClient<{ backup_codes: string[] }>('/me/2fa/verify', {
        method: 'POST',
        body: JSON.stringify({ code }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['me'] })
    },
  })
}

export function useTwoFADisable() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: { password?: string; code?: string }) =>
      apiClient<void>('/me/2fa', {
        method: 'DELETE',
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['me'] })
    },
  })
}

export function useSetPassword() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (new_password: string) =>
      apiClient<void>('/me/password/set', {
        method: 'POST',
        body: JSON.stringify({ new_password }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['me'] })
    },
  })
}