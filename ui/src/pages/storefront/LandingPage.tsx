import { Link } from 'react-router-dom'
import { PublicPricingTable } from '../../components/catalog/PublicPricingTable'
import { BrandMark } from '../../components/brand/BrandMark'
import { SiteNoticeBell } from '../../components/site/SiteNoticeBell'
import { Markdown } from '../../components/ui/Markdown'
import { usePublicCatalog } from '../../hooks/useProviders'
import { useSiteConfig } from '../../hooks/useSiteConfig'
import { useTranslation } from '../../lib/i18n'

export default function LandingPage() {
  const { data, isLoading } = usePublicCatalog()
  const { data: site } = useSiteConfig()
  const { t } = useTranslation()
  const models = data?.data ?? []

  const showRegister = site?.register_enabled !== false
  const footerText = site?.footer?.trim() || `${site?.system_name ?? 'VoidLLM'} — OpenAI-compatible API marketplace`

  return (
    <div className="min-h-screen bg-bg-primary text-text-primary">
      <header className="border-b border-white/5">
        <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between">
          <BrandMark />
          <div className="flex items-center gap-2">
            <SiteNoticeBell />
            <Link
              to="/login"
              className="text-sm text-text-secondary hover:text-text-primary no-underline"
            >
              {t('storefront.sign_in')}
            </Link>
            {showRegister && (
              <Link
                to="/register"
                className="text-sm bg-accent text-white px-4 py-2 rounded-lg no-underline hover:opacity-90"
              >
                {t('storefront.get_started')}
              </Link>
            )}
          </div>
        </div>
      </header>

      <section className="max-w-5xl mx-auto px-6 pt-14 pb-14 text-center">
        <h1 className="text-4xl md:text-5xl font-bold leading-tight">
          {t('storefront.hero_title')}
        </h1>
        <p className="mt-4 text-lg text-text-tertiary max-w-2xl mx-auto">
          {t('storefront.hero_subtitle')}
        </p>
        <div className="mt-8 flex items-center justify-center gap-4">
          {showRegister && (
            <Link
              to="/register"
              className="bg-accent text-white px-6 py-3 rounded-lg text-sm font-medium no-underline hover:opacity-90"
            >
              {t('storefront.cta_signup')}
            </Link>
          )}
          <a
            href="#pricing"
            className="border border-white/10 px-6 py-3 rounded-lg text-sm text-text-secondary no-underline hover:text-text-primary"
          >
            {t('storefront.cta_pricing')}
          </a>
        </div>
      </section>

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

      {site?.home_page_content?.trim() && (
        <section className="max-w-5xl mx-auto px-6 py-10">
          <div className="rounded-xl border border-white/5 bg-bg-secondary p-8">
            <Markdown>{site.home_page_content}</Markdown>
          </div>
        </section>
      )}

      <section id="pricing" className="max-w-5xl mx-auto px-6 py-14">
        <h2 className="text-2xl font-bold text-center">{t('storefront.pricing_title')}</h2>
        <p className="mt-2 text-sm text-text-tertiary text-center">
          {t('storefront.pricing_subtitle')}
        </p>

        <div className="mt-8">
          <PublicPricingTable
            models={models}
            isLoading={isLoading}
            emptyMessage={t('storefront.no_models')}
            variant="storefront"
          />
        </div>
        <p className="mt-3 text-xs text-text-tertiary text-center">
          {t('storefront.pricing_note')}
        </p>
      </section>

      <footer className="border-t border-white/5 py-8 text-center text-xs text-text-tertiary space-y-2">
        <p>{footerText}</p>
        {(site?.user_agreement_enabled || site?.privacy_policy_enabled) && (
          <p className="flex items-center justify-center gap-3">
            {site.user_agreement_enabled && (
              <Link to="/legal/terms" className="text-accent no-underline hover:opacity-80">
                {t('legal.terms_title')}
              </Link>
            )}
            {site.privacy_policy_enabled && (
              <Link to="/legal/privacy" className="text-accent no-underline hover:opacity-80">
                {t('legal.privacy_title')}
              </Link>
            )}
          </p>
        )}
      </footer>
    </div>
  )
}