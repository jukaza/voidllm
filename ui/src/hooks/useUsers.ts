import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export type UserRole = 'member' | 'admin' | 'root'
export type UserStatus = 'active' | 'disabled'

export interface UserResponse {
  id: string
  email: string
  display_name: string
  auth_provider: string
  role: UserRole
  status: UserStatus
  is_system_admin: boolean
  created_at: string
  updated_at: string
  deleted_at?: string | null
}

export interface CreateUserParams {
  email: string
  display_name: string
  password: string
  role?: UserRole
}

export interface UpdateUserParams {
  email?: string
  display_name?: string
  password?: string
  role?: UserRole
  status?: UserStatus
}

export interface UsersListParams {
  cursor?: string
  search?: string
  role?: UserRole
  status?: UserStatus
}

export interface PaginatedUsers {
  data: UserResponse[]
  has_more: boolean
  next_cursor?: string
}

export interface UserWallet {
  balance: number
  currency: string
}

export interface AdjustWalletParams {
  amount: number
  description?: string
}

export interface OAuthConnection {
  provider: string
  label?: string
  linked: boolean
}

function buildUsersQuery(params?: UsersListParams) {
  const qs = new URLSearchParams({ limit: '20' })
  if (params?.cursor) qs.set('cursor', params.cursor)
  if (params?.search) qs.set('search', params.search)
  if (params?.role) qs.set('role', params.role)
  if (params?.status) qs.set('status', params.status)
  return `/users?${qs.toString()}`
}

export function useUser(userId: string) {
  return useQuery({
    queryKey: ['user', userId],
    queryFn: () => apiClient<UserResponse>(`/users/${userId}`),
    enabled: !!userId,
    staleTime: 5 * 60 * 1000,
  })
}

export function useUsers(params?: UsersListParams) {
  return useQuery({
    queryKey: ['users-list', params],
    queryFn: () => apiClient<PaginatedUsers>(buildUsersQuery(params)),
  })
}

export function useUserWallet(userId: string | null) {
  return useQuery({
    queryKey: ['user-wallet', userId],
    queryFn: () => apiClient<UserWallet>(`/users/${userId}/wallet`),
    enabled: !!userId,
  })
}

export function useUserConnections(userId: string | null) {
  return useQuery({
    queryKey: ['user-connections', userId],
    queryFn: () =>
      apiClient<{ connections: OAuthConnection[] }>(
        `/admin/users/${userId}/connections`,
      ),
    enabled: !!userId,
  })
}

export function useCreateUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: CreateUserParams) =>
      apiClient<UserResponse>('/users', {
        method: 'POST',
        body: JSON.stringify(params),
      }),
    onSuccess: (user) => {
      queryClient.setQueryData(['user', user.id], user)
      queryClient.invalidateQueries({ queryKey: ['users-list'] })
    },
  })
}

export function useUpdateUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, params }: { userId: string; params: UpdateUserParams }) =>
      apiClient<UserResponse>(`/users/${userId}`, {
        method: 'PATCH',
        body: JSON.stringify(params),
      }),
    onSuccess: (user) => {
      queryClient.setQueryData(['user', user.id], user)
      queryClient.invalidateQueries({ queryKey: ['users-list'] })
    },
  })
}

export function useManageUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, action }: { userId: string; action: 'enable' | 'disable' }) =>
      apiClient<UserResponse>(`/users/${userId}/manage`, {
        method: 'POST',
        body: JSON.stringify({ action }),
      }),
    onSuccess: (user) => {
      queryClient.setQueryData(['user', user.id], user)
      queryClient.invalidateQueries({ queryKey: ['users-list'] })
    },
  })
}

export function useDeleteUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) =>
      apiClient<void>(`/users/${userId}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['users-list'] }),
  })
}

export function useAdjustUserWallet() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, params }: { userId: string; params: AdjustWalletParams }) =>
      apiClient<{ balance: number }>(`/users/${userId}/wallet/adjust`, {
        method: 'POST',
        body: JSON.stringify(params),
      }),
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({ queryKey: ['user-wallet', vars.userId] })
      queryClient.invalidateQueries({ queryKey: ['users-list'] })
    },
  })
}

export function useRevokeUserSessions() {
  return useMutation({
    mutationFn: (userId: string) =>
      apiClient<void>(`/admin/users/${userId}/sessions`, { method: 'DELETE' }),
  })
}