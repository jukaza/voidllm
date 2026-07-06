import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { StatCard } from '../../components/ui/StatCard'
import { AreaChart, DonutChart, HorizontalBar, type DonutSegment } from '../../components/ui/charts'
import { useMe } from '../../hooks/useMe'
import { useMyUsage, type MyUsageFilters, type UsageDataPoint } from '../../hooks/useUsage'
import { useAPIKeys } from '../../hooks/useAPIKeys'
import { formatCost, formatNumber, formatTokens } from '../../lib/utils'
import { useTranslation } from '../../lib/i18n'
import { useUsageLive } from '../../hooks/useUsageLive'
import { PillGroup } from '../../components/ui/PillGroup'
import { TOKEN_CHART_COLORS, TOKEN_METRIC_STYLES, tokenBarSegments } from '../../lib/tokenColors'

type PeriodDays = 1 | 7 | 15 | 30
type ChartMetric = 'requests' | 'payment' | 'tokens'

const PERIOD_OPTIONS: PeriodDays[] = [1, 7, 15, 30]
const METRIC_COLORS: Record<ChartMetric, string> = {
  requests: '#8b5cf6',
  payment: '#10b981',
  tokens: '#3b82f6',
}

function getRange(days: PeriodDays): { from: string; to: string } {
  const to = new Date()
  const from = new Date(to.getTime() - days * 24 * 3_600_000)
  return { from: from.toISOString(), to: to.toISOString() }
}

function formatChartLabel(key: string, byHour: boolean): string {
  if (!key) return ''
  if (byHour) {
    const d = new Date(key)
    if (!Number.isNaN(d.getTime())) {
      return d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
    }
  }
  const datePart = key.slice(0, 10)
  const [y, m, d] = datePart.split('-')
  if (y && m && d) return `${d}/${m}`
  return datePart
}

function metricValue(row: UsageDataPoint, metric: ChartMetric): number {
  switch (metric) {
    case 'payment':
      return row.revenue
    case 'tokens':
      return row.total_tokens
    default:
      return row.total_requests
  }
}

function formatMetric(n: number, metric: ChartMetric): string {
  if (metric === 'payment') return formatCost(n)
  if (metric === 'tokens') return formatTokens(n)
  return formatNumber(n)
}

export default function AnalyticsOverviewPage() {
  const { data: me } = useMe()
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const isAdmin = me?.is_system_admin ?? false

  const [periodDays, setPeriodDays] = useState<PeriodDays>(7)
  const [chartMetric, setChartMetric] = useState<ChartMetric>('requests')
  const [keyId, setKeyId] = useState('')
  const [model, setModel] = useState('')

  useEffect(() => {
    const fromUrl = searchParams.get('key_id')
    if (fromUrl) setKeyId(fromUrl)
  }, [searchParams])

  const { from, to } = useMemo(() => getRange(periodDays), [periodDays])
  const filters = useMemo<MyUsageFilters>(
    () => ({
      keyId: keyId || undefined,
      model: model || undefined,
    }),
    [keyId, model],
  )
  const trendGroupBy = periodDays === 1 ? 'hour' : 'day'

  const totals = useMyUsage(from, to, '', filters)
  const trend = useMyUsage(from, to, trendGroupBy, filters)
  const byModel = useMyUsage(from, to, 'model', filters)
  const modelOptions = useMyUsage(from, to, 'model', { keyId: keyId || undefined })

  const { data: keysData } = useAPIKeys()

  const summary = useMemo(() => {
    const rows = totals.data?.data ?? []
    if (rows.length === 0) {
      return {
        requests: 0,
        promptTokens: 0,
        completionTokens: 0,
        cachedTokens: 0,
        revenue: 0,
      }
    }
    const row = rows[0]
    return {
      requests: row.total_requests,
      promptTokens: row.prompt_tokens,
      completionTokens: row.completion_tokens,
      cachedTokens: row.cached_tokens,
      revenue: row.revenue,
    }
  }, [totals.data])

  const chartData = useMemo(() => {
    return (trend.data?.data ?? []).map((d) => ({
      label: formatChartLabel(d.group_key ?? '', trendGroupBy === 'hour'),
      value: metricValue(d, chartMetric),
    }))
  }, [trend.data, chartMetric, trendGroupBy])

  const topModels = useMemo(() => {
    return [...(byModel.data?.data ?? [])]
      .sort((a, b) => b.total_tokens - a.total_tokens)
      .slice(0, 6)
  }, [byModel.data])

  const donutSegments = useMemo(() => {
    if (topModels.length === 0) return []
    const prompt = topModels.reduce((acc, m) => acc + m.prompt_tokens, 0)
    const cached = topModels.reduce((acc, m) => acc + (m.cached_tokens ?? 0), 0)
    const completion = topModels.reduce((acc, m) => acc + m.completion_tokens, 0)
    const freshInput = Math.max(0, prompt - cached)
    if (prompt + completion === 0) return []

    const segments: DonutSegment[] = [
      { label: t('analytics.tokens_input'), value: freshInput, color: TOKEN_CHART_COLORS.input },
      { label: t('analytics.tokens_output'), value: completion, color: TOKEN_CHART_COLORS.output },
    ]
    if (cached > 0) {
      segments.splice(1, 0, {
        label: t('analytics.cache_read'),
        value: cached,
        color: TOKEN_CHART_COLORS.cacheRead,
      })
    }
    return segments.filter((s) => s.value > 0)
  }, [topModels, t])

  const periodLabel = useMemo(() => {
    const map: Record<PeriodDays, string> = {
      1: t('analytics.period_1d'),
      7: t('analytics.period_7d'),
      15: t('analytics.period_15d'),
      30: t('analytics.period_30d'),
    }
    return map[periodDays]
  }, [periodDays, t])

  const loading = totals.isLoading
  const { live, connected } = useUsageLive()

  const metricOptions = [
    { value: 'requests' as ChartMetric, label: t('analytics.metric_requests') },
    { value: 'payment' as ChartMetric, label: t('analytics.metric_payment') },
    { value: 'tokens' as ChartMetric, label: t('analytics.metric_tokens') },
  ]

  return (
    <div className="flex flex-col gap-6">
      {isAdmin && connected && live && (
        <div className="flex flex-wrap items-center gap-3 text-sm rounded-xl border border-border bg-bg-secondary px-4 py-3">
          <span className="inline-flex items-center gap-1.5 text-emerald-400 font-medium">
            <span className="h-2 w-2 rounded-full bg-emerald-400 animate-pulse" />
            {t('analytics.live')}
          </span>
          <span className="text-text-secondary">
            {t('analytics.rpm')}: {formatNumber(live.rpm)}
          </span>
          <span className="text-text-secondary">
            {t('analytics.tpm')}: {formatNumber(live.tpm)}
          </span>
          <span className="text-text-secondary">
            {t('analytics.active')}: {formatNumber(live.active_count)}
          </span>
        </div>
      )}

      <div className="flex flex-wrap items-end gap-4 rounded-xl border border-border bg-bg-secondary p-4">
        <PillGroup
          label={t('analytics.period')}
          options={PERIOD_OPTIONS.map((d) => ({
            value: d,
            label: d === 1 ? '1d' : `${d}d`,
          }))}
          value={periodDays}
          onChange={setPeriodDays}
        />
        <div>
          <label className="text-xs text-text-tertiary block mb-1">{t('analytics.filter_key')}</label>
          <select
            className="bg-bg-primary border border-border rounded-lg px-3 py-1.5 text-sm min-w-[160px]"
            value={keyId}
            onChange={(e) => setKeyId(e.target.value)}
          >
            <option value="">{t('analytics.all_keys')}</option>
            {(keysData?.data ?? []).map((k) => (
              <option key={k.id} value={k.id}>
                {k.name || k.key_hint}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="text-xs text-text-tertiary block mb-1">{t('analytics.filter_model')}</label>
          <select
            className="bg-bg-primary border border-border rounded-lg px-3 py-1.5 text-sm min-w-[180px]"
            value={model}
            onChange={(e) => setModel(e.target.value)}
          >
            <option value="">{t('analytics.all_models')}</option>
            {(modelOptions.data?.data ?? []).map((m) => (
              <option key={m.group_key} value={m.group_key}>
                {m.group_label || m.group_key}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="grid gap-4 grid-cols-2 lg:grid-cols-5">
        <StatCard
          label={`${t('analytics.requests')} (${periodLabel})`}
          value={loading ? '...' : formatNumber(summary.requests)}
        />
        <StatCard
          label={`${t('analytics.tokens_input')} (${periodLabel})`}
          value={loading ? '...' : formatTokens(summary.promptTokens)}
          valueStyle={{ color: TOKEN_METRIC_STYLES.input.color }}
        />
        <StatCard
          label={`${t('analytics.tokens_output')} (${periodLabel})`}
          value={loading ? '...' : formatTokens(summary.completionTokens)}
          valueStyle={{ color: TOKEN_METRIC_STYLES.output.color }}
        />
        <StatCard
          label={`${t('analytics.tokens_cached')} (${periodLabel})`}
          value={loading ? '...' : formatTokens(summary.cachedTokens)}
          valueStyle={{ color: TOKEN_METRIC_STYLES.cacheRead.color }}
        />
        <StatCard
          label={`${t('analytics.payment')} (${periodLabel})`}
          value={loading ? '...' : formatCost(summary.revenue)}
        />
      </div>

      <div className="bg-bg-secondary rounded-xl border border-border p-6">
        <div className="flex flex-wrap items-center justify-between gap-3 mb-6">
          <h3 className="text-sm font-semibold text-text-primary">{t('analytics.trend')}</h3>
          <PillGroup options={metricOptions} value={chartMetric} onChange={setChartMetric} />
        </div>
        {trend.isLoading ? (
          <div className="rounded-lg bg-bg-tertiary animate-pulse h-[240px]" />
        ) : chartData.length > 0 ? (
          <AreaChart
            data={chartData}
            height={240}
            color={METRIC_COLORS[chartMetric]}
            formatValue={(n) => formatMetric(n, chartMetric)}
            valueLabel={metricOptions.find((m) => m.value === chartMetric)?.label}
          />
        ) : (
          <div className="flex items-center justify-center h-[240px] text-text-tertiary text-sm">
            {t('analytics.no_data')}
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-bg-secondary rounded-xl border border-border p-6">
          <h3 className="text-sm font-semibold text-text-primary mb-6">{t('analytics.top_models')}</h3>
          {byModel.isLoading ? (
            <div className="rounded-lg bg-bg-tertiary animate-pulse h-[200px]" />
          ) : topModels.length > 0 ? (
            <HorizontalBar
              items={topModels.map((m) => ({
                label: m.group_label || m.group_key,
                value: m.total_tokens,
                detail: formatTokens(m.total_tokens),
                segments: tokenBarSegments(m.prompt_tokens, m.completion_tokens, m.cached_tokens ?? 0),
              }))}
            />
          ) : (
            <p className="text-text-tertiary text-sm">{t('analytics.no_data')}</p>
          )}
        </div>

        <div className="bg-bg-secondary rounded-xl border border-border p-6">
          <h3 className="text-sm font-semibold text-text-primary mb-6">{t('analytics.token_split')}</h3>
          {donutSegments.length > 0 ? (
            <DonutChart segments={donutSegments} size={160} />
          ) : (
            <p className="text-text-tertiary text-sm">{t('analytics.no_data')}</p>
          )}
        </div>
      </div>
    </div>
  )
}