import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { PageHeader } from '../../components/ui/PageHeader'
import { useMe } from '../../hooks/useMe'
import { useTranslation } from '../../lib/i18n'
import { ConnectionsTab } from './ConnectionsTab'
import { PreferencesTab } from './PreferencesTab'
import { ProfileTab } from './ProfileTab'
import { SecurityTab } from './SecurityTab'
import { AccountHero } from './components/AccountHero'
import { AccountLayout } from './components/AccountLayout'
import { AccountStatusStrip } from './components/AccountStatusStrip'
import { isAccountTabKey, type AccountTabKey } from './accountTabs'

export default function AccountSettingsPage() {
  const { t } = useTranslation()
  const { isLoading } = useMe()
  const [searchParams, setSearchParams] = useSearchParams()

  const tabParam = searchParams.get('tab')
  const [tab, setTab] = useState<AccountTabKey>(
    isAccountTabKey(tabParam) ? tabParam : 'profile',
  )

  useEffect(() => {
    if (isAccountTabKey(tabParam) && tabParam !== tab) {
      setTab(tabParam)
    }
  }, [tabParam, tab])

  function changeTab(next: AccountTabKey) {
    setTab(next)
    const params = new URLSearchParams(searchParams)
    params.set('tab', next)
    setSearchParams(params, { replace: true })
  }

  if (isLoading) {
    return (
      <>
        <PageHeader title={t('account.title')} description={t('account.description')} />
        <div className="max-w-6xl space-y-6">
          <div className="h-40 animate-pulse rounded-lg border border-border bg-bg-secondary" />
          <div className="h-64 animate-pulse rounded-lg border border-border bg-bg-secondary" />
        </div>
      </>
    )
  }

  return (
    <>
      <PageHeader title={t('account.title')} description={t('account.description')} />
      <div className="max-w-6xl">
        <AccountHero />
        <AccountStatusStrip />
        <AccountLayout activeTab={tab} onTabChange={changeTab}>
          {tab === 'profile' && <ProfileTab />}
          {tab === 'security' && <SecurityTab />}
          {tab === 'connections' && <ConnectionsTab />}
          {tab === 'preferences' && <PreferencesTab />}
        </AccountLayout>
      </div>
    </>
  )
}