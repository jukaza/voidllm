import { Link } from 'react-router-dom'
import { PackageCover } from './PackageCover'
import { useTranslation } from '../../lib/i18n'
import { formatCost } from '../../lib/utils'
import type { PublicSubscriptionPackage } from '../../hooks/useSubscriptions'

export function PackageMarketCard({ pkg }: { pkg: PublicSubscriptionPackage }) {
  const { t } = useTranslation()

  return (
    <Link
      to={`/plans/${pkg.id}`}
      className="group flex flex-col overflow-hidden rounded-2xl border border-border/80 bg-bg-secondary shadow-sm transition-all hover:border-accent/40 hover:shadow-md hover:shadow-accent/5"
    >
      <PackageCover pkg={pkg} compact />
      <div className="flex flex-1 flex-col gap-3 p-4">
        <div>
          <p className="text-[10px] font-semibold uppercase tracking-wide text-text-tertiary">
            {t('plans.official')}
          </p>
          <h3 className="mt-1 text-base font-bold text-text-primary line-clamp-2 group-hover:text-accent transition-colors">
            {pkg.name}
          </h3>
          {pkg.description && (
            <p className="mt-1.5 text-xs text-text-tertiary line-clamp-2">{pkg.description}</p>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-2 text-[11px] text-text-tertiary">
          <span>
            {(pkg.model_count ?? 0) > 0
              ? t('plans.models_count', { count: pkg.model_count ?? 0 })
              : t('plans.no_models')}
          </span>
          <span className="text-border">•</span>
          <span>
            {(pkg.subscriber_count ?? 0) > 0
              ? t('plans.subscribers_count', { count: pkg.subscriber_count ?? 0 })
              : t('plans.no_subscribers')}
          </span>
        </div>
        <div className="mt-auto flex items-end justify-between gap-2 pt-1">
          <div>
            <p className="text-[10px] text-text-tertiary">{t('plans.from_price')}</p>
            <p className="text-lg font-bold text-emerald-400 tabular-nums">
              {formatCost(pkg.min_price ?? pkg.plans[0]?.price ?? 0)}
            </p>
          </div>
          <span className="rounded-lg bg-accent/10 px-2.5 py-1 text-[11px] font-semibold text-accent">
            {pkg.plans.length} {t('plans.plan_options')}
          </span>
        </div>
      </div>
    </Link>
  )
}