import { Badge } from '../../../components/ui/Badge'
import { Button } from '../../../components/ui/Button'
import { useTranslation } from '../../../lib/i18n'

interface SecurityActionTilesProps {
  twoFAEnabled: boolean
  sessionCount: number
  onTwoFA: () => void
  onPassword: () => void
  onSessions: () => void
}

export function SecurityActionTiles({
  twoFAEnabled,
  sessionCount,
  onTwoFA,
  onPassword,
  onSessions,
}: SecurityActionTilesProps) {
  const { t } = useTranslation()

  const tiles = [
    {
      title: t('account.twofa_title'),
      desc: t('account.twofa_desc'),
      badge: twoFAEnabled ? t('account.twofa_on') : t('account.twofa_off'),
      badgeVariant: twoFAEnabled ? ('success' as const) : ('muted' as const),
      action: twoFAEnabled ? t('account.twofa_manage') : t('account.twofa_enable'),
      onClick: onTwoFA,
    },
    {
      title: t('account.password_title'),
      desc: t('account.password_desc'),
      action: t('account.password_change'),
      onClick: onPassword,
    },
    {
      title: t('account.sessions_title'),
      desc: t('account.sessions_count', { count: sessionCount }),
      action: t('account.sessions_manage'),
      onClick: onSessions,
    },
  ]

  return (
    <div className="space-y-3">
      {tiles.map((tile) => (
        <div
          key={tile.title}
          className="flex items-center justify-between gap-3 rounded-md border border-border p-4"
        >
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm font-medium text-text-primary">{tile.title}</span>
              {tile.badge && (
                <Badge variant={tile.badgeVariant} className="text-xs">
                  {tile.badge}
                </Badge>
              )}
            </div>
            <p className="mt-0.5 text-xs text-text-tertiary">{tile.desc}</p>
          </div>
          <Button size="sm" variant="secondary" className="shrink-0" onClick={tile.onClick}>
            {tile.action}
          </Button>
        </div>
      ))}
    </div>
  )
}