import type { ReactNode } from 'react'
import { cn } from '../../lib/utils'
import TabSwitcher from '../ui/TabSwitcher'

export interface TabbedPageTab {
  key: string
  label: string
  icon?: ReactNode
}

interface TabbedPageLayoutProps {
  tabs: TabbedPageTab[]
  activeTab: string
  onTabChange: (key: string) => void
  navLabel: string
  children: ReactNode
  maxWidthClass?: string
}

export function TabbedPageLayout({
  tabs,
  activeTab,
  onTabChange,
  navLabel,
  children,
  maxWidthClass = 'max-w-6xl',
}: TabbedPageLayoutProps) {
  return (
    <div className={cn('mx-auto', maxWidthClass)}>
      <div className="lg:hidden">
        <TabSwitcher
          tabs={tabs}
          activeKey={activeTab}
          onChange={onTabChange}
          scrollable
          className="mb-6 w-full max-w-full"
        />
      </div>

      <div className="flex flex-col gap-6 lg:flex-row lg:items-start">
        <nav
          className="hidden lg:flex lg:w-52 shrink-0 flex-col gap-0.5 rounded-lg border border-border bg-bg-secondary p-1.5"
          aria-label={navLabel}
        >
          {tabs.map((tab) => {
            const active = tab.key === activeTab
            return (
              <button
                key={tab.key}
                type="button"
                onClick={() => onTabChange(tab.key)}
                className={cn(
                  'flex items-center gap-2.5 rounded-md px-3 py-2.5 text-left text-sm font-medium transition-colors',
                  active
                    ? 'bg-bg-tertiary text-text-primary shadow-sm'
                    : 'text-text-tertiary hover:bg-bg-tertiary/60 hover:text-text-secondary',
                )}
              >
                {tab.icon && (
                  <span className={cn('shrink-0', active ? 'text-accent' : 'opacity-70')}>{tab.icon}</span>
                )}
                <span className="truncate">{tab.label}</span>
              </button>
            )
          })}
        </nav>

        <div className="min-w-0 flex-1">{children}</div>
      </div>
    </div>
  )
}