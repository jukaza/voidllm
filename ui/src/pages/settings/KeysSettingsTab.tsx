import { useEffect, useState } from 'react'
import { Input } from '../../components/ui/Input'
import { Toggle } from '../../components/ui/Toggle'
import { useKeysPolicy, useUpdateKeysPolicy, type KeysPolicy } from '../../hooks/useAPIKeys'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { LiveBadge } from './components/PreviewBadge'
import { SettingsSectionCard } from './components/SettingsSectionCard'
import { SettingsTabFooter } from './components/SettingsTabFooter'

export function KeysSettingsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data, isLoading, isError, error, refetch } = useKeysPolicy()
  const update = useUpdateKeysPolicy()

  const [form, setForm] = useState<KeysPolicy | null>(null)

  useEffect(() => {
    if (data) setForm(data)
  }, [data])

  function save() {
    if (!form) return
    update.mutate(form, {
      onSuccess: () => toast({ variant: 'success', message: t('common.saved') }),
      onError: (err) => toast({ variant: 'error', message: err.message }),
    })
  }

  if (isLoading) {
    return <p className="text-sm text-text-tertiary">{t('common.loading')}</p>
  }

  if (isError || !form) {
    return (
      <div className="space-y-3">
        <p className="text-sm text-error">{error?.message ?? t('settings.load_error')}</p>
        <button type="button" className="text-sm text-accent" onClick={() => void refetch()}>
          {t('common.refresh')}
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <SettingsSectionCard
        title={t('settings.keys_title')}
        description={t('settings.keys_desc')}
        badge={<LiveBadge />}
      >
        <Input
          label={t('settings.keys_max_per_user')}
          type="number"
          min={1}
          value={String(form.max_per_user)}
          onChange={(e) =>
            setForm({ ...form, max_per_user: Math.max(1, Number(e.target.value) || 1) })
          }
          description={t('settings.keys_max_per_user_hint')}
        />
        <Input
          label={t('settings.keys_default_expiry')}
          type="number"
          min={0}
          value={String(form.default_expiry_days)}
          onChange={(e) =>
            setForm({ ...form, default_expiry_days: Math.max(0, Number(e.target.value) || 0) })
          }
          description={t('settings.keys_default_expiry_hint')}
        />
        <Toggle
          checked={form.auto_create_on_register}
          onChange={(v) => setForm({ ...form, auto_create_on_register: v })}
          label={t('settings.keys_auto_create')}
        />
        <p className="text-xs text-text-tertiary -mt-2">{t('settings.keys_auto_create_hint')}</p>
        <Toggle
          checked={form.allow_custom_key}
          onChange={(v) => setForm({ ...form, allow_custom_key: v })}
          label={t('settings.keys_allow_custom')}
        />
        <p className="text-xs text-text-tertiary -mt-2">{t('settings.keys_allow_custom_hint')}</p>
        <Toggle
          checked={form.trust_forwarded_ip}
          onChange={(v) => setForm({ ...form, trust_forwarded_ip: v })}
          label={t('settings.keys_trust_forwarded_ip')}
        />
        <p className="text-xs text-text-tertiary -mt-2">{t('settings.keys_trust_forwarded_ip_hint')}</p>
      </SettingsSectionCard>

      <SettingsTabFooter mode="live" loading={update.isPending} onSave={save} />
    </div>
  )
}