import { PageHeader } from '../components/ui/PageHeader'
import { BrandIcon } from '../components/ui/BrandIcon'
import { usePublicCatalog } from '../hooks/useProviders'
import { useTranslation } from '../lib/i18n'

function formatPrice(v: number | null | undefined): string {
  if (v === null || v === undefined) return '—'
  return `$${v.toFixed(2)}`
}

export default function CatalogPage() {
  const { data, isLoading } = usePublicCatalog()
  const { t } = useTranslation()
  const models = data?.data ?? []

  return (
    <>
      <PageHeader title={t('sidebar.catalog')} description={t('catalog.desc')} />

      {isLoading ? (
        <div className="rounded-lg border border-border bg-bg-secondary p-12 text-center">
          <p className="text-sm text-text-tertiary">{t('common.loading')}</p>
        </div>
      ) : models.length === 0 ? (
        <div className="rounded-lg border border-border bg-bg-secondary p-12 text-center">
          <p className="text-sm text-text-tertiary">{t('catalog.empty')}</p>
        </div>
      ) : (
        <div className="rounded-xl border border-border overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-bg-secondary text-text-tertiary text-left">
                <th className="px-4 py-3 font-medium">{t('storefront.col_model')}</th>
                <th className="px-4 py-3 font-medium">{t('storefront.col_type')}</th>
                <th className="px-4 py-3 font-medium text-right">{t('storefront.col_input')} / 1M</th>
                <th className="px-4 py-3 font-medium text-right">{t('storefront.col_output')} / 1M</th>
                <th className="px-4 py-3 font-medium text-right">{t('catalog.col_per_request')}</th>
              </tr>
            </thead>
            <tbody>
              {models.map((m) => (
                <tr key={m.name} className="border-t border-border hover:bg-bg-secondary/50">
                  <td className="px-4 py-3 font-medium text-text-primary">
                    <div className="flex items-center gap-2.5">
                      <BrandIcon logo={m.logo} modelName={m.name} size={20} />
                      <span>{m.name}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-text-secondary">{m.type}</td>
                  <td className="px-4 py-3 text-right tabular-nums text-text-secondary">
                    {m.bill_per_token ? formatPrice(m.sell_input_per_1m) : '—'}
                  </td>
                  <td className="px-4 py-3 text-right tabular-nums text-text-secondary">
                    {m.bill_per_token ? formatPrice(m.sell_output_per_1m) : '—'}
                  </td>
                  <td className="px-4 py-3 text-right tabular-nums text-text-secondary">
                    {m.bill_per_request ? formatPrice(m.sell_per_request) : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  )
}