import { Badge } from '../../../components/ui/Badge'
import { useAdminFeaturesSettings } from '../../../hooks/useFeaturesSettings'
import { useAdminPaymentSettings } from '../../../hooks/usePaymentSettings'
import { useAdminSiteSettings } from '../../../hooks/useSiteConfig'
import { useTranslation } from '../../../lib/i18n'

export function SettingsStatusStrip() {
  const { t } = useTranslation()
  const { data: site } = useAdminSiteSettings()
  const { data: payment } = useAdminPaymentSettings()
  const { data: features } = useAdminFeaturesSettings()

  const chips = [
    {
      label: t('settings.status_register'),
      on: site?.register_enabled !== false,
    },
    {
      label: t('settings.status_payment'),
      on: payment?.sepay?.enabled === true,
    },
    {
      label: t('settings.status_enforce_balance'),
      on: features?.wallet.enforce_balance === true,
    },
    {
      label: t('settings.status_playground'),
      on: features?.modules.playground !== false,
    },
  ]

  return (
    <div className="mb-6 flex flex-wrap gap-2">
      {chips.map((chip) => (
        <Badge key={chip.label} variant={chip.on ? 'success' : 'muted'}>
          {chip.label}: {chip.on ? t('settings.status_on') : t('settings.status_off')}
        </Badge>
      ))}
    </div>
  )
}