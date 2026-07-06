import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export type BackupType = 'config' | 'full'

export interface BackupS3Config {
  endpoint: string
  region: string
  bucket: string
  access_key_id: string
  secret_access_key?: string
  secret_configured: boolean
  prefix: string
  force_path_style: boolean
}

export interface BackupScheduleConfig {
  enabled: boolean
  start_hour: number
  start_minute: number
  backups_per_day: number
  backup_type: BackupType
  retain_days: number
  retain_count: number
}

export interface BackupRecord {
  id: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  backup_type: BackupType
  file_name: string
  s3_key: string
  size_bytes: number
  triggered_by: string
  error_message?: string
  started_at: string
  finished_at?: string
  progress?: string
  restore_status?: string
  restore_error?: string
  restored_at?: string
}

export interface SettingsExportPayload {
  exported_at: string
  version: number
  site: unknown
  security: unknown
  payment: unknown
  features: unknown
}

const backupKeys = {
  s3: ['backup', 's3'] as const,
  schedule: ['backup', 'schedule'] as const,
  list: ['backup', 'list'] as const,
}

export function useBackupS3Config() {
  return useQuery({
    queryKey: backupKeys.s3,
    queryFn: () => apiClient<BackupS3Config>('/admin/backups/s3-config'),
  })
}

export function useUpdateBackupS3Config() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (cfg: BackupS3Config) =>
      apiClient<BackupS3Config>('/admin/backups/s3-config', {
        method: 'PUT',
        body: JSON.stringify(cfg),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: backupKeys.s3 }),
  })
}

export function useTestBackupS3() {
  return useMutation({
    mutationFn: (cfg: BackupS3Config) =>
      apiClient<{ ok: boolean; message: string }>('/admin/backups/s3-config/test', {
        method: 'POST',
        body: JSON.stringify(cfg),
      }),
  })
}

export function useBackupSchedule() {
  return useQuery({
    queryKey: backupKeys.schedule,
    queryFn: () =>
      apiClient<{ schedule: BackupScheduleConfig; slots: string[] }>('/admin/backups/schedule'),
  })
}

export function useUpdateBackupSchedule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (schedule: BackupScheduleConfig) =>
      apiClient<{ schedule: BackupScheduleConfig; slots: string[] }>('/admin/backups/schedule', {
        method: 'PUT',
        body: JSON.stringify(schedule),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: backupKeys.schedule }),
  })
}

export function useBackupList() {
  return useQuery({
    queryKey: backupKeys.list,
    queryFn: () => apiClient<{ items: BackupRecord[] }>('/admin/backups'),
    refetchInterval: (query) => {
      const items = query.state.data?.items ?? []
      const active = items.some(
        (r) =>
          r.status === 'running' ||
          r.restore_status === 'running' ||
          r.status === 'pending',
      )
      return active ? 2000 : false
    },
  })
}

export function useCreateBackup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (backup_type: BackupType) =>
      apiClient<BackupRecord>('/admin/backups', {
        method: 'POST',
        body: JSON.stringify({ backup_type }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: backupKeys.list }),
  })
}

export function useDeleteBackup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/admin/backups/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: backupKeys.list }),
  })
}

export function useBackupDownloadURL() {
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<{ url: string }>(`/admin/backups/${id}/download-url`),
  })
}

export function useRestoreBackup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (args: { id: string; password: string; pre_backup?: boolean }) =>
      apiClient<{ status: string }>(`/admin/backups/${args.id}/restore`, {
        method: 'POST',
        body: JSON.stringify({ password: args.password, pre_backup: args.pre_backup ?? true }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: backupKeys.list }),
  })
}

export function useExportAdminSettings() {
  return useMutation({
    mutationFn: () => apiClient<SettingsExportPayload>('/admin/settings/export'),
  })
}

export function usePreviewSettingsImport() {
  return useMutation({
    mutationFn: (payload: SettingsExportPayload) =>
      apiClient<{ changed: string[] }>('/admin/settings/import/preview', {
        method: 'POST',
        body: JSON.stringify({ payload, confirm: false }),
      }),
  })
}

export function useImportAdminSettings() {
  return useMutation({
    mutationFn: (payload: SettingsExportPayload) =>
      apiClient<{ ok: boolean }>('/admin/settings/import', {
        method: 'POST',
        body: JSON.stringify({ payload, confirm: true }),
      }),
  })
}