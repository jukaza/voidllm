import { useRef, useState } from 'react'
import { Button } from '../../components/ui/Button'
import { Banner } from '../../components/ui/Banner'
import { Toggle } from '../../components/ui/Toggle'
import apiClient from '../../api/client'
import { useAdminPaymentSettings } from '../../hooks/usePaymentSettings'
import { useAdminSiteSettings } from '../../hooks/useSiteConfig'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { PreviewBadge } from './components/PreviewBadge'
import { SettingsSectionCard } from './components/SettingsSectionCard'
import { SettingsTabFooter } from './components/SettingsTabFooter'

function redactSecrets<T extends Record<string, unknown>>(obj: T): T {
  const out = { ...obj } as Record<string, unknown>
  for (const key of Object.keys(out)) {
    const val = out[key]
    if (
      typeof key === 'string' &&
      (key.includes('password') ||
        key.includes('secret') ||
        key.includes('token') ||
        key === 'webhook_token' ||
        key === 'webhook_secret')
    ) {
      if (val && String(val).length > 0) out[key] = '••••••••'
    } else if (val && typeof val === 'object' && !Array.isArray(val)) {
      out[key] = redactSecrets(val as Record<string, unknown>)
    }
  }
  return out as T
}

export function BackupSettingsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const fileRef = useRef<HTMLInputElement>(null)
  const [importPreview, setImportPreview] = useState<string | null>(null)
  const [scheduleEnabled, setScheduleEnabled] = useState(false)

  const { data: site } = useAdminSiteSettings()
  const { data: payment } = useAdminPaymentSettings()

  async function exportConfig() {
    try {
      const [siteData, paymentData] = await Promise.all([
        site ?? apiClient('/admin/settings/site'),
        payment ?? apiClient('/admin/settings/payment'),
      ])
      const payload = {
        exported_at: new Date().toISOString(),
        version: 1,
        site: siteData,
        payment: redactSecrets(paymentData as Record<string, unknown>),
      }
      const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `voidllm-settings-${new Date().toISOString().slice(0, 10)}.json`
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
        const parsed = JSON.parse(String(reader.result))
        setImportPreview(JSON.stringify(parsed, null, 2).slice(0, 2000))
        toast({ variant: 'info', message: t('settings.backup_import_preview') })
      } catch {
        toast({ variant: 'error', message: t('settings.backup_import_invalid') })
      }
    }
    reader.readAsText(file)
  }

  function copyDbCommand(cmd: string) {
    void navigator.clipboard.writeText(cmd)
    toast({ variant: 'success', message: t('common.copied') })
  }

  return (
    <div className="space-y-6">
      <SettingsSectionCard
        title={t('settings.backup_export_title')}
        description={t('settings.backup_export_desc')}
        badge={<PreviewBadge />}
      >
        <p className="text-sm text-text-secondary">{t('settings.backup_export_note')}</p>
        <Button variant="secondary" onClick={() => void exportConfig()}>
          {t('settings.backup_export_btn')}
        </Button>
      </SettingsSectionCard>

      <SettingsSectionCard title={t('settings.backup_import_title')} description={t('settings.backup_import_desc')}>
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
        {importPreview && (
          <pre className="max-h-48 overflow-auto rounded-md border border-border bg-bg-tertiary p-3 text-xs font-mono text-text-secondary">
            {importPreview}
            {importPreview.length >= 2000 ? '\n…' : ''}
          </pre>
        )}
      </SettingsSectionCard>

      <SettingsSectionCard title={t('settings.backup_db_title')} description={t('settings.backup_db_desc')}>
        <div className="space-y-3">
          <div className="rounded-md border border-border bg-bg-tertiary p-3">
            <div className="mb-1 text-xs font-medium text-text-tertiary">SQLite</div>
            <code className="block break-all text-xs font-mono">cp data/voidllm.db data/voidllm.db.backup</code>
            <Button
              size="sm"
              variant="ghost"
              className="mt-2"
              onClick={() => copyDbCommand('cp data/voidllm.db data/voidllm.db.backup')}
            >
              {t('common.copy')}
            </Button>
          </div>
          <div className="rounded-md border border-border bg-bg-tertiary p-3">
            <div className="mb-1 text-xs font-medium text-text-tertiary">PostgreSQL</div>
            <code className="block break-all text-xs font-mono">
              pg_dump -Fc postgres://user:pass@host:5432/voidllm &gt; voidllm.dump
            </code>
            <Button
              size="sm"
              variant="ghost"
              className="mt-2"
              onClick={() =>
                copyDbCommand('pg_dump -Fc postgres://user:pass@host:5432/voidllm > voidllm.dump')
              }
            >
              {t('common.copy')}
            </Button>
          </div>
        </div>
        <Banner variant="info" title={t('settings.backup_yaml_hint')} />
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.backup_schedule_title')}
        description={t('settings.backup_schedule_desc')}
        badge={<PreviewBadge />}
      >
        <Toggle
          checked={scheduleEnabled}
          onChange={setScheduleEnabled}
          label={t('settings.backup_schedule_enabled')}
          disabled
        />
        <p className="text-xs text-text-tertiary">{t('settings.backup_schedule_soon')}</p>
      </SettingsSectionCard>

      <SettingsTabFooter
        mode="preview"
        onSave={() => toast({ variant: 'success', message: t('settings.saved_preview') })}
      />
    </div>
  )
}