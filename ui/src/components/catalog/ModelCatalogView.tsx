import { useMemo, useState } from 'react'
import { Drawer } from '../ui/Drawer'
import type { CatalogModelItem } from '../../hooks/useProviders'
import {
  DEFAULT_CATALOG_FILTERS,
  type CatalogFilterState,
  applyCatalogFilters,
  filterChatModels,
  CATALOG_PAGE_SIZE,
} from '../../lib/catalog-utils'
import { useTranslation } from '../../lib/i18n'
import { ModelCatalogFilters } from './ModelCatalogFilters'
import { ModelCatalogGrid } from './ModelCatalogGrid'
import { ModelCatalogToolbar } from './ModelCatalogToolbar'

interface ModelCatalogViewProps {
  models: CatalogModelItem[]
  isLoading?: boolean
  emptyMessage: string
}

export function ModelCatalogView({
  models,
  isLoading = false,
  emptyMessage,
}: ModelCatalogViewProps) {
  const { t } = useTranslation()
  const [filters, setFilters] = useState<CatalogFilterState>(DEFAULT_CATALOG_FILTERS)
  const [visibleCount, setVisibleCount] = useState(CATALOG_PAGE_SIZE)
  const [drawerOpen, setDrawerOpen] = useState(false)

  const chatModels = useMemo(() => filterChatModels(models), [models])

  const filtered = useMemo(
    () => applyCatalogFilters(chatModels, filters),
    [chatModels, filters],
  )

  function patchFilters(patch: Partial<CatalogFilterState>) {
    setFilters((prev) => ({ ...prev, ...patch }))
    setVisibleCount(CATALOG_PAGE_SIZE)
  }

  return (
    <div className="flex flex-col gap-6 lg:flex-row lg:items-start">
      <aside className="hidden w-56 shrink-0 lg:block">
        <ModelCatalogFilters
          billing={filters.billing}
          sort={filters.sort}
          onBillingChange={(billing) => patchFilters({ billing })}
          onSortChange={(sort) => patchFilters({ sort })}
        />
      </aside>

      <div className="min-w-0 flex-1 space-y-5">
        <ModelCatalogToolbar
          models={chatModels}
          search={filters.search}
          family={filters.family}
          cacheOnly={filters.cacheOnly}
          shown={Math.min(visibleCount, filtered.length)}
          total={filtered.length}
          onSearchChange={(search) => patchFilters({ search })}
          onFamilyChange={(family) => patchFilters({ family })}
          onCacheOnlyChange={(cacheOnly) => patchFilters({ cacheOnly })}
          onOpenFilters={() => setDrawerOpen(true)}
        />

        <ModelCatalogGrid
          models={filtered}
          visibleCount={visibleCount}
          isLoading={isLoading}
          emptyMessage={emptyMessage}
          noResultsMessage={t('catalog.no_results')}
          onLoadMore={() => setVisibleCount((n) => n + CATALOG_PAGE_SIZE)}
        />
      </div>

      <Drawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        title={t('catalog.filter_title')}
      >
        <ModelCatalogFilters
          billing={filters.billing}
          sort={filters.sort}
          onBillingChange={(billing) => patchFilters({ billing })}
          onSortChange={(sort) => patchFilters({ sort })}
        />
      </Drawer>
    </div>
  )
}