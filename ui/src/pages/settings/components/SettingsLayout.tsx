import type { ReactNode } from 'react'
import { TabbedPageLayout } from '../../../components/layout/TabbedPageLayout'
import { useTranslation } from '../../../lib/i18n'
import { SETTINGS_TAB_I18N, SETTINGS_TAB_KEYS, type SettingsTabKey } from '../settingsTabs'
import { SettingsTabIcon } from './SettingsNavIcons'

interface SettingsLayoutProps {
  activeTab: SettingsTabKey
  onTabChange: (tab: SettingsTabKey) => void
  children: ReactNode
}

export function SettingsLayout({ activeTab, onTabChange, children }: SettingsLayoutProps) {
  const { t } = useTranslation()

  const tabs = SETTINGS_TAB_KEYS.map((key) => ({
    key,
    label: t(SETTINGS_TAB_I18N[key]),
    icon: <SettingsTabIcon tab={key} />,
  }))

  return (
    <TabbedPageLayout
      tabs={tabs}
      activeTab={activeTab}
      onTabChange={(key) => onTabChange(key as SettingsTabKey)}
      navLabel={t('settings.nav_label')}
    >
      {children}
    </TabbedPageLayout>
  )
}