import { ModelCatalogView } from '../components/catalog/ModelCatalogView'
import { PageHeader } from '../components/ui/PageHeader'
import { useMemberCatalog } from '../hooks/useProviders'
import { useTranslation } from '../lib/i18n'

export default function CatalogPage() {
  const { data, isLoading } = useMemberCatalog()
  const { t } = useTranslation()
  const models = data?.data ?? []

  return (
    <>
      <PageHeader title={t('sidebar.catalog')} />
      <ModelCatalogView
        models={models}
        isLoading={isLoading}
        emptyMessage={t('catalog.empty')}
      />
    </>
  )
}