import { Link, Outlet, useLocation } from 'react-router-dom'
import { PageHeader } from '../../components/ui/PageHeader'
import { useMe } from '../../hooks/useMe'
import { useTranslation } from '../../lib/i18n'

export default function AnalyticsLayout() {
  const location = useLocation()
  const { data: me } = useMe()
  const { t } = useTranslation()
  const isAdmin = me?.is_system_admin ?? false

  const tabs = [
    { path: '/analytics', label: t('analytics.overview'), exact: true },
    { path: '/analytics/logs', label: t('analytics.request_logs') },
    ...(isAdmin
      ? [
          { path: '/analytics/channels', label: t('analytics.channels') },
          { path: '/analytics/profit', label: t('analytics.revenue_report') },
        ]
      : []),
  ]

  return (
    <>
      <PageHeader title={t('analytics.title')} description={t('analytics.description')} />

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