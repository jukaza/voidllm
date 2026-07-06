import { useQuery } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface MeResponse {
  id: string
  email: string
  display_name: string
  role: string
  is_system_admin: boolean
  has_password: boolean
  auth_provider: string
  active_session_count: number
}

export function useMe() {
  return useQuery({
    queryKey: ['me'],
    queryFn: () => apiClient<MeResponse>('/me'),
    staleTime: 5 * 60 * 1000,
  })
}