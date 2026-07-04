import { formatNumber } from '../../lib/utils'
import { useTranslation } from '../../lib/i18n'

interface TokenBreakdownProps {
  promptTokens: number
  completionTokens: number
  cachedTokens?: number
  cacheWriteTokens?: number
  compact?: boolean
}

export function TokenBreakdown({
  promptTokens,
  completionTokens,
  cachedTokens = 0,
  cacheWriteTokens = 0,
  compact = false,
}: TokenBreakdownProps) {
  const { t } = useTranslation()
  const hasCache = cachedTokens > 0 || cacheWriteTokens > 0

  if (promptTokens === 0 && completionTokens === 0 && !hasCache) {
    return <span className="text-text-tertiary text-xs">—</span>
  }

  return (
    <div className={compact ? 'flex flex-col gap-0.5' : 'space-y-1'}>
      <span className="font-mono text-xs tabular-nums text-text-secondary">
        {formatNumber(promptTokens)} / {formatNumber(completionTokens)}
      </span>
      {hasCache && (
        <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[11px] text-text-tertiary">
          {cachedTokens > 0 && (
            <span>
              {t('analytics.cache_read')} ↓ {formatNumber(cachedTokens)}
            </span>
          )}
          {cacheWriteTokens > 0 && (
            <span>
              {t('analytics.cache_write')} ↑ {formatNumber(cacheWriteTokens)}
            </span>
          )}
        </div>
      )}
    </div>
  )
}