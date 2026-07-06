import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { PackageMarketCard } from '../components/subscriptions/PackageMarketCard'
import { useTranslation } from '../lib/i18n'
import { usePublicSubscriptionCatalog, useMySubscriptions } from '../hooks/useSubscriptions'

export default function PlansPage() {
  const { t } = useTranslation()
  const { data, isLoading } = usePublicSubscriptionCatalog()
  const { data: mySubs } = useMySubscriptions()
  const [search, setSearch] = useState('')

  const packages = data?.data ?? []
  const activeCount = useMemo(
    () => (mySubs?.data ?? []).filter((s) => s.status === 'active').length,
    [mySubs?.data],
  )

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return packages
    return packages.filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        p.description?.toLowerCase().includes(q) ||
        p.plans.some((pl) => pl.name.toLowerCase().includes(q)),
    )
  }, [packages, search])

  return (
    <>
      <PageHeader
        title={t('plans.title')}
        description={t('plans.desc')}
        actions={
          activeCount > 0 ? (
            <Link to="/my-subscriptions">
              <Button variant="secondary">
                {t('plans.my_subscriptions', { count: activeCount })}
              </Button>
            </Link>
          ) : undefined
        }
      />

      <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t('plans.search')}
          className="max-w-md"
        />
      </div>

      {isLoading ? (
        <div className="rounded-2xl border border-border bg-bg-secondary p-20 text-center text-text-tertiary">
          …
        </div>
      ) : filtered.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-border p-20 text-center">
          <p className="text-text-secondary">{t('plans.empty')}</p>
        </div>
      ) : (
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
          {filtered.map((pkg) => (
            <PackageMarketCard key={pkg.id} pkg={pkg} />
          ))}
        </div>
      )}
    </>
  )
}