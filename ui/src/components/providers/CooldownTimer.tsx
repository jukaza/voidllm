import { useEffect, useState } from 'react'
import { cn } from '../../lib/utils'
import { useTranslation } from '../../lib/i18n'

function formatRemaining(ms: number): string {
  const totalSec = Math.ceil(ms / 1000)
  if (totalSec < 60) return `${totalSec}s`
  const min = Math.floor(totalSec / 60)
  const sec = totalSec % 60
  if (min < 60) return sec > 0 ? `${min}m ${sec}s` : `${min}m`
  const hr = Math.floor(min / 60)
  const remMin = min % 60
  return remMin > 0 ? `${hr}h ${remMin}m` : `${hr}h`
}

export interface CooldownTimerProps {
  /** ISO 8601 timestamp when the lock expires. */
  until: string | null | undefined
  className?: string
}

/** Countdown from an ISO lock-until timestamp; hides when expired or invalid. */
export function CooldownTimer({ until, className }: CooldownTimerProps) {
  const { t } = useTranslation()
  const [now, setNow] = useState(() => Date.now())

  const end = until ? Date.parse(until) : Number.NaN
  const hasActiveLock = !Number.isNaN(end) && end > now

  useEffect(() => {
    if (!hasActiveLock) return
    const id = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(id)
  }, [hasActiveLock, until])

  if (!until || Number.isNaN(end)) return null

  const remaining = end - now
  if (remaining <= 0) return null

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 text-xs font-medium text-amber-600 dark:text-amber-400 tabular-nums',
        className,
      )}
      title={new Date(until).toLocaleString()}
    >
      <span aria-hidden="true">⏳</span>
      {t('provider_detail.lock_remaining', { time: formatRemaining(remaining) })}
    </span>
  )
}