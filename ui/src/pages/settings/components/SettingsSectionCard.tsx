import type { ReactNode } from 'react'
import { cn } from '../../../lib/utils'

interface SettingsSectionCardProps {
  title: string
  description?: string
  badge?: ReactNode
  children: ReactNode
  className?: string
}

export function SettingsSectionCard({
  title,
  description,
  badge,
  children,
  className,
}: SettingsSectionCardProps) {
  return (
    <div className={cn('rounded-lg border border-border bg-bg-secondary mb-0', className)}>
      <div className="flex items-start justify-between gap-3 border-b border-border px-6 py-4">
        <div className="min-w-0">
          <h2 className="text-sm font-semibold text-text-primary">{title}</h2>
          {description && <p className="mt-1 text-xs text-text-tertiary">{description}</p>}
        </div>
        {badge}
      </div>
      <div className="space-y-5 p-6">{children}</div>
    </div>
  )
}