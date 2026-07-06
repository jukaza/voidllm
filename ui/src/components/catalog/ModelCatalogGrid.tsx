import { Button } from '../ui/Button'
import type { CatalogModelItem } from '../../hooks/useProviders'
import { CATALOG_PAGE_SIZE } from '../../lib/catalog-utils'
import { useTranslation } from '../../lib/i18n'
import { ModelCatalogCard } from './ModelCatalogCard'

interface ModelCatalogGridProps {
  models: CatalogModelItem[]
  visibleCount: number
  isLoading?: boolean
  emptyMessage: string
  noResultsMessage: string
  onLoadMore: () => void
}

export function ModelCatalogGrid({
  models,
  visibleCount,
  isLoading = false,
  emptyMessage,
  noResultsMessage,
  onLoadMore,
}: ModelCatalogGridProps) {
  const { t } = useTranslation()
  const visible = models.slice(0, visibleCount)
  const hasMore = visibleCount < models.length

  if (isLoading) {
    return (
      <div className="flex min-h-[240px] items-center justify-center text-sm text-text-tertiary">
        {t('common.loading')}
      </div>
    )
  }

  if (models.length === 0) {
    return (
      <div className="flex min-h-[240px] items-center justify-center rounded-xl border border-dashed border-border px-6 text-center text-sm text-text-tertiary">
        {emptyMessage}
      </div>
    )
  }

  if (visible.length === 0) {
    return (
      <div className="flex min-h-[240px] items-center justify-center rounded-xl border border-dashed border-border px-6 text-center text-sm text-text-tertiary">
        {noResultsMessage}
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
        {visible.map((model) => (
          <ModelCatalogCard key={model.name} model={model} />
        ))}
      </div>
      {hasMore && (
        <div className="flex justify-center">
          <Button variant="secondary" size="sm" onClick={onLoadMore}>
            {t('catalog.load_more')}
          </Button>
        </div>
      )}
    </div>
  )
}

export { CATALOG_PAGE_SIZE }