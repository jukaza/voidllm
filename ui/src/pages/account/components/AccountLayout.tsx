import type { ReactNode } from 'react'
import { TabbedPageLayout } from '../../../components/layout/TabbedPageLayout'
import { useTranslation } from '../../../lib/i18n'
import { ACCOUNT_TAB_I18N, ACCOUNT_TAB_KEYS, type AccountTabKey } from '../accountTabs'
import { AccountTabIcon } from './AccountNavIcons'

interface AccountLayoutProps {
  activeTab: AccountTabKey
  onTabChange: (tab: AccountTabKey) => void
  children: ReactNode
}

export function AccountLayout({ activeTab, onTabChange, children }: AccountLayoutProps) {
  const { t } = useTranslation()

  const tabs = ACCOUNT_TAB_KEYS.map((key) => ({
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