import { Link } from 'react-router-dom'
import { Input } from '../../components/ui/Input'
import { Toggle } from '../../components/ui/Toggle'
import { useAdminPaymentSettings } from '../../hooks/usePaymentSettings'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { PreviewBadge } from './components/PreviewBadge'
import { SettingsSectionCard } from './components/SettingsSectionCard'
import { SettingsTabFooter } from './components/SettingsTabFooter'
import { useSettingsDraft } from './useSettingsDraft'

function DeepLinkRow({ label, hint, to }: { label: string; hint: string; to: string }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-border px-4 py-3">
      <div>
        <div className="text-sm font-medium text-text-primary">{label}</div>
        <div className="text-xs text-text-tertiary">{hint}</div>
      </div>
      <Link to={to} className="text-xs text-accent hover:underline shrink-0">
        → {to}
      </Link>
    </div>
  )
}

export function FeaturesSettingsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { draft, setDraft } = useSettingsDraft()
  const { data: payment } = useAdminPaymentSettings()

  return (
    <div className="space-y-6">
      <SettingsSectionCard
        title={t('settings.features_wallet_title')}
        description={t('settings.features_wallet_desc')}
        badge={<PreviewBadge />}
      >
        <div className="flex items-center justify-between gap-4 rounded-md border border-border px-4 py-3">
          <div>
            <div className="text-sm font-medium">{t('settings.payment_enabled')}</div>
            <div className="text-xs text-text-tertiary">{t('settings.features_payment_hint')}</div>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-text-tertiary">
              {payment?.sepay?.enabled ? t('settings.status_on') : t('settings.status_off')}
            </span>
            <Link to="/settings?tab=payment" className="text-xs text-accent hover:underline">
              {t('settings.configure')}
            </Link>
          </div>
        </div>
        <Toggle
          checked={draft.enforce_balance}
          onChange={(v) => setDraft({ enforce_balance: v })}
          label={t('settings.enforce_balance')}
        />
        <p className="text-xs text-text-tertiary -mt-2">{t('settings.enforce_balance_hint')}</p>
        <Input
          label={t('settings.initial_wallet_balance')}
          type="number"
          value={String(draft.initial_wallet_balance)}
          onChange={(e) => setDraft({ initial_wallet_balance: Number(e.target.value) || 0 })}
          description={t('settings.initial_wallet_balance_hint')}
        />
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.features_ux_title')}
        description={t('settings.features_ux_desc')}
        badge={<PreviewBadge />}
      >
        <Toggle
          checked={draft.public_catalog_enabled}
          onChange={(v) => setDraft({ public_catalog_enabled: v })}
          label={t('settings.public_catalog_enabled')}
        />
        <Toggle
          checked={draft.playground_enabled}
          onChange={(v) => setDraft({ playground_enabled: v })}
          label={t('settings.playground_enabled')}
        />
      </SettingsSectionCard>

      <SettingsSectionCard title={t('settings.features_limits_title')} description={t('settings.features_limits_desc')}>
        <DeepLinkRow
          label={t('settings.limits_models')}
          hint={t('settings.limits_models_hint')}
          to="/models"
        />
        <DeepLinkRow
          label={t('settings.limits_providers')}
          hint={t('settings.limits_providers_hint')}
          to="/providers"
        />
        <DeepLinkRow label={t('settings.limits_keys')} hint={t('settings.limits_keys_hint')} to="/keys" />
        <DeepLinkRow
          label={t('settings.limits_finance')}
          hint={t('settings.limits_finance_hint')}
          to="/finance"
        />
      </SettingsSectionCard>

      <SettingsTabFooter
        mode="preview"
        onSave={() => toast({ variant: 'success', message: t('settings.saved_preview') })}
      />
    </div>
  )
}