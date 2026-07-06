import { Badge } from '../../../components/ui/Badge'
import { useMe } from '../../../hooks/useMe'
import { useMeConnections } from '../../../hooks/useSecuritySettings'
import { useTranslation } from '../../../lib/i18n'

export function AccountStatusStrip() {
  const { t } = useTranslation()
  const { data: me } = useMe()
  const { data: connections } = useMeConnections()

  if (!me) return null

  const oauthCount = [connections?.google, connections?.github].filter((c) => c?.linked).length

  const chips = [
    {
      label: t('account.status_oauth'),
      on: oauthCount > 0,
      detail: oauthCount > 0 ? String(oauthCount) : undefined,
    },
    {
      label: t('account.status_sessions'),
      on: me.active_session_count > 0,
      detail: String(me.active_session_count),
    },
  ]

  return (
    <div className="mb-6 flex flex-wrap gap-2">
      {chips.map((chip) => (
        <Badge key={chip.label} variant={chip.on ? 'success' : 'muted'}>
          {chip.label}: {chip.on ? (chip.detail ?? t('account.status_on')) : t('account.status_off')}
        </Badge>
      ))}
    </div>
  )
}