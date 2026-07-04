import { BrandIcon } from '../ui/BrandIcon'
import { PriceCell, PriceColumnHeader } from '../ui/PriceDisplay'
import type { CatalogModelItem } from '../../hooks/useProviders'
import { useTranslation } from '../../lib/i18n'
import { cn } from '../../lib/utils'

interface PublicPricingTableProps {
  models: CatalogModelItem[]
  isLoading?: boolean
  emptyMessage: string
  variant?: 'admin' | 'storefront'
}

export function PublicPricingTable({
  models,
  isLoading = false,
  emptyMessage,
  variant = 'admin',
}: PublicPricingTableProps) {
  const { t } = useTranslation()
  const isStorefront = variant === 'storefront'

  return (
    <div
      className={cn(
        'overflow-x-auto rounded-xl border',
        isStorefront ? 'border-white/5' : 'border-border',
      )}
    >
      <table className="w-full text-sm min-w-[640px]">
        <thead className={isStorefront ? 'bg-bg-secondary' : 'bg-bg-secondary/80'}>
          <tr className="text-text-tertiary text-left">
            <PriceColumnHeader label={t('storefront.col_model')} align="left" />
            <PriceColumnHeader label={t('storefront.col_type')} align="left" />
            <PriceColumnHeader
              label={t('storefront.col_input')}
              unit={t('common.price_per_1m_unit')}
              align="right"
            />
            <PriceColumnHeader
              label={t('storefront.col_output')}
              unit={t('common.price_per_1m_unit')}
              align="right"
            />
            <PriceColumnHeader
              label={t('catalog.col_per_request')}
              unit={t('common.price_flat_unit')}
              align="right"
            />
          </tr>
        </thead>
        <tbody>
          {isLoading && (
            <tr>
              <td colSpan={5} className="px-4 py-10 text-center text-text-tertiary">
                {t('common.loading')}
              </td>
            </tr>
          )}
          {!isLoading && models.length === 0 && (
            <tr>
              <td colSpan={5} className="px-4 py-10 text-center text-text-tertiary">
                {emptyMessage}
              </td>
            </tr>
          )}
          {!isLoading &&
            models.map((m) => (
              <tr
                key={m.name}
                className={cn(
                  'border-t transition-colors',
                  isStorefront
                    ? 'border-white/5 hover:bg-bg-secondary/40'
                    : 'border-border hover:bg-bg-secondary/50',
                )}
              >
                <td className="px-4 py-3.5 font-medium text-text-primary">
                  <div className="flex items-center gap-2.5">
                    <BrandIcon logo={m.logo} modelName={m.name} size={isStorefront ? 18 : 20} />
                    <span className={isStorefront ? 'font-mono' : undefined}>{m.name}</span>
                  </div>
                </td>
                <td className="px-4 py-3.5 text-text-secondary capitalize">{m.type}</td>
                <td className="px-4 py-3.5 text-right">
                  <PriceCell value={m.bill_per_token ? m.sell_input_per_1m : null} />
                </td>
                <td className="px-4 py-3.5 text-right">
                  <PriceCell value={m.bill_per_token ? m.sell_output_per_1m : null} />
                </td>
                <td className="px-4 py-3.5 text-right">
                  <PriceCell value={m.bill_per_request ? m.sell_per_request : null} />
                </td>
              </tr>
            ))}
        </tbody>
      </table>
    </div>
  )
}