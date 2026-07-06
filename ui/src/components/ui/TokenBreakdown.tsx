import { formatNumber } from '../../lib/utils'
import { useTranslation } from '../../lib/i18n'
import { TOKEN_METRIC_STYLES, type TokenMetricKey } from '../../lib/tokenColors'

interface TokenBreakdownProps {
  promptTokens: number
  completionTokens: number
  cachedTokens?: number
  cacheWriteTokens?: number
  compact?: boolean
}

function TokenPill({
  label,
  value,
  metric,
}: {
  label: string
  value: number
  metric: TokenMetricKey
}) {
  const style = TOKEN_METRIC_STYLES[metric]
  return (
    <span
      className="inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 font-mono text-[11px] font-semibold tabular-nums"
      style={{
        color: style.color,
        backgroundColor: style.backgroundColor,
        borderColor: `${style.color}55`,
      }}
    >
      <span className="font-sans font-medium opacity-90">{label}</span>
      {formatNumber(value)}
    </span>
  )
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
    <div className={compact ? 'flex flex-col gap-1' : 'flex flex-col gap-1.5'}>
      <div className="flex flex-wrap items-center gap-1">
        <TokenPill label={t('analytics.tokens_input')} value={promptTokens} metric="input" />
        <TokenPill label={t('analytics.tokens_output')} value={completionTokens} metric="output" />
      </div>
      {hasCache && (
        <div className="flex flex-wrap items-center gap-1">
          {cachedTokens > 0 && (
            <TokenPill label={t('analytics.cache_read')} value={cachedTokens} metric="cacheRead" />
          )}
          {cacheWriteTokens > 0 && (
            <TokenPill label={t('analytics.cache_write')} value={cacheWriteTokens} metric="cacheWrite" />
          )}
        </div>
      )}
    </div>
  )
}