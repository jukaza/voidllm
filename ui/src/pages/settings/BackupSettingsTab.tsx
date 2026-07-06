import { useEffect, useRef, useState } from 'react'
import { Button } from '../../components/ui/Button'
import { Banner } from '../../components/ui/Banner'
import { Input } from '../../components/ui/Input'
import { Select } from '../../components/ui/Select'
import { Toggle } from '../../components/ui/Toggle'
import { useToast } from '../../hooks/useToast'
import {
  useBackupDownloadURL,
  useBackupList,
  useBackupSchedule,
  useBackupS3Config,
  useCreateBackup,
  useDeleteBackup,
  useExportAdminSettings,
  useImportAdminSettings,
  usePreviewSettingsImport,
  useRestoreBackup,
  useTestBackupS3,
  useUpdateBackupS3Config,
  useUpdateBackupSchedule,
  type BackupRecord,
  type BackupS3Config,
  type BackupScheduleConfig,
  type BackupType,
  type SettingsExportPayload,
} from '../../hooks/useBackupSettings'
import { useTranslation } from '../../lib/i18n'
import { BackupR2SetupGuide } from './components/BackupR2SetupGuide'
import { LiveBadge } from './components/PreviewBadge'
import { SettingsSectionCard } from './components/SettingsSectionCard'

/** Default SQLite path in tavo.dev.yaml / tavo.yaml */
const DEFAULT_SQLITE_PATH = './tavo.db'

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

function BackupRow({
  record,
  onDownload,
  onRestore,
  onDelete,
}: {
  record: BackupRecord
  onDownload: (id: string) => void
  onRestore: (record: BackupRecord) => void
  onDelete: (id: string) => void
}) {
  const { t } = useTranslation()
  return (
    <tr className="border-b border-border text-sm">
      <td className="py-2 pr-3">{new Date(record.started_at).toLocaleString()}</td>
      <td className="py-2 pr-3">
        <span className="rounded bg-bg-tertiary px-1.5 py-0.5 text-xs font-medium">
          {record.backup_type === 'full' ? t('settings.backup_type_full') : t('settings.backup_type_config')}
        </span>
      </td>
      <td className="py-2 pr-3">{formatBytes(record.size_bytes)}</td>
      <td className="py-2 pr-3">{record.triggered_by}</td>
      <td className="py-2 pr-3">
        {record.status}
        {record.progress ? ` (${record.progress})` : ''}
        {record.error_message ? ` — ${record.error_message}` : ''}
        {record.restore_status ? ` / restore: ${record.restore_status}` : ''}
      </td>
      <td className="py-2">
        <div className="flex flex-wrap gap-1">
          {record.status === 'completed' && (
            <>
              <Button size="sm" variant="ghost" onClick={() => onDownload(record.id)}>
                {t('settings.backup_download')}
              </Button>
              <Button size="sm" variant="ghost" onClick={() => onRestore(record)}>
                {t('settings.backup_restore')}
              </Button>
            </>
          )}
          <Button size="sm" variant="ghost" onClick={() => onDelete(record.id)}>
            {t('common.delete')}
          </Button>
        </div>
      </td>
    </tr>
  )
}

export function BackupSettingsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const fileRef = useRef<HTMLInputElement>(null)

  const { data: s3Data } = useBackupS3Config()
  const { data: scheduleData } = useBackupSchedule()
  const { data: listData } = useBackupList()

  const updateS3 = useUpdateBackupS3Config()
  const testS3 = useTestBackupS3()
  const updateSchedule = useUpdateBackupSchedule()
  const createBackup = useCreateBackup()
  const deleteBackup = useDeleteBackup()
  const downloadURL = useBackupDownloadURL()
  const restoreBackup = useRestoreBackup()
  const exportSettings = useExportAdminSettings()
  const previewImport = usePreviewSettingsImport()
  const importSettings = useImportAdminSettings()

  const [s3Draft, setS3Draft] = useState<BackupS3Config | null>(null)
  const [scheduleDraft, setScheduleDraft] = useState<BackupScheduleConfig | null>(null)
  const [backupType, setBackupType] = useState<BackupType>('config')
  const [importPayload, setImportPayload] = useState<SettingsExportPayload | null>(null)
  const [importChanged, setImportChanged] = useState<string[]>([])

  useEffect(() => {
    if (s3Data && !s3Draft) setS3Draft(s3Data)
  }, [s3Data, s3Draft])

  useEffect(() => {
    if (scheduleData?.schedule && !scheduleDraft) setScheduleDraft(scheduleData.schedule)
  }, [scheduleData, scheduleDraft])

  async function handleExportSettings() {
    try {
      const data = await exportSettings.mutateAsync()
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `tavo-settings-${new Date().toISOString().slice(0, 10)}.json`
      a.click()
      URL.revokeObjectURL(url)
      toast({ variant: 'success', message: t('settings.backup_exported') })
    } catch (err) {
      toast({
        variant: 'error',
        message: err instanceof Error ? err.message : t('settings.backup_export_error'),
      })
    }
  }

  function handleImportFile(file: File) {
    const reader = new FileReader()
    reader.onload = () => {
      try {
        const parsed = JSON.parse(String(reader.result)) as SettingsExportPayload
        setImportPayload(parsed)
        void previewImport
          .mutateAsync(parsed)
          .then((res) => {
            setImportChanged(res.changed)
            toast({ variant: 'info', message: t('settings.backup_import_ready') })
          })
          .catch((err: Error) => {
            toast({ variant: 'error', message: err.message })
          })
      } catch {
        toast({ variant: 'error', message: t('settings.backup_import_invalid') })
      }
    }
    reader.readAsText(file)
  }

  async function handleConfirmImport() {
    if (!importPayload) return
    try {
      await importSettings.mutateAsync(importPayload)
      setImportPayload(null)
      setImportChanged([])
      toast({ variant: 'success', message: t('settings.backup_import_done') })
    } catch (err) {
      toast({
        variant: 'error',
        message: err instanceof Error ? err.message : t('settings.backup_import_error'),
      })
    }
  }

  async function handleSaveS3() {
    if (!s3Draft) return
    try {
      await updateS3.mutateAsync(s3Draft)
      toast({ variant: 'success', message: t('settings.saved') })
    } catch (err) {
      toast({ variant: 'error', message: err instanceof Error ? err.message : t('settings.save_error') })
    }
  }

  async function handleTestS3() {
    if (!s3Draft) return
    try {
      const res = await testS3.mutateAsync(s3Draft)
      toast({
        variant: res.ok ? 'success' : 'error',
        message: res.message,
      })
    } catch (err) {
      toast({ variant: 'error', message: err instanceof Error ? err.message : t('settings.backup_s3_test_fail') })
    }
  }

  async function handleSaveSchedule() {
    if (!scheduleDraft) return
    try {
      await updateSchedule.mutateAsync(scheduleDraft)
      toast({ variant: 'success', message: t('settings.saved') })
    } catch (err) {
      toast({ variant: 'error', message: err instanceof Error ? err.message : t('settings.save_error') })
    }
  }

  async function handleBackupNow() {
    try {
      await createBackup.mutateAsync(backupType)
      toast({ variant: 'success', message: t('settings.backup_started') })
    } catch (err) {
      toast({ variant: 'error', message: err instanceof Error ? err.message : t('settings.backup_start_error') })
    }
  }

  async function handleDownload(id: string) {
    try {
      const res = await downloadURL.mutateAsync(id)
      window.open(res.url, '_blank')
    } catch (err) {
      toast({ variant: 'error', message: err instanceof Error ? err.message : t('settings.backup_download_error') })
    }
  }

  function handleRestore(record: BackupRecord) {
    const ok = window.confirm(
      record.backup_type === 'full'
        ? t('settings.backup_restore_confirm_full')
        : t('settings.backup_restore_confirm_config'),
    )
    if (!ok) return
    const password = window.prompt(t('settings.backup_restore_password'))
    if (!password) return
    void restoreBackup
      .mutateAsync({ id: record.id, password })
      .then(() => toast({ variant: 'success', message: t('settings.backup_restore_started') }))
      .catch((err: Error) => toast({ variant: 'error', message: err.message }))
  }

  function handleDelete(id: string) {
    if (!window.confirm(t('settings.backup_delete_confirm'))) return
    void deleteBackup
      .mutateAsync(id)
      .then(() => toast({ variant: 'success', message: t('settings.backup_deleted') }))
      .catch((err: Error) => toast({ variant: 'error', message: err.message }))
  }

  function copyDbCommand(cmd: string) {
    void navigator.clipboard.writeText(cmd)
    toast({ variant: 'success', message: t('common.copied') })
  }

  const slots = scheduleData?.slots ?? []

  return (
    <div className="space-y-6">
      <SettingsSectionCard
        title={t('settings.backup_export_title')}
        description={t('settings.backup_export_desc')}
        badge={<LiveBadge />}
      >
        <p className="mb-3 text-sm text-text-secondary">{t('settings.backup_export_note')}</p>
        <div className="flex flex-wrap gap-2">
          <Button variant="secondary" onClick={() => void handleExportSettings()}>
            {t('settings.backup_export_btn')}
          </Button>
          <input
            ref={fileRef}
            type="file"
            accept="application/json,.json"
            className="hidden"
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (file) handleImportFile(file)
              e.target.value = ''
            }}
          />
          <Button variant="secondary" onClick={() => fileRef.current?.click()}>
            {t('settings.backup_import_btn')}
          </Button>
        </div>
        {importPayload && (
          <div className="mt-4 rounded-md border border-border bg-bg-tertiary p-3">
            <p className="text-sm text-text-secondary">{t('settings.backup_import_confirm_hint')}</p>
            <p className="mt-1 text-xs text-text-tertiary">
              {t('settings.backup_import_domains')}: {importChanged.join(', ')}
            </p>
            <Button className="mt-3" size="sm" onClick={() => void handleConfirmImport()}>
              {t('settings.backup_import_confirm')}
            </Button>
          </div>
        )}
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.backup_s3_title')}
        description={t('settings.backup_s3_desc')}
        badge={<LiveBadge />}
      >
        <BackupR2SetupGuide />
        {s3Draft && (
          <div className="grid gap-3 sm:grid-cols-2">
            <Input
              label={t('settings.backup_s3_endpoint')}
              description={t('settings.backup_s3_endpoint_hint')}
              value={s3Draft.endpoint}
              onChange={(e) => setS3Draft({ ...s3Draft, endpoint: e.target.value })}
            />
            <Input
              label={t('settings.backup_s3_region')}
              description={t('settings.backup_s3_region_hint')}
              value={s3Draft.region}
              onChange={(e) => setS3Draft({ ...s3Draft, region: e.target.value })}
            />
            <Input
              label={t('settings.backup_s3_bucket')}
              description={t('settings.backup_s3_bucket_hint')}
              value={s3Draft.bucket}
              onChange={(e) => setS3Draft({ ...s3Draft, bucket: e.target.value })}
            />
            <Input
              label={t('settings.backup_s3_prefix')}
              description={t('settings.backup_s3_prefix_hint')}
              value={s3Draft.prefix}
              onChange={(e) => setS3Draft({ ...s3Draft, prefix: e.target.value })}
            />
            <Input
              label={t('settings.backup_s3_access_key')}
              description={t('settings.backup_s3_access_key_hint')}
              value={s3Draft.access_key_id}
              onChange={(e) => setS3Draft({ ...s3Draft, access_key_id: e.target.value })}
            />
            <Input
              label={t('settings.backup_s3_secret')}
              description={t('settings.backup_s3_secret_hint')}
              type="password"
              placeholder={s3Draft.secret_configured ? '••••••••' : ''}
              onChange={(e) => setS3Draft({ ...s3Draft, secret_access_key: e.target.value })}
            />
            <Toggle
              checked={s3Draft.force_path_style}
              onChange={(v) => setS3Draft({ ...s3Draft, force_path_style: v })}
              label={t('settings.backup_s3_path_style')}
            />
          </div>
        )}
        <div className="mt-4 flex flex-wrap gap-2">
          <Button variant="secondary" onClick={() => void handleTestS3()}>
            {t('settings.backup_s3_test')}
          </Button>
          <Button onClick={() => void handleSaveS3()} loading={updateS3.isPending}>
            {t('common.save')}
          </Button>
        </div>
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.backup_schedule_title')}
        description={t('settings.backup_schedule_desc')}
        badge={<LiveBadge />}
      >
        {scheduleDraft && (
          <div className="space-y-3">
            <Toggle
              checked={scheduleDraft.enabled}
              onChange={(v) => setScheduleDraft({ ...scheduleDraft, enabled: v })}
              label={t('settings.backup_schedule_enabled')}
            />
            <div className="grid gap-3 sm:grid-cols-2">
              <Input
                label={t('settings.backup_schedule_start_hour')}
                type="number"
                min={0}
                max={23}
                value={String(scheduleDraft.start_hour)}
                onChange={(e) => setScheduleDraft({ ...scheduleDraft, start_hour: Number(e.target.value) })}
              />
              <Input
                label={t('settings.backup_schedule_start_minute')}
                type="number"
                min={0}
                max={59}
                value={String(scheduleDraft.start_minute)}
                onChange={(e) => setScheduleDraft({ ...scheduleDraft, start_minute: Number(e.target.value) })}
              />
              <Input
                label={t('settings.backup_schedule_per_day')}
                type="number"
                min={1}
                max={24}
                value={String(scheduleDraft.backups_per_day)}
                onChange={(e) => setScheduleDraft({ ...scheduleDraft, backups_per_day: Number(e.target.value) })}
              />
              <Select
                label={t('settings.backup_schedule_type')}
                value={scheduleDraft.backup_type}
                onChange={(v) => setScheduleDraft({ ...scheduleDraft, backup_type: v as BackupType })}
                options={[
                  { value: 'config', label: t('settings.backup_type_config') },
                  { value: 'full', label: t('settings.backup_type_full') },
                ]}
              />
              <Input
                label={t('settings.backup_schedule_retain_days')}
                type="number"
                min={0}
                value={String(scheduleDraft.retain_days)}
                onChange={(e) => setScheduleDraft({ ...scheduleDraft, retain_days: Number(e.target.value) })}
              />
              <Input
                label={t('settings.backup_schedule_retain_count')}
                type="number"
                min={0}
                value={String(scheduleDraft.retain_count)}
                onChange={(e) => setScheduleDraft({ ...scheduleDraft, retain_count: Number(e.target.value) })}
              />
            </div>
            {slots.length > 0 && (
              <p className="text-xs text-text-tertiary">
                {t('settings.backup_schedule_slots')}: {slots.join(', ')}
              </p>
            )}
            <Button onClick={() => void handleSaveSchedule()} loading={updateSchedule.isPending}>
              {t('common.save')}
            </Button>
          </div>
        )}
      </SettingsSectionCard>

      <SettingsSectionCard title={t('settings.backup_ops_title')} description={t('settings.backup_ops_desc')}>
        <div className="mb-4 flex flex-wrap items-end gap-3">
          <Select
            label={t('settings.backup_now_type')}
            value={backupType}
            onChange={(v) => setBackupType(v as BackupType)}
            options={[
              { value: 'config', label: t('settings.backup_type_config') },
              { value: 'full', label: t('settings.backup_type_full') },
            ]}
          />
          <Button onClick={() => void handleBackupNow()} loading={createBackup.isPending}>
            {t('settings.backup_now')}
          </Button>
        </div>
        <p className="mb-3 text-xs text-text-tertiary">{t('settings.backup_scope_hint')}</p>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-left">
            <thead>
              <tr className="border-b border-border text-xs text-text-tertiary">
                <th className="pb-2 pr-3">{t('settings.backup_col_time')}</th>
                <th className="pb-2 pr-3">{t('settings.backup_col_type')}</th>
                <th className="pb-2 pr-3">{t('settings.backup_col_size')}</th>
                <th className="pb-2 pr-3">{t('settings.backup_col_trigger')}</th>
                <th className="pb-2 pr-3">{t('settings.backup_col_status')}</th>
                <th className="pb-2">{t('settings.backup_col_actions')}</th>
              </tr>
            </thead>
            <tbody>
              {(listData?.items ?? []).map((r) => (
                <BackupRow
                  key={r.id}
                  record={r}
                  onDownload={handleDownload}
                  onRestore={handleRestore}
                  onDelete={handleDelete}
                />
              ))}
            </tbody>
          </table>
          {(listData?.items ?? []).length === 0 && (
            <p className="py-4 text-sm text-text-tertiary">{t('settings.backup_list_empty')}</p>
          )}
        </div>
      </SettingsSectionCard>

      <SettingsSectionCard title={t('settings.backup_db_title')} description={t('settings.backup_db_desc')}>
        <Banner variant="info" title={t('settings.backup_sqlite_default')} />
        <div className="mt-3 space-y-3">
          <div className="rounded-md border border-border bg-bg-tertiary p-3">
            <div className="mb-1 text-xs font-medium text-text-secondary">SQLite</div>
            <p className="mb-2 text-xs text-text-tertiary">{t('settings.backup_sqlite_path_hint')}</p>
            <code className="block break-all text-xs font-mono">
              cp {DEFAULT_SQLITE_PATH} {DEFAULT_SQLITE_PATH}.backup
            </code>
            <Button
              size="sm"
              variant="ghost"
              className="mt-2"
              onClick={() => copyDbCommand(`cp ${DEFAULT_SQLITE_PATH} ${DEFAULT_SQLITE_PATH}.backup`)}
            >
              {t('common.copy')}
            </Button>
          </div>
          <details className="rounded-md border border-border bg-bg-tertiary p-3">
            <summary className="cursor-pointer text-xs font-medium text-text-tertiary">
              {t('settings.backup_postgres_optional')}
            </summary>
            <p className="mt-2 text-xs text-text-tertiary">{t('settings.backup_postgres_hint')}</p>
            <code className="mt-2 block break-all text-xs font-mono">
              pg_dump -Fc postgres://user:pass@host:5432/tavo &gt; tavo.dump
            </code>
            <Button
              size="sm"
              variant="ghost"
              className="mt-2"
              onClick={() => copyDbCommand('pg_dump -Fc postgres://user:pass@host:5432/tavo > tavo.dump')}
            >
              {t('common.copy')}
            </Button>
          </details>
        </div>
        <Banner variant="info" className="mt-3" title={t('settings.backup_yaml_hint')} />
      </SettingsSectionCard>
    </div>
  )
}