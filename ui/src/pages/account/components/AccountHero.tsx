import { Link } from 'react-router-dom'
import { Badge } from '../../../components/ui/Badge'
import { useMe } from '../../../hooks/useMe'
import { useMyWallet } from '../../../hooks/useWallet'
import { useMyUsage } from '../../../hooks/useUsage'
import { useAPIKeys } from '../../../hooks/useAPIKeys'
import { useTranslation } from '../../../lib/i18n'
import { formatCost, formatNumber } from '../../../lib/utils'
import { useMemo } from 'react'
import { useAccountDraft } from '../useAccountDraft'

function usageRange30d() {
  const to = new Date()
  const from = new Date(to.getTime() - 30 * 24 * 3_600_000)
  return { from: from.toISOString(), to: to.toISOString() }
}

export function AccountHero() {
  const { t } = useTranslation()
  const { draft } = useAccountDraft()
  const { data: me, isLoading: meLoading } = useMe()
  const { data: wallet, isLoading: walletLoading } = useMyWallet()
  const range = useMemo(() => usageRange30d(), [])
  const { data: usage, isLoading: usageLoading } = useMyUsage(range.from, range.to, 'day')
  const { data: keys, isLoading: keysLoading } = useAPIKeys()

  const loading = meLoading || walletLoading || usageLoading || keysLoading

  const totalTokens = useMemo(
    () => (usage?.data ?? []).reduce((sum, row) => sum + row.total_tokens, 0),
    [usage],
  )

  const keyCount = keys?.data.length ?? 0
  const keyLabel = keys?.has_more ? `${keyCount}+` : String(keyCount)

  if (loading || !me) {
    return (
      <div className="mb-6 overflow-hidden rounded-lg border border-border bg-bg-secondary animate-pulse">
        <div className="h-24 border-b border-border" />
        <div className="grid grid-cols-3 gap-4 p-4 h-20" />
      </div>
    )
  }

  const displayName = draft.display_name || me.display_name

  const stats = [
    {
      label: t('account.stat_balance'),
      value: formatCost(wallet?.balance ?? 0),
      hint: t('account.stat_balance_hint'),
      to: '/wallet',
    },
    {
      label: t('account.stat_usage'),
      value: formatNumber(totalTokens),
      hint: t('account.stat_usage_hint'),
      to: '/analytics',
    },
    {
      label: t('account.stat_keys'),
      value: keyLabel,
      hint: t('account.stat_keys_hint'),
      to: '/keys',
    },
  ]

  return (
    <div className="mb-6 overflow-hidden rounded-lg border border-border bg-bg-secondary">
      <div className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:p-5">
        <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-xl bg-accent/10 text-2xl font-semibold text-accent sm:h-16 sm:w-16">
          {displayName?.[0]?.toUpperCase() || 'U'}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="truncate text-xl font-semibold text-text-primary sm:text-2xl">
              {displayName || t('account.unnamed')}
            </h2>
            <Badge variant="default" className="text-xs capitalize">
              {me.role?.replace('_', ' ')}
            </Badge>
            {me.is_system_admin && (
              <Badge variant="info" className="text-xs">
                Admin
              </Badge>
            )}
          </div>
          <p className="mt-1 truncate font-mono text-sm text-text-tertiary">{me.email}</p>
        </div>
      </div>
      <div className="grid grid-cols-1 divide-y border-t border-border sm:grid-cols-3 sm:divide-x sm:divide-y-0">
        {stats.map((stat) => (
          <Link
            key={stat.label}
            to={stat.to}
            className="group px-4 py-3.5 transition-colors hover:bg-bg-tertiary/50 sm:px-5 sm:py-4"
          >
            <div className="text-xs font-medium uppercase tracking-wide text-text-tertiary">
              {stat.label}
            </div>
            <div className="mt-1 font-mono text-lg font-bold tabular-nums text-text-primary sm:text-xl">
              {stat.value}
            </div>
            <div className="mt-0.5 text-xs text-text-tertiary group-hover:text-accent">
              {stat.hint} →
            </div>
          </Link>
        ))}
      </div>
    </div>
  )
}