import { PageHeader } from '../components/ui/PageHeader'
import { PublicPricingTable } from '../components/catalog/PublicPricingTable'
import { usePublicCatalog } from '../hooks/useProviders'
import { useTranslation } from '../lib/i18n'

export default function CatalogPage() {
  const { data, isLoading } = usePublicCatalog()
  const { t } = useTranslation()
  const models = data?.data ?? []

  return (
    <>
      <PageHeader title={t('sidebar.catalog')} description={t('catalog.desc')} />
      <PublicPricingTable
        models={models}
        isLoading={isLoading}
        emptyMessage={t('catalog.empty')}
        variant="admin"
      />
    </>
  )
}