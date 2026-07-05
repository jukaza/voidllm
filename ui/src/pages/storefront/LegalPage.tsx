import { Link, useParams } from 'react-router-dom'
import { Markdown } from '../../components/ui/Markdown'
import { BrandMark } from '../../components/brand/BrandMark'
import { useSiteConfig } from '../../hooks/useSiteConfig'
import { useTranslation } from '../../lib/i18n'

type LegalKind = 'terms' | 'privacy'

export default function LegalPage() {
  const { kind } = useParams<{ kind: LegalKind }>()
  const { data, isLoading } = useSiteConfig()
  const { t } = useTranslation()

  const isTerms = kind === 'terms'
  const title = isTerms ? t('legal.terms_title') : t('legal.privacy_title')
  const content = isTerms ? data?.user_agreement : data?.privacy_policy

  return (
    <div className="min-h-screen bg-bg-primary text-text-primary">
      <header className="border-b border-white/5">
        <div className="max-w-3xl mx-auto px-6 py-4 flex items-center justify-between">
          <Link to="/" className="no-underline">
            <BrandMark />
          </Link>
          <Link to="/login" className="text-sm text-text-secondary hover:text-text-primary no-underline">
            {t('storefront.sign_in')}
          </Link>
        </div>
      </header>

      <main className="max-w-3xl mx-auto px-6 py-10">
        <h1 className="text-2xl font-bold mb-6">{title}</h1>
        {isLoading && <p className="text-sm text-text-tertiary">{t('common.loading')}</p>}
        {!isLoading && content?.trim() ? (
          <div className="rounded-xl border border-white/5 bg-bg-secondary p-6">
            <Markdown>{content}</Markdown>
          </div>
        ) : (
          <p className="text-sm text-text-tertiary">{t('legal.not_available')}</p>
        )}
      </main>
    </div>
  )
}