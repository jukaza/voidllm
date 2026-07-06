import { PageHeader } from '../components/ui/PageHeader'
import { useTranslation } from '../lib/i18n'
import { IntegrationsTab } from './dashboard/IntegrationsTab'

export default function IntegrationsPage() {
  const { t } = useTranslation()

  return (
    <div className="mx-auto max-w-6xl space-y-8">
      <PageHeader title={t('integrations.page_title')} description={t('integrations.page_desc')} />
      <IntegrationsTab />
    </div>
  )
}