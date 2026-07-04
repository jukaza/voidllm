import { Link, useParams, useSearchParams } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { PageHeader } from '../components/ui/PageHeader'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import TabSwitcher from '../components/ui/TabSwitcher'
import { BrandIcon } from '../components/ui/BrandIcon'
import { ConnectionList } from '../components/providers/ConnectionList'
import { UpstreamModelsSection } from '../components/providers/UpstreamModelsSection'
import { ProviderSettingsDialog } from '../components/providers/ProviderSettingsDialog'
import { Banner } from '../components/ui/Banner'
import { useProvider } from '../hooks/useProviders'
import { useTranslation } from '../lib/i18n'

type DetailTab = 'connections' | 'models'

export default function ProviderDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const { t } = useTranslation()
  const initialTab = searchParams.get('tab') === 'models' ? 'models' : 'connections'
  const [tab, setTab] = useState<DetailTab>(initialTab)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const showSetup = searchParams.get('setup') === '1'

  const { data: provider, isLoading, isError } = useProvider(id ?? '')

  useEffect(() => {
    const qTab = searchParams.get('tab')
    if (qTab === 'models' || qTab === 'connections') {
      setTab(qTab)
    }
  }, [searchParams])

  function dismissSetup() {
    const next = new URLSearchParams(searchParams)
    next.delete('setup')
    setSearchParams(next, { replace: true })
  }

  function changeTab(next: DetailTab) {
    setTab(next)
    const nextParams = new URLSearchParams(searchParams)
    nextParams.set('tab', next)
    nextParams.delete('setup')
    setSearchParams(nextParams, { replace: true })
  }

  if (!id) {
    return (
      <>
        <PageHeader title={t('providers.title')} />
        <p className="text-sm text-text-tertiary">{t('provider_detail.not_found')}</p>
      </>
    )
  }

  if (isLoading) {
    return (
      <>
        <PageHeader title={t('provider_detail.loading')} />
        <div className="rounded-lg border border-border bg-bg-secondary p-12 text-center">
          <p className="text-sm text-text-tertiary">{t('common.loading')}</p>
        </div>
      </>
    )
  }

  if (isError || !provider) {
    return (
      <>
        <PageHeader title={t('provider_detail.not_found')} />
        <div className="rounded-lg border border-border bg-bg-secondary p-8 text-center space-y-4">
          <p className="text-sm text-text-tertiary">{t('provider_detail.not_found')}</p>
          <Link to="/providers" className="inline-block">
            <Button variant="secondary">{t('wizard.back_providers')}</Button>
          </Link>
        </div>
      </>
    )
  }

  const tabs = [
    { key: 'connections', label: t('provider_detail.tab_connections') },
    { key: 'models', label: t('provider_detail.tab_models') },
  ]

  return (
    <>
      <div className="mb-2">
        <Link
          to="/providers"
          className="text-sm text-text-tertiary hover:text-text-secondary no-underline"
        >
          {t('wizard.back_providers')}
        </Link>
      </div>

      <PageHeader
        title={provider.name}
        description={t('provider_detail.desc')}
        actions={
          <div className="flex items-center gap-2">
            <Badge variant={provider.status === 'active' ? 'success' : 'muted'}>
              {provider.status === 'active'
                ? t('providers.status_active')
                : t('providers.status_paused')}
            </Badge>
            <Button variant="secondary" size="sm" onClick={() => setSettingsOpen(true)}>
              {t('provider_detail.edit_settings')}
            </Button>
          </div>
        }
      />

      {showSetup && (
        <div className="mb-4">
          <Banner
            variant="info"
            title={t('provider_detail.setup_banner')}
            onDismiss={dismissSetup}
          />
        </div>
      )}

      <div className="mb-6 rounded-lg border border-border bg-bg-secondary p-4">
        <div className="flex flex-wrap items-start gap-4">
          <BrandIcon
            logo={provider.logo}
            slug={provider.slug}
            protocol={provider.protocol}
            size={40}
          />
          <div className="min-w-0 flex-1 grid gap-3 sm:grid-cols-2 lg:grid-cols-4 text-sm">
            <div>
              <p className="text-xs text-text-tertiary mb-0.5">{t('common.slug')}</p>
              <p className="font-mono text-text-primary">{provider.slug || '—'}</p>
            </div>
            <div>
              <p className="text-xs text-text-tertiary mb-0.5">{t('common.protocol')}</p>
              <p className="text-text-primary">{provider.protocol || '—'}</p>
            </div>
            <div className="sm:col-span-2">
              <p className="text-xs text-text-tertiary mb-0.5">{t('common.endpoint')}</p>
              <p className="font-mono text-xs text-text-primary break-all">
                {provider.base_url || '—'}
              </p>
            </div>
            {provider.contact_info && (
              <div>
                <p className="text-xs text-text-tertiary mb-0.5">
                  {t('marketplace.provider_contact')}
                </p>
                <p className="text-text-primary">{provider.contact_info}</p>
              </div>
            )}
            {provider.notes && (
              <div className="sm:col-span-2">
                <p className="text-xs text-text-tertiary mb-0.5">
                  {t('marketplace.provider_notes')}
                </p>
                <p className="text-text-secondary">{provider.notes}</p>
              </div>
            )}
          </div>
        </div>
      </div>

      <TabSwitcher tabs={tabs} activeKey={tab} onChange={(k) => changeTab(k as DetailTab)} />

      {tab === 'connections' && <ConnectionList providerId={id} />}
      {tab === 'models' && <UpstreamModelsSection providerId={id} />}

      <ProviderSettingsDialog
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        provider={provider}
      />
    </>
  )
}