import type { ReactNode } from 'react'
import { CatalogTooltip } from './CatalogTooltip'
import { cn } from '../../lib/utils'

export type CapabilityVariant = 'tools' | 'vision' | 'cache'

const ACTIVE_STYLES: Record<CapabilityVariant, string> = {
  tools: 'border-info/30 bg-info/10 text-info',
  vision: 'border-accent/30 bg-accent/10 text-accent',
  cache: 'border-success/30 bg-success/10 text-success',
}

interface CatalogCapabilityIconProps {
  label: string
  tooltip?: ReactNode
  active: boolean
  variant: CapabilityVariant
  children: ReactNode
}

export function CatalogCapabilityIcon({
  label,
  tooltip,
  active,
  variant,
  children,
}: CatalogCapabilityIconProps) {
  const button = (
    <button
      tabIndex={0}
      type="button"
      aria-label={label}
      className={cn(
        'inline-flex h-[17px] w-[17px] shrink-0 items-center justify-center rounded-md border transition-colors',
        active
          ? ACTIVE_STYLES[variant]
          : 'border-border/80 bg-bg-primary/40 text-text-tertiary',
        tooltip ? 'cursor-help' : 'cursor-default',
      )}
    >
      <span
        className={cn(
          'inline-flex [&_svg]:h-3 [&_svg]:w-3',
          active ? undefined : 'opacity-50',
        )}
      >
        {children}
      </span>
    </button>
  )

  if (!tooltip) return button
  return <CatalogTooltip content={tooltip}>{button}</CatalogTooltip>
}