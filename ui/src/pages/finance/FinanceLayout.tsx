import { Link, Outlet, useLocation } from 'react-router-dom'
import { PageHeader } from '../../components/ui/PageHeader'
import { useMe } from '../../hooks/useMe'
import { useTranslation } from '../../lib/i18n'

export default function FinanceLayout() {
  const location = useLocation()
  const { data: me } = useMe()
  const { t } = useTranslation()

  if (me && !me.is_system_admin) {
    return (
      <>
        <PageHeader title={t('finance.title')} description={t('finance.subtitle')} />
        <div className="rounded-lg border border-border bg-bg-secondary p-12 text-center">
          <p className="text-sm text-text-tertiary">{t('finance.admin_required')}</p>
        </div>
      </>
    )
  }

  const tabs = [
    { path: '/finance', label: t('finance.tab_overview'), exact: true },
    { path: '/finance/topups', label: t('finance.tab_topups') },
    { path: '/finance/ledger', label: t('finance.tab_ledger') },
  ]

  return (
    <>
      <PageHeader title={t('finance.title')} description={t('finance.subtitle')} />

      <div className="flex items-center gap-1 mb-6 border-b border-border">
        {tabs.map((tab) => {
          const isActive = tab.exact
            ? location.pathname === tab.path
            : location.pathname.startsWith(tab.path)
          return (
            <Link
              key={tab.path}
              to={tab.path}
              className={`px-4 py-2.5 text-sm font-medium transition-colors -mb-px no-underline ${
                isActive
                  ? 'text-accent border-b-2 border-accent'
                  : 'text-text-secondary hover:text-text-primary'
              }`}
            >
              {tab.label}
            </Link>
          )
        })}
      </div>

      <Outlet />
    </>
  )
}