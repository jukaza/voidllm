import { useEffect, useState } from 'react'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Toggle } from '../../components/ui/Toggle'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import {
  useAdminEmailSettings,
  useUpdateEmailSettings,
  type EmailSettings,
} from '../../hooks/useEmailSettings'

function SectionCard({ title, description, children }: { title: string; description?: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-border bg-bg-secondary mb-6">
      <div className="px-6 py-4 border-b border-border">
        <h2 className="text-sm font-semibold text-text-primary">{title}</h2>
        {description && <p className="mt-1 text-xs text-text-tertiary">{description}</p>}
      </div>
      <div className="p-6 space-y-5">{children}</div>
    </div>
  )
}

const GMAIL_DEFAULTS = {
  host: 'smtp.gmail.com',
  port: 587,
  ssl_enabled: false,
} as const

export function EmailSettingsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data, isLoading, isError, error, refetch } = useAdminEmailSettings()
  const update = useUpdateEmailSettings()

  const [form, setForm] = useState<EmailSettings | null>(null)
  const [password, setPassword] = useState('')

  useEffect(() => {
    if (data) setForm(data)
  }, [data])

  function applyGmailDefaults() {
    if (!form) return
    setForm({
      ...form,
      host: GMAIL_DEFAULTS.host,
      port: GMAIL_DEFAULTS.port,
      ssl_enabled: GMAIL_DEFAULTS.ssl_enabled,
    })
  }

  function save() {
    if (!form) return
    update.mutate(
      {
        smtp: {
          enabled: form.enabled,
          host: form.host.trim(),
          port: form.port,
          username: form.username.trim(),
          from: form.from.trim(),
          ssl_enabled: form.ssl_enabled,
          password: password || undefined,
        },
      },
      {
        onSuccess: () => {
          setPassword('')
          toast({ variant: 'success', message: t('common.saved') })
        },
        onError: (e) => toast({ variant: 'error', message: e.message }),
      },
    )
  }

  if (isLoading) return <p className="text-sm text-text-tertiary">{t('common.loading')}</p>
  if (isError || !form) {
    return (
      <div>
        <p className="text-sm text-error">{error?.message ?? t('settings.email_load_error')}</p>
        <Button className="mt-3" variant="secondary" onClick={() => void refetch()}>
          {t('common.refresh')}
        </Button>
      </div>
    )
  }

  return (
    <div>
      <SectionCard title={t('settings.email_setup_title')} description={t('settings.email_setup_desc')}>
        <ol className="list-decimal list-inside space-y-2 text-sm text-text-secondary">
          <li>{t('settings.email_setup_step1')}</li>
          <li>{t('settings.email_setup_step2')}</li>
          <li>{t('settings.email_setup_step3')}</li>
          <li>{t('settings.email_setup_step4')}</li>
        </ol>
        <div className="rounded-md border border-border bg-bg-tertiary px-3 py-2 text-xs text-text-tertiary space-y-2">
          <p>{t('settings.email_setup_docs')}</p>
          <ul className="list-disc list-inside space-y-1">
            <li>
              <a
                className="text-accent hover:underline"
                href="https://support.google.com/accounts/answer/185833"
                target="_blank"
                rel="noreferrer"
              >
                {t('settings.email_setup_doc_app_password')}
              </a>
            </li>
            <li>
              <a
                className="text-accent hover:underline"
                href="https://support.google.com/mail/answer/7126229"
                target="_blank"
                rel="noreferrer"
              >
                {t('settings.email_setup_doc_gmail_smtp')}
              </a>
            </li>
          </ul>
          <p className="text-text-secondary">{t('settings.email_setup_limit_note')}</p>
        </div>
      </SectionCard>

      <SectionCard title={t('settings.email_smtp_title')} description={t('settings.email_smtp_desc')}>
        <Toggle
          checked={form.enabled}
          onChange={(v) => setForm({ ...form, enabled: v })}
          label={t('settings.email_enabled')}
        />

        <div className="flex justify-end">
          <Button size="sm" variant="secondary" onClick={applyGmailDefaults}>
            {t('settings.email_use_gmail_defaults')}
          </Button>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Input
            label={t('settings.email_host')}
            value={form.host}
            onChange={(e) => setForm({ ...form, host: e.target.value })}
            placeholder="smtp.gmail.com"
          />
          <Input
            label={t('settings.email_port')}
            type="number"
            value={String(form.port)}
            onChange={(e) => setForm({ ...form, port: Number(e.target.value) })}
            description={t('settings.email_port_hint')}
          />
        </div>

        <Input
          label={t('settings.email_username')}
          value={form.username}
          onChange={(e) => setForm({ ...form, username: e.target.value })}
          placeholder="your@gmail.com"
          description={t('settings.email_username_hint')}
        />

        <Input
          label={t('settings.email_password')}
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder={form.password_configured ? '••••••••' : ''}
          description={t('settings.email_password_hint')}
        />

        <Input
          label={t('settings.email_from')}
          value={form.from}
          onChange={(e) => setForm({ ...form, from: e.target.value })}
          placeholder="VoidLLM <noreply@yourdomain.com>"
          description={t('settings.email_from_hint')}
        />

        <Toggle
          checked={form.ssl_enabled}
          onChange={(v) =>
            setForm({
              ...form,
              ssl_enabled: v,
              port: v ? 465 : 587,
            })
          }
          label={t('settings.email_ssl_enabled')}
        />
        <p className="text-xs text-text-tertiary -mt-3">{t('settings.email_ssl_hint')}</p>
      </SectionCard>

      <div className="flex justify-end">
        <Button loading={update.isPending} onClick={save}>
          {t('common.save')}
        </Button>
      </div>
    </div>
  )
}