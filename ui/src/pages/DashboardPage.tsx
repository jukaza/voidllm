import { Navigate, useSearchParams } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { useMe } from '../hooks/useMe'
import { useTranslation } from '../lib/i18n'
import { OverviewTab } from './dashboard/OverviewTab'

export default function DashboardPage() {
  const [searchParams] = useSearchParams()
  if (searchParams.get('tab') === 'integrations') {
    return <Navigate to="/integrations" replace />
  }

  const { t } = useTranslation()
  const { data: me } = useMe()

  const displayName = me?.display_name || me?.email?.split('@')[0] || ''
  const title = displayName ? t('dashboard.welcome', { name: displayName }) : t('sidebar.dashboard')

  return (
    <div className="mx-auto max-w-6xl space-y-8">
      <PageHeader title={title} description={t('dashboard.overview_desc')} />
      <OverviewTab />
    </div>
  )
}