import type { ReactNode } from 'react'
import { TabbedPageLayout } from '../../../components/layout/TabbedPageLayout'
import { usePublicAuthConfig } from '../../../hooks/useSecuritySettings'
import { useTranslation } from '../../../lib/i18n'
import { hasOAuthLinking } from '../../../lib/oauthProviders'
import { ACCOUNT_TAB_I18N, ACCOUNT_TAB_KEYS, type AccountTabKey } from '../accountTabs'
import { AccountTabIcon } from './AccountNavIcons'

interface AccountLayoutProps {
  activeTab: AccountTabKey
  onTabChange: (tab: AccountTabKey) => void
  children: ReactNode
}

export function AccountLayout({ activeTab, onTabChange, children }: AccountLayoutProps) {
  const { t } = useTranslation()
  const { data: authConfig } = usePublicAuthConfig()

  const tabKeys = ACCOUNT_TAB_KEYS.filter(
    (key) => key !== 'connections' || hasOAuthLinking(authConfig),
  )

  const tabs = tabKeys.map((key) => ({
    key,
    label: t(ACCOUNT_TAB_I18N[key]),
    icon: <AccountTabIcon tab={key} />,
  }))

  return (
    <TabbedPageLayout
      tabs={tabs}
      activeTab={activeTab}
      onTabChange={(key) => onTabChange(key as AccountTabKey)}
      navLabel={t('account.nav_label')}
    >
      {children}
    </TabbedPageLayout>
  )
}