import { Link } from 'react-router-dom'
import { BrandIcon } from '../../components/ui/BrandIcon'
import { usePublicCatalog } from '../../hooks/useProviders'
import { useTranslation } from '../../lib/i18n'

function formatPrice(v: number | null | undefined): string {
  if (v === null || v === undefined) return '—'
  return `$${v.toFixed(2)}`
}

export default function LandingPage() {
  const { data, isLoading } = usePublicCatalog()
  const { t } = useTranslation()
  const models = data?.data ?? []

  return (
    <div className="min-h-screen bg-bg-primary text-text-primary">
      {/* Nav */}
      <header className="border-b border-white/5">
        <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <img src="/logo.svg" alt="VoidLLM" className="h-7 w-7" />
            <span className="gradient-text text-xl font-bold">VoidLLM</span>
          </div>
          <div className="flex items-center gap-3">
            <Link
              to="/login"
              className="text-sm text-text-secondary hover:text-text-primary no-underline"
            >
              {t('storefront.sign_in')}
            </Link>
            <Link
              to="/register"
              className="text-sm bg-accent text-white px-4 py-2 rounded-lg no-underline hover:opacity-90"
            >
              {t('storefront.get_started')}
            </Link>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="max-w-5xl mx-auto px-6 pt-20 pb-14 text-center">
        <h1 className="text-4xl md:text-5xl font-bold leading-tight">
          {t('storefront.hero_title')}
        </h1>
        <p className="mt-4 text-lg text-text-tertiary max-w-2xl mx-auto">
          {t('storefront.hero_subtitle')}
        </p>
        <div className="mt-8 flex items-center justify-center gap-4">
          <Link
            to="/register"
            className="bg-accent text-white px-6 py-3 rounded-lg text-sm font-medium no-underline hover:opacity-90"
          >
            {t('storefront.cta_signup')}
          </Link>
          <a
            href="#pricing"
            className="border border-white/10 px-6 py-3 rounded-lg text-sm text-text-secondary no-underline hover:text-text-primary"
          >
            {t('storefront.cta_pricing')}
          </a>
        </div>
      </section>

      {/* How it works */}
      <section className="max-w-5xl mx-auto px-6 py-10 grid md:grid-cols-3 gap-6">
        {[
          { title: t('storefront.step1_title'), desc: t('storefront.step1_desc') },
          { title: t('storefront.step2_title'), desc: t('storefront.step2_desc') },
          { title: t('storefront.step3_title'), desc: t('storefront.step3_desc') },
        ].map((s, i) => (
          <div key={i} className="bg-bg-secondary border border-white/5 rounded-xl p-6">
            <div className="w-8 h-8 rounded-full bg-accent/15 text-accent flex items-center justify-center font-bold mb-3">
              {i + 1}
            </div>
            <h3 className="font-semibold">{s.title}</h3>
            <p className="mt-1 text-sm text-text-tertiary">{s.desc}</p>
          </div>
        ))}
      </section>

      {/* Pricing table */}
      <section id="pricing" className="max-w-5xl mx-auto px-6 py-14">
        <h2 className="text-2xl font-bold text-center">{t('storefront.pricing_title')}</h2>
        <p className="mt-2 text-sm text-text-tertiary text-center">
          {t('storefront.pricing_subtitle')}
        </p>

        <div className="mt-8 overflow-x-auto rounded-xl border border-white/5">
          <table className="w-full text-sm">
            <thead className="bg-bg-secondary">
              <tr className="text-left text-text-tertiary">
                <th className="px-4 py-3 font-medium">{t('storefront.col_model')}</th>
                <th className="px-4 py-3 font-medium">{t('storefront.col_type')}</th>
                <th className="px-4 py-3 font-medium text-right">{t('storefront.col_input')}</th>
                <th className="px-4 py-3 font-medium text-right">{t('storefront.col_output')}</th>
                <th className="px-4 py-3 font-medium text-right">Per request</th>
              </tr>
            </thead>
            <tbody>
              {isLoading && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-text-tertiary">
                    ...
                  </td>
                </tr>
              )}
              {!isLoading && models.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-text-tertiary">
                    {t('storefront.no_models')}
                  </td>
                </tr>
              )}
              {models.map((m) => (
                <tr key={m.name} className="border-t border-white/5">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2.5 font-mono">
                      <BrandIcon logo={m.logo} modelName={m.name} size={18} />
                      <span>{m.name}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-text-tertiary">{m.type}</td>
                  <td className="px-4 py-3 text-right">{m.bill_per_token ? formatPrice(m.sell_input_per_1m) : '—'}</td>
                  <td className="px-4 py-3 text-right">{m.bill_per_token ? formatPrice(m.sell_output_per_1m) : '—'}</td>
                  <td className="px-4 py-3 text-right text-text-tertiary">
                    {m.bill_per_request ? formatPrice(m.sell_per_request) : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="mt-3 text-xs text-text-tertiary text-center">
          {t('storefront.pricing_note')}
        </p>
      </section>

      {/* Footer */}
      <footer className="border-t border-white/5 py-8 text-center text-xs text-text-tertiary">
        VoidLLM — OpenAI-compatible API marketplace
      </footer>
    </div>
  )
}
