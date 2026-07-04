import { cn } from '../../lib/utils'
import { formatLatencySeconds, latencyVariant, type LatencyVariant } from '../../lib/latency'

const VARIANT_STYLES: Record<LatencyVariant, string> = {
  fast: 'border-emerald-500/35 bg-emerald-500/10 text-emerald-400',
  warn: 'border-amber-500/40 bg-amber-500/10 text-amber-400',
  slow: 'border-rose-500/45 bg-rose-500/10 text-rose-400',
  unknown: 'border-border bg-bg-tertiary/40 text-text-tertiary',
}

interface LatencyBadgeProps {
  durationMs: number
  ttftMs?: number | null
  isStream?: boolean
  className?: string
}

export function LatencyBadge({ durationMs, ttftMs, isStream, className }: LatencyBadgeProps) {
  const variant = latencyVariant(durationMs)
  const label = formatLatencySeconds(durationMs)

  if (label === '—') {
    return <span className="text-text-tertiary text-xs">—</span>
  }

  return (
    <div className={cn('inline-flex flex-col items-end gap-0.5', className)}>
      <span
        className={cn(
          'inline-flex items-center rounded-md border px-1.5 py-0.5 font-mono text-xs tabular-nums',
          VARIANT_STYLES[variant],
        )}
      >
        {label}
      </span>
      {isStream && ttftMs != null && ttftMs > 0 && (
        <span className="font-mono text-[10px] text-text-tertiary tabular-nums">
          TTFT {(ttftMs / 1000).toFixed(1)}s
        </span>
      )}
    </div>
  )
}