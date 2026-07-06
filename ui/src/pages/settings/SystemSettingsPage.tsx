import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { PageHeader } from '../../components/ui/PageHeader'
import { Button } from '../../components/ui/Button'
import { Banner } from '../../components/ui/Banner'
import { useAdminSiteSettings } from '../../hooks/useSiteConfig'
import { useTranslation } from '../../lib/i18n'
import { SettingsLayout } from './components/SettingsLayout'
import { SettingsStatusStrip } from './components/SettingsStatusStrip'
import { GeneralSettingsTab } from './GeneralSettingsTab'
import { SecuritySettingsTab } from './SecuritySettingsTab'
import { FeaturesSettingsTab } from './FeaturesSettingsTab'
import { LegalNoticeSettingsTab } from './LegalNoticeSettingsTab'
import { BackupSettingsTab } from './BackupSettingsTab'
import { PaymentSettingsTab } from './PaymentSettingsTab'
import { KeysSettingsTab } from './KeysSettingsTab'

import { isSettingsTabKey, type SettingsTabKey } from './settingsTabs'

export default function SystemSettingsPage() {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()
  const { isLoading, isError, error, refetch } = useAdminSiteSettings()

  const tabParam = searchParams.get('tab')
  const [tab, setTab] = useState<SettingsTabKey>(
    isSettingsTabKey(tabParam) ? tabParam : 'general',
  )

  useEffect(() => {
    if (isSettingsTabKey(tabParam) && tabParam !== tab) {
      setTab(tabParam)
    }
  }, [tabParam, tab])

  function changeTab(next: SettingsTabKey) {
    setTab(next)
    const params = new URLSearchParams(searchParams)
    params.set('tab', next)
    setSearchParams(params, { replace: true })
  }

  if (isLoading) {
    return (
      <>
        <PageHeader title={t('settings.title')} description={t('settings.description')} />
        <p className="text-sm text-text-tertiary">{t('common.loading')}</p>
      </>
    )
  }

  if (isError) {
    return (
      <>
        <PageHeader title={t('settings.title')} description={t('settings.description')} />
        <Banner
          variant="error"
          title={t('settings.load_error')}
          description={error?.message ?? t('settings.load_error_hint')}
        />
        <Button className="mt-4" variant="secondary" onClick={() => void refetch()}>
          {t('common.refresh')}
        </Button>
      </>
    )
  }

  return (
    <>
      <PageHeader title={t('settings.title')} description={t('settings.description')} />
      <SettingsStatusStrip />
      <SettingsLayout activeTab={tab} onTabChange={changeTab}>
        {tab === 'general' && <GeneralSettingsTab />}
        {tab === 'security' && <SecuritySettingsTab />}
        {tab === 'features' && <FeaturesSettingsTab />}
        {tab === 'keys' && <KeysSettingsTab />}
        {tab === 'payment' && <PaymentSettingsTab />}
        {tab === 'legal' && <LegalNoticeSettingsTab />}
        {tab === 'backup' && <BackupSettingsTab />}
      </SettingsLayout>
    </>
  )
}