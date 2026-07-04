import { useMemo } from 'react'
import type { ChannelUsageRow } from '../../../hooks/useChannelUsage'
import type { LiveSnapshot } from '../../../hooks/useUsageLive'
import { BrandIcon } from '../../../components/ui/BrandIcon'
import { formatCost, formatNumber } from '../../../lib/utils'
import { useTranslation } from '../../../lib/i18n'

interface ChannelTopologyProps {
  channels: ChannelUsageRow[]
  live: LiveSnapshot | null
}

export function ChannelTopology({ channels, live }: ChannelTopologyProps) {
  const { t } = useTranslation()

  const activeProviders = useMemo(() => {
    const set = new Set<string>()
    for (const g of live?.active_requests ?? []) {
      if (g.provider) set.add(g.provider.toLowerCase())
    }
    for (const r of live?.recent_requests ?? []) {
      if (r.provider) set.add(r.provider.toLowerCase())
    }
    return set
  }, [live])

  const layout = useMemo(() => {
    const cx = 400
    const cy = 210
    const count = Math.max(channels.length, 1)
    const rx = Math.max(280, ((180 + 24) * count) / (2 * Math.PI))
    const ry = Math.max(160, rx * 0.55)

    const nodes = channels.map((ch, i) => {
      const angle = (2 * Math.PI * i) / count - Math.PI / 2
      const keys = [ch.provider_slug, ch.provider, ch.channel_label]
        .filter(Boolean)
        .map((k) => k!.toLowerCase())
      const active = keys.some((k) => activeProviders.has(k) || [...activeProviders].some((p) => k.includes(p) || p.includes(k)))
      return {
        ch,
        x: cx + rx * Math.cos(angle),
        y: cy + ry * Math.sin(angle),
        active,
      }
    })

    return { cx, cy, nodes }
  }, [channels, activeProviders])

  if (channels.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-sm text-text-tertiary">
        {t('analytics.no_channels')}
      </div>
    )
  }

  return (
    <div className="relative w-full overflow-hidden rounded-xl border border-border bg-bg-secondary min-h-[320px]">
      <svg
        viewBox="0 0 800 420"
        className="absolute inset-0 w-full h-full pointer-events-none"
        role="presentation"
      >
        {layout.nodes.map(({ ch, x, y, active }) => (
          <line
            key={`edge-${ch.channel_id}`}
            x1={layout.cx}
            y1={layout.cy}
            x2={x}
            y2={y}
            stroke={active ? '#22c55e' : 'var(--color-border, #334155)'}
            strokeWidth={active ? 2.5 : 1.5}
            strokeOpacity={active ? 0.9 : 0.45}
            className={active ? 'animate-pulse' : undefined}
          />
        ))}
      </svg>

      <div
        className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-10 flex items-center justify-center gap-2 rounded-xl border-2 border-accent/40 bg-accent/10 px-5 py-2.5"
      >
        <span className="text-sm font-bold text-text-primary">VoidLLM</span>
        {live && live.active_count > 0 && (
          <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-accent px-1.5 text-[10px] font-bold text-white">
            {live.active_count}
          </span>
        )}
      </div>

      {layout.nodes.map(({ ch, x, y, active }) => {
        const left = `${(x / 800) * 100}%`
        const top = `${(y / 420) * 100}%`
        const label = ch.channel_label.length > 20 ? ch.channel_label.slice(0, 18) + '…' : ch.channel_label
        return (
          <div
            key={ch.channel_id}
            className={`absolute z-10 flex w-44 -translate-x-1/2 -translate-y-1/2 items-center gap-2.5 rounded-xl border bg-bg-secondary px-3 py-2 shadow-sm ${
              active ? 'border-green-500/70 shadow-green-500/10' : 'border-border'
            }`}
            style={{ left, top }}
          >
            <BrandIcon
              logo={ch.provider_logo}
              slug={ch.provider_slug}
              protocol={ch.provider_protocol}
              size={24}
            />
            <div className="min-w-0 flex-1">
              <div className="truncate text-xs font-semibold text-text-primary">{label}</div>
              <div className="text-[10px] text-text-tertiary tabular-nums">
                {formatCost(ch.revenue)} · {formatNumber(ch.total_requests)} req
              </div>
            </div>
            {active && <span className="h-2 w-2 shrink-0 rounded-full bg-green-500 animate-ping" />}
          </div>
        )
      })}
    </div>
  )
}