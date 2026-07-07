import { PageHeader } from '../components/ui/PageHeader'
import { useMe } from '../hooks/useMe'
import { useTranslation } from '../lib/i18n'
import ModelsPage from './ModelsPage'

export default function ModelsLayout() {
  const { data: me } = useMe()
  const { t } = useTranslation()

  if (me && me.role !== 'root') {
    return (
      <>
        <PageHeader title={t('models.title')} description={t('models.desc')} />
        <div className="rounded-lg border border-border bg-bg-secondary p-12 text-center">
          <p className="text-sm text-text-tertiary">{t('models.admin_required')}</p>
        </div>
      </>
    )
  }

  return <ModelsPage />
}