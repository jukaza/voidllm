import { Badge } from '../../../components/ui/Badge'
import { useTranslation } from '../../../lib/i18n'
import { useAccountDraft } from '../useAccountDraft'

export function AccountStatusStrip() {
  const { t } = useTranslation()
  const { draft } = useAccountDraft()

  const oauthCount = [draft.oauth_google, draft.oauth_github, draft.oauth_oidc].filter(
    (o) => o.bound,
  ).length

  const chips = [
    {
      label: t('account.status_2fa'),
      on: draft.two_fa_enabled,
    },
    {
      label: t('account.status_oauth'),
      on: oauthCount > 0,
      detail: oauthCount > 0 ? String(oauthCount) : undefined,
    },
    {
      label: t('account.status_sessions'),
      on: draft.sessions.length > 0,
      detail: String(draft.sessions.length),
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