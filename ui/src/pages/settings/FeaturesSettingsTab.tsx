import { useEffect, useState } from 'react'
import { Input } from '../../components/ui/Input'
import { Toggle } from '../../components/ui/Toggle'
import {
  useAdminFeaturesSettings,
  useUpdateFeaturesSettings,
  type FeaturesSettings,
} from '../../hooks/useFeaturesSettings'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { LiveBadge } from './components/PreviewBadge'
import { SettingsSectionCard } from './components/SettingsSectionCard'
import { SettingsTabFooter } from './components/SettingsTabFooter'

export function FeaturesSettingsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data, isLoading, isError, error, refetch } = useAdminFeaturesSettings()
  const update = useUpdateFeaturesSettings()

  const [form, setForm] = useState<FeaturesSettings | null>(null)

  useEffect(() => {
    if (data) setForm(data)
  }, [data])

  function save() {
    if (!form) return
    update.mutate(
      {
        wallet: form.wallet,
        modules: form.modules,
      },
      {
        onSuccess: () => toast({ variant: 'success', message: t('common.saved') }),
        onError: (err) => toast({ variant: 'error', message: err.message }),
      },
    )
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
        title={t('settings.features_wallet_title')}
        description={t('settings.features_wallet_desc')}
        badge={<LiveBadge />}
      >
        <Toggle
          checked={form.wallet.enforce_balance}
          onChange={(v) => setForm({ ...form, wallet: { ...form.wallet, enforce_balance: v } })}
          label={t('settings.enforce_balance')}
        />
        <p className="text-xs text-text-tertiary -mt-2">{t('settings.enforce_balance_hint')}</p>
        <Input
          label={t('settings.initial_wallet_balance')}
          type="number"
          min={0}
          value={String(form.wallet.initial_balance_vnd)}
          onChange={(e) =>
            setForm({
              ...form,
              wallet: { ...form.wallet, initial_balance_vnd: Number(e.target.value) || 0 },
            })
          }
          description={t('settings.initial_wallet_balance_hint')}
        />
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.features_ux_title')}
        description={t('settings.features_ux_desc')}
        badge={<LiveBadge />}
      >
        <Toggle
          checked={form.modules.public_catalog}
          onChange={(v) => setForm({ ...form, modules: { ...form.modules, public_catalog: v } })}
          label={t('settings.public_catalog_enabled')}
        />
        <Toggle
          checked={form.modules.playground}
          onChange={(v) => setForm({ ...form, modules: { ...form.modules, playground: v } })}
          label={t('settings.playground_enabled')}
        />
      </SettingsSectionCard>

      <SettingsTabFooter mode="live" loading={update.isPending} onSave={save} />
    </div>
  )
}