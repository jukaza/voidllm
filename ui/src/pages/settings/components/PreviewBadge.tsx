import { Badge } from '../../../components/ui/Badge'
import { useTranslation } from '../../../lib/i18n'

export function PreviewBadge() {
  const { t } = useTranslation()
  return (
    <Badge variant="warning" className="shrink-0">
      {t('settings.preview_badge')}
    </Badge>
  )
}

export function LiveBadge() {
  const { t } = useTranslation()
  return (
    <Badge variant="success" className="shrink-0">
      {t('settings.live_badge')}
    </Badge>
  )
}