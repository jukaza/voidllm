import { Badge } from '../../../components/ui/Badge'
import { useAdminPaymentSettings } from '../../../hooks/usePaymentSettings'
import { useAdminSiteSettings } from '../../../hooks/useSiteConfig'
import { useTranslation } from '../../../lib/i18n'

export function SettingsStatusStrip() {
  const { t } = useTranslation()
  const { data: site } = useAdminSiteSettings()
  const { data: payment } = useAdminPaymentSettings()

  const chips = [
    {
      label: t('settings.status_register'),
      on: site?.register_enabled !== false,
    },
    {
      label: t('settings.status_payment'),
      on: payment?.sepay?.enabled === true,
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