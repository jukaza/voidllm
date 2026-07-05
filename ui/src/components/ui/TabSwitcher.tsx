import type { ReactNode } from 'react'
import { cn } from '../../lib/utils'

export interface Tab {
  key: string
  label: string
  icon?: ReactNode
}

export interface TabSwitcherProps {
  tabs: Tab[]
  activeKey: string
  onChange: (key: string) => void
  className?: string
  scrollable?: boolean
}

export default function TabSwitcher({
  tabs,
  activeKey,
  onChange,
  className,
  scrollable = false,
}: TabSwitcherProps) {
  return (
    <div
      role="tablist"
      className={cn(
        scrollable ? 'overflow-x-auto max-w-full' : 'inline-flex',
        'rounded-lg bg-bg-tertiary p-1',
        className ?? 'mb-6',
      )}
    >
      <div className={cn('flex gap-1', scrollable && 'min-w-max')}>
        {tabs.map((tab) => (
          <button
            key={tab.key}
            type="button"
            role="tab"
            aria-selected={tab.key === activeKey}
            onClick={() => onChange(tab.key)}
            className={cn(
              'inline-flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-md transition-all duration-200 whitespace-nowrap shrink-0',
              tab.key === activeKey
                ? 'bg-bg-secondary text-text-primary shadow-sm'
                : 'text-text-tertiary hover:text-text-secondary',
            )}
          >
            {tab.icon && <span className="opacity-80">{tab.icon}</span>}
            {tab.label}
          </button>
        ))}
      </div>
    </div>
  )
}
