import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { PackageCover, presetGradients } from './PackageCover'
import { QuotaMeter } from './QuotaMeter'
import { Badge } from '../ui/Badge'
import { useTranslation } from '../../lib/i18n'
import { cn, formatNumber } from '../../lib/utils'
import type { PublicSubscriptionPackage, UserSubscription } from '../../hooks/useSubscriptions'

function fallbackCover(packageId?: string) {
  const keys = Object.keys(presetGradients)
  const idx = packageId
    ? packageId.split('').reduce((a, c) => a + c.charCodeAt(0), 0) % keys.length
    : 0
  return { cover_type: 'default' as const, cover_value: keys[idx] ?? 'aurora' }
}

function statusVariant(status: string): 'success' | 'warning' | 'error' | 'muted' {
  if (status === 'active') return 'success'
  if (status === 'expired') return 'error'
  if (status === 'cancelled') return 'muted'
  return 'warning'
}

function formatShortDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

export function ActiveSubscriptionCard({
  sub,
  pkg,
}: {
  sub: UserSubscription
  pkg?: PublicSubscriptionPackage | null
}) {
  const { t } = useTranslation()
  const plan = sub.plan
  const usage = sub.usage
  const cover = pkg ?? fallbackCover(sub.package_id)

  const quotas = useMemo(() => {
    const items: { label: string; used: number; limit: number }[] = []
    if (plan?.daily_token_limit) {
      items.push({
        label: t('plans.daily_tokens'),
        used: usage?.daily_tokens ?? 0,
        limit: plan.daily_token_limit,
      })
    }
    if (plan?.monthly_token_limit) {
      items.push({
        label: t('plans.monthly_tokens'),
        used: usage?.monthly_tokens ?? 0,
        limit: plan.monthly_token_limit,
      })
    }
    if (plan?.daily_request_limit) {
      items.push({
        label: t('plans.daily_requests'),
        used: usage?.requests_per_day ?? 0,
        limit: plan.daily_request_limit,
      })
    }
    if (plan?.monthly_request_limit) {
      items.push({
        label: t('plans.monthly_requests'),
        used: usage?.monthly_requests ?? 0,
        limit: plan.monthly_request_limit,
      })
    }
    return items
  }, [plan, usage, t])

  const statusLabels: Record<string, string> = {
    active: t('my_subs.status_active'),
    expired: t('my_subs.status_expired'),
    cancelled: t('my_subs.status_cancelled'),
  }
  const statusLabel = statusLabels[sub.status] ?? sub.status
  const daysLeft = sub.days_remaining
  const urgent = daysLeft !== undefined && daysLeft <= 7

  return (
    <article className="group flex flex-col overflow-hidden rounded-2xl border border-border/80 bg-bg-secondary shadow-sm transition-all hover:border-accent/40 hover:shadow-md hover:shadow-accent/5">
      <PackageCover pkg={cover} compact />
      <div className="flex flex-1 flex-col gap-2.5 p-4">
        <div className="flex flex-wrap items-center gap-1.5">
          <Badge variant={statusVariant(sub.status)}>{statusLabel}</Badge>
        </div>

        <div>
          <h3 className="text-base font-bold text-text-primary line-clamp-2 group-hover:text-accent transition-colors">
            {sub.package_name || sub.plan_name}
          </h3>
          {sub.plan_name && sub.package_name && (
            <p className="mt-0.5 text-xs text-text-tertiary line-clamp-1">{sub.plan_name}</p>
          )}
        </div>

        <div className="grid grid-cols-2 gap-2 rounded-lg border border-border/60 bg-bg-primary/40 p-2.5">
          <div>
            <p className="text-[9px] font-semibold uppercase tracking-wide text-text-tertiary">
              {t('my_subs.validity_left')}
            </p>
            <p
              className={cn(
                'mt-0.5 text-base font-bold tabular-nums leading-tight',
                urgent ? 'text-warning' : 'text-accent',
              )}
            >
              {daysLeft !== undefined ? (
                <>
                  {formatNumber(daysLeft)}
                  <span className="ml-0.5 text-[11px] font-semibold text-text-tertiary">
                    {t('plans.days')}
                  </span>
                </>
              ) : (
                '—'
              )}
            </p>
          </div>
          <div>
            <p className="text-[9px] font-semibold uppercase tracking-wide text-text-tertiary">
              {t('my_subs.expires')}
            </p>
            <p className="mt-0.5 text-sm font-semibold tabular-nums text-text-primary">
              {formatShortDate(sub.expires_at)}
            </p>
            <p className="text-[10px] text-text-tertiary">
              {t('my_subs.started')}: {formatShortDate(sub.starts_at)}
            </p>
          </div>
        </div>

        {quotas.length > 0 ? (
          <div className="space-y-1.5">
            <p className="text-[9px] font-semibold uppercase tracking-wide text-text-tertiary">
              {t('my_subs.quota_usage')}
            </p>
            {quotas.map((q) => (
              <QuotaMeter key={q.label} label={q.label} used={q.used} limit={q.limit} />
            ))}
          </div>
        ) : (
          <p className="text-[11px] text-text-tertiary">{t('my_subs.no_quota')}</p>
        )}

        <div className="mt-auto flex items-center justify-between gap-2 border-t border-border/50 pt-2.5 text-[11px]">
          {sub.package_id ? (
            <Link
              to={`/plans/${sub.package_id}`}
              className="font-medium text-accent hover:underline"
            >
              {t('my_subs.view_package')}
            </Link>
          ) : (
            <span />
          )}
          <Link to="/keys" className="font-medium text-text-secondary hover:text-accent hover:underline">
            {t('my_subs.attach_key')}
          </Link>
        </div>
      </div>
    </article>
  )
}