import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { Button } from '../components/ui/Button'
import { ActiveSubscriptionCard } from '../components/subscriptions/ActiveSubscriptionCard'
import { useTranslation } from '../lib/i18n'
import {
  useMySubscriptions,
  usePublicSubscriptionCatalog,
  buildPublicPlanLookup,
  enrichPublicPackage,
  enrichUserSubscription,
  type PublicSubscriptionPackage,
} from '../hooks/useSubscriptions'

function IconSubscription() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="M7 8h10M7 12h6" />
    </svg>
  )
}

export default function MySubscriptionsPage() {
  const { t } = useTranslation()
  const { data, isLoading } = useMySubscriptions()
  const { data: catalogData } = usePublicSubscriptionCatalog()
  const subs = data?.data ?? []

  const packages = useMemo(
    () => (catalogData?.data ?? []).map(enrichPublicPackage),
    [catalogData?.data],
  )

  const packageMap = useMemo(() => {
    const map = new Map<string, PublicSubscriptionPackage>()
    for (const pkg of packages) {
      map.set(pkg.id, pkg)
    }
    return map
  }, [packages])

  const planLookup = useMemo(() => buildPublicPlanLookup(packages), [packages])

  const enrichedSubs = useMemo(
    () => subs.map((s) => enrichUserSubscription(s, planLookup)),
    [subs, planLookup],
  )

  const activeCount = useMemo(
    () => enrichedSubs.filter((s) => s.status === 'active').length,
    [enrichedSubs],
  )
  const expiringSoon = useMemo(
    () =>
      enrichedSubs.filter((s) => s.status === 'active' && (s.days_remaining ?? 999) <= 7).length,
    [enrichedSubs],
  )

  const summary =
    enrichedSubs.length > 0
      ? t('my_subs.summary', {
          total: enrichedSubs.length,
          active: activeCount,
          expiring: expiringSoon,
        })
      : undefined

  return (
    <>
      <PageHeader
        title={t('my_subs.title')}
        description={summary ?? t('my_subs.desc')}
        actions={
          <Link to="/plans">
            <Button>{t('my_subs.browse')}</Button>
          </Link>
        }
      />

      {isLoading ? (
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-72 animate-pulse rounded-2xl border border-border bg-bg-secondary" />
          ))}
        </div>
      ) : enrichedSubs.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border/80 bg-bg-secondary/50 px-8 py-16 text-center">
          <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-accent/10 text-accent">
            <IconSubscription />
          </div>
          <h2 className="text-base font-semibold text-text-primary">{t('my_subs.empty_title')}</h2>
          <p className="mt-1.5 max-w-sm text-sm text-text-tertiary">{t('my_subs.empty')}</p>
          <Link to="/plans" className="mt-5">
            <Button>{t('my_subs.browse')}</Button>
          </Link>
        </div>
      ) : (
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
          {enrichedSubs.map((sub) => (
            <ActiveSubscriptionCard
              key={sub.id}
              sub={sub}
              pkg={sub.package_id ? packageMap.get(sub.package_id) : undefined}
            />
          ))}
        </div>
      )}
    </>
  )
}