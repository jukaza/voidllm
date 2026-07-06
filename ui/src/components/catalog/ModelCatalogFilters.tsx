import type { BillingFilter, CatalogSort } from '../../lib/catalog-utils'
import { useTranslation } from '../../lib/i18n'
import { cn } from '../../lib/utils'

interface ModelCatalogFiltersProps {
  billing: BillingFilter
  sort: CatalogSort
  onBillingChange: (value: BillingFilter) => void
  onSortChange: (value: CatalogSort) => void
  className?: string
}

function FilterSection<T extends string>({
  title,
  value,
  options,
  onChange,
}: {
  title: string
  value: T
  options: Array<{ value: T; label: string }>
  onChange: (value: T) => void
}) {
  return (
    <div>
      <p className="mb-1.5 text-[10px] font-semibold uppercase tracking-wider text-text-tertiary">
        {title}
      </p>
      <div className="flex flex-col gap-1">
        {options.map((opt) => {
          const active = value === opt.value
          return (
            <button
              key={opt.value}
              type="button"
              onClick={() => onChange(opt.value)}
              className={cn(
                'h-8 w-full rounded-lg border px-2.5 text-left text-xs font-medium transition-colors',
                active
                  ? 'border-accent/35 bg-accent/15 text-accent'
                  : 'border-border/80 bg-bg-primary/30 text-text-secondary hover:border-white/10 hover:bg-bg-tertiary hover:text-text-primary',
              )}
            >
              <span className="block truncate">{opt.label}</span>
            </button>
          )
        })}
      </div>
    </div>
  )
}

export function ModelCatalogFilters({
  billing,
  sort,
  onBillingChange,
  onSortChange,
  className,
}: ModelCatalogFiltersProps) {
  const { t } = useTranslation()

  return (
    <div className={className}>
      <div className="space-y-4 rounded-xl border border-border bg-bg-secondary/50 p-3">
        <FilterSection
          title={t('catalog.billing_label')}
          value={billing}
          onChange={onBillingChange}
          options={[
            { value: 'all', label: t('catalog.billing_all') },
            { value: 'token', label: t('catalog.billing_token') },
            { value: 'request', label: t('catalog.billing_request') },
          ]}
        />
        <FilterSection
          title={t('catalog.sort_label')}
          value={sort}
          onChange={onSortChange}
          options={[
            { value: 'name', label: t('catalog.sort_name') },
            { value: 'price_asc', label: t('catalog.sort_price_asc') },
            { value: 'price_desc', label: t('catalog.sort_price_desc') },
          ]}
        />
      </div>
    </div>
  )
}