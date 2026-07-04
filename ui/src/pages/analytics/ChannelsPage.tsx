import { useMemo, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { StatCard } from '../../components/ui/StatCard'
import { Select } from '../../components/ui/Select'
import { useMe } from '../../hooks/useMe'
import { useChannelUsage, type ChannelUsageRow } from '../../hooks/useChannelUsage'
import { useUsageLive } from '../../hooks/useUsageLive'
import { formatCost, formatNumber, formatTokens } from '../../lib/utils'
import { useTranslation } from '../../lib/i18n'
import { BrandIcon } from '../../components/ui/BrandIcon'
import { ChannelTopology } from './components/ChannelTopology'

const PERIODS = ['today', '24h', '7d', '30d'] as const
type Period = (typeof PERIODS)[number]

function getRange(period: Period): { from: string; to: string } {
  const now = new Date()
  const to = now.toISOString()
  if (period === 'today') {
    const start = new Date(now)
    start.setHours(0, 0, 0, 0)
    return { from: start.toISOString(), to }
  }
  const hours = period === '24h' ? 24 : period === '7d' ? 7 * 24 : 30 * 24
  const from = new Date(now.getTime() - hours * 3_600_000)
  return { from: from.toISOString(), to }
}

export default function ChannelsPage() {
  const { data: me } = useMe()
  const { t } = useTranslation()
  const [period, setPeriod] = useState<Period>('today')
  const [expanded, setExpanded] = useState<string | null>(null)
  const isAdmin = me?.is_system_admin ?? false

  const { from, to } = useMemo(() => getRange(period), [period])
  const { data, isLoading } = useChannelUsage(from, to, isAdmin)
  const { live } = useUsageLive()

  if (me && !isAdmin) {
    return <Navigate to="/analytics" replace />
  }

  const totals = data?.totals

  const toggle = (id: string) => setExpanded((prev) => (prev === id ? null : id))

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center gap-3">
        <Select
          value={period}
          onChange={(v) => setPeriod(v as Period)}
          options={PERIODS.map((p) => ({
            value: p,
            label: t(`analytics.period_${p}`),
          }))}
          className="w-48"
          fullWidth={false}
        />
      </div>

      <div className="grid gap-4 grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        <StatCard
          label={t('analytics.tokens_input')}
          value={isLoading ? '...' : formatTokens(totals?.prompt_tokens ?? 0)}
        />
        <StatCard
          label={t('analytics.tokens_output')}
          value={isLoading ? '...' : formatTokens(totals?.completion_tokens ?? 0)}
        />
        <StatCard
          label={t('analytics.tokens_cached')}
          value={isLoading ? '...' : formatTokens(totals?.cached_tokens ?? 0)}
        />
        <StatCard
          label={t('analytics.requests')}
          value={isLoading ? '...' : formatNumber(totals?.total_requests ?? 0)}
        />
        <StatCard
          label={t('analytics.revenue')}
          value={isLoading ? '...' : formatCost(totals?.revenue ?? 0)}
        />
      </div>

      <div>
        <h3 className="text-sm font-semibold text-text-primary mb-3">
          {t('analytics.channel_topology')}
        </h3>
        <ChannelTopology channels={data?.channels ?? []} live={live} />
      </div>

      <div className="bg-bg-secondary rounded-xl border border-border overflow-hidden">
        <div className="px-4 py-3 border-b border-border">
          <h3 className="text-sm font-semibold text-text-primary">
            {t('analytics.channel_breakdown')}
          </h3>
        </div>
        {isLoading ? (
          <p className="p-6 text-sm text-text-tertiary">{t('analytics.loading')}</p>
        ) : (data?.channels.length ?? 0) === 0 ? (
          <p className="p-6 text-sm text-text-tertiary">{t('analytics.no_channels')}</p>
        ) : (
          <div className="divide-y divide-border">
            {data!.channels.map((ch) => (
              <ChannelRow
                key={ch.channel_id}
                channel={ch}
                open={expanded === ch.channel_id}
                onToggle={() => toggle(ch.channel_id)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function ChannelRow({
  channel,
  open,
  onToggle,
}: {
  channel: ChannelUsageRow
  open: boolean
  onToggle: () => void
}) {
  const { t } = useTranslation()

  return (
    <div>
      <button
        type="button"
        onClick={onToggle}
        className="w-full flex flex-wrap items-center gap-3 px-4 py-3 text-left hover:bg-bg-tertiary/40 transition-colors"
      >
        <span className="text-text-tertiary w-4">{open ? '▾' : '▸'}</span>
        <div className="flex items-center gap-2 flex-1 min-w-[200px]">
          <BrandIcon
            logo={channel.provider_logo}
            slug={channel.provider_slug}
            protocol={channel.provider_protocol}
            size={22}
          />
          <div>
            <div className="text-sm font-medium text-text-primary">{channel.channel_label}</div>
            <div className="text-xs text-text-tertiary">
              {channel.models.length} {t('analytics.models')}
            </div>
          </div>
        </div>
        <MetricPill label={t('analytics.requests')} value={formatNumber(channel.total_requests)} />
        <MetricPill
          label={t('analytics.tokens_input')}
          value={formatTokens(channel.prompt_tokens)}
        />
        <MetricPill
          label={t('analytics.tokens_output')}
          value={formatTokens(channel.completion_tokens)}
        />
        <MetricPill
          label={t('analytics.tokens_cached')}
          value={formatTokens(channel.cached_tokens)}
        />
        <MetricPill label={t('analytics.revenue')} value={formatCost(channel.revenue)} highlight />
      </button>
      {open && (
        <div className="bg-bg-tertiary/20 px-4 pb-3">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-text-tertiary text-xs">
                <th className="text-left py-2 pl-8 font-medium">{t('analytics.model')}</th>
                <th className="text-right py-2 font-medium">{t('analytics.requests')}</th>
                <th className="text-right py-2 font-medium">{t('analytics.tokens_input')}</th>
                <th className="text-right py-2 font-medium">{t('analytics.tokens_output')}</th>
                <th className="text-right py-2 font-medium">{t('analytics.tokens_cached')}</th>
                <th className="text-right py-2 font-medium">{t('analytics.revenue')}</th>
              </tr>
            </thead>
            <tbody>
              {channel.models.map((m) => (
                <tr key={m.model_name} className="border-t border-border/50">
                  <td className="py-2 pl-8 text-text-primary font-medium">{m.model_name}</td>
                  <td className="py-2 text-right tabular-nums">{formatNumber(m.total_requests)}</td>
                  <td className="py-2 text-right tabular-nums">{formatTokens(m.prompt_tokens)}</td>
                  <td className="py-2 text-right tabular-nums">
                    {formatTokens(m.completion_tokens)}
                  </td>
                  <td className="py-2 text-right tabular-nums">{formatTokens(m.cached_tokens)}</td>
                  <td className="py-2 text-right tabular-nums text-accent">
                    {formatCost(m.revenue)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function MetricPill({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="text-right min-w-[88px] hidden sm:block">
      <div className="text-[10px] uppercase tracking-wide text-text-tertiary">{label}</div>
      <div className={`text-sm tabular-nums ${highlight ? 'text-accent font-medium' : 'text-text-primary'}`}>
        {value}
      </div>
    </div>
  )
}