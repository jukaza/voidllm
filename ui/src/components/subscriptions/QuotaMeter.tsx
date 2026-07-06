import { cn, formatNumber } from '../../lib/utils'

export function QuotaMeter({
  label,
  used,
  limit,
  className,
  variant = 'inline',
}: {
  label: string
  used: number
  limit: number
  className?: string
  variant?: 'inline' | 'card'
}) {
  if (limit <= 0) return null
  const pct = Math.min(100, Math.round((used / limit) * 100))
  const warn = pct >= 85
  const critical = pct >= 95

  const bar = (
    <div
      className={cn(
        'overflow-hidden rounded-full bg-bg-tertiary',
        variant === 'card' ? 'h-2' : 'h-1.5',
      )}
    >
      <div
        className={cn(
          'h-full rounded-full transition-all duration-500',
          critical ? 'bg-error' : warn ? 'bg-amber-500' : 'bg-accent',
        )}
        style={{ width: `${Math.max(pct, 2)}%` }}
      />
    </div>
  )

  if (variant === 'card') {
    return (
      <div
        className={cn(
          'rounded-xl border border-border/70 bg-bg-primary/50 p-4 shadow-sm',
          className,
        )}
      >
        <div className="mb-2.5 flex items-center justify-between gap-2">
          <span className="text-[10px] font-semibold uppercase tracking-wide text-text-tertiary">
            {label}
          </span>
          <span
            className={cn(
              'rounded-md px-1.5 py-0.5 text-[10px] font-bold tabular-nums',
              critical
                ? 'bg-error/15 text-error'
                : warn
                  ? 'bg-warning/15 text-warning'
                  : 'bg-accent/15 text-accent',
            )}
          >
            {pct}%
          </span>
        </div>
        {bar}
        <p className="mt-2.5 text-sm font-semibold tabular-nums text-text-primary">
          {formatNumber(used)}
          <span className="font-normal text-text-tertiary"> / {formatNumber(limit)}</span>
        </p>
      </div>
    )
  }

  return (
    <div className={cn('space-y-1', className)}>
      <div className="flex items-center justify-between text-[11px]">
        <span className="text-text-tertiary">{label}</span>
        <span
          className={cn(
            'tabular-nums font-medium',
            critical ? 'text-error' : warn ? 'text-amber-400' : 'text-text-secondary',
          )}
        >
          {formatNumber(used)} / {formatNumber(limit)}
        </span>
      </div>
      {bar}
    </div>
  )
}