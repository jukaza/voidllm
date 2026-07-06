import { useState } from 'react'
import { BrandIcon } from '../ui/BrandIcon'
import { CatalogCapabilityIcon } from './CatalogCapabilityIcon'
import { CatalogTooltip } from './CatalogTooltip'
import type { CatalogModelItem } from '../../hooks/useProviders'
import {
  formatCatalogLatency,
  formatCatalogRequestCount,
  formatCatalogSuccessRate,
  formatCatalogTps,
  formatContextTokens,
  inferCapabilities,
} from '../../lib/catalog-utils'
import { useTranslation } from '../../lib/i18n'
import { cn, formatCost } from '../../lib/utils'

interface ModelCatalogCardProps {
  model: CatalogModelItem
}

function IconTool() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} aria-hidden="true">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M14.7 6.3a1 1 0 000 1.4l1.6 1.6a1 1 0 001.4 0l3.77-3.77a6 6 0 01-7.94 7.94l-6.91 6.91a2.12 2.12 0 01-3-3l6.91-6.91a6 6 0 017.94-7.94l-3.76 3.76z"
      />
    </svg>
  )
}

function IconEye() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} aria-hidden="true">
      <path strokeLinecap="round" strokeLinejoin="round" d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  )
}

function IconCache() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} aria-hidden="true">
      <path strokeLinecap="round" strokeLinejoin="round" d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
    </svg>
  )
}

function PriceBox({
  label,
  value,
  tooltip,
  className,
}: {
  label: string
  value: number | null | undefined
  tooltip?: string
  className?: string
}) {
  const { t } = useTranslation()
  const hasValue = value != null && value > 0
  const box = (
    <div
      className={cn(
        'flex min-w-0 flex-col items-center gap-0.5 rounded-lg border border-border/80 bg-bg-primary/60 px-1.5 py-1.5',
        tooltip && 'cursor-help',
        className,
      )}
    >
      <span className="font-mono text-[8px] uppercase tracking-tight text-text-tertiary">{label}</span>
      <span
        className={cn(
          'truncate font-mono text-[11px] font-bold tabular-nums',
          hasValue ? 'text-text-primary' : 'text-text-tertiary',
        )}
      >
        {hasValue ? formatCost(value) : t('catalog.stat_na')}
      </span>
    </div>
  )
  if (!tooltip) return box
  return <CatalogTooltip content={tooltip}>{box}</CatalogTooltip>
}

function StatCell({
  label,
  value,
  tooltip,
  valueClassName,
}: {
  label: string
  value: string
  tooltip: string
  valueClassName?: string
}) {
  return (
    <CatalogTooltip content={tooltip}>
      <div className="flex min-w-0 cursor-help flex-col items-center gap-0.5">
        <span className="font-mono text-[8px] uppercase tracking-tight text-text-tertiary">{label}</span>
        <span className={cn('truncate font-mono text-[10px] font-bold', valueClassName ?? 'text-text-primary')}>
          {value}
        </span>
      </div>
    </CatalogTooltip>
  )
}

export function ModelCatalogCard({ model }: ModelCatalogCardProps) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)
  const contextLabel = formatContextTokens(model.max_context_tokens)
  const caps = inferCapabilities(model)
  const cachePrice = model.sell_cached_input_per_1m
  const stats = model.stats
  const reqLabel = formatCatalogRequestCount(stats?.request_count) ?? t('catalog.stat_na')
  const okLabel = formatCatalogSuccessRate(stats?.success_rate) ?? t('catalog.stat_na')
  const latencyLabel = formatCatalogLatency(stats?.avg_latency_ms) ?? t('catalog.stat_na')
  const tpsLabel = formatCatalogTps(stats?.avg_tps) ?? t('catalog.stat_na')
  const hasOkRate = stats?.request_count != null && stats.request_count > 0 && stats.success_rate != null

  async function copyName() {
    try {
      await navigator.clipboard.writeText(model.name)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      /* ignore */
    }
  }

  const cacheTooltip =
    caps.cache && cachePrice != null ? (
      <span>
        {t('catalog.tooltip_cache_yes')}
        {' — '}
        <strong className="text-text-primary">{formatCost(cachePrice)}</strong>
        <span className="text-text-tertiary"> /1M</span>
      </span>
    ) : undefined

  return (
    <article className="group flex h-full flex-col rounded-2xl border border-border/80 bg-gradient-to-b from-bg-secondary/90 to-bg-primary/40 p-3.5 transition-all duration-200 hover:border-accent/25 hover:shadow-[0_0_24px_rgba(99,102,241,0.08)]">
      <div className="flex items-start gap-2.5">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-white/5 bg-bg-primary/50">
          <BrandIcon logo={model.logo} modelName={model.name} size={28} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-1">
            <p className="truncate font-mono text-sm font-semibold leading-tight text-text-primary">
              {model.name}
            </p>
            <button
              type="button"
              onClick={() => void copyName()}
              aria-label={t('common.copy')}
              className="shrink-0 rounded-md p-1 text-text-tertiary opacity-0 transition-opacity hover:bg-bg-tertiary hover:text-text-primary group-hover:opacity-100"
            >
              {copied ? (
                <svg className="h-3.5 w-3.5 text-success" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
                  <polyline points="20 6 9 17 4 12" />
                </svg>
              ) : (
                <svg className="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
                  <rect x="9" y="9" width="13" height="13" rx="2" />
                  <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
                </svg>
              )}
            </button>
          </div>
          <div className="mt-1.5 flex flex-wrap items-center gap-1">
            {contextLabel && (
              <span className="rounded-full border border-border bg-bg-primary/50 px-1.5 py-0.5 font-mono text-[9px] text-text-tertiary">
                {contextLabel}
              </span>
            )}
            {model.bill_per_request && (
              <span className="rounded-full border border-accent/20 bg-accent/10 px-1.5 py-0.5 font-mono text-[9px] font-semibold uppercase tracking-wide text-accent">
                {t('catalog.badge_per_req')}
              </span>
            )}
          </div>
        </div>
      </div>

      <div className="mt-2.5 flex items-center gap-1">
        <CatalogCapabilityIcon
          variant="tools"
          active={caps.tools}
          label={t('catalog.cap_tools')}
          tooltip={caps.tools ? t('catalog.tooltip_tools_yes') : undefined}
        >
          <IconTool />
        </CatalogCapabilityIcon>
        <CatalogCapabilityIcon
          variant="vision"
          active={caps.vision}
          label={t('catalog.cap_vision')}
          tooltip={caps.vision ? t('catalog.tooltip_vision_yes') : undefined}
        >
          <IconEye />
        </CatalogCapabilityIcon>
        <CatalogCapabilityIcon
          variant="cache"
          active={caps.cache}
          label={t('catalog.cap_cache')}
          tooltip={cacheTooltip}
        >
          <IconCache />
        </CatalogCapabilityIcon>
      </div>

      {model.bill_per_token && (
        <div className="mt-3 grid grid-cols-3 gap-1 rounded-lg border border-border/60 bg-bg-primary/50 p-1">
          <PriceBox label={t('catalog.price_input_short')} value={model.sell_input_per_1m} />
          <PriceBox label={t('catalog.price_output_short')} value={model.sell_output_per_1m} />
          <PriceBox
            label={t('catalog.price_min_short')}
            value={model.bill_min_per_request ? model.sell_min_per_request : undefined}
            tooltip={t('catalog.tooltip_price_min')}
          />
        </div>
      )}

      {model.bill_per_request && !model.bill_per_token && (
        <div className="mt-3 grid grid-cols-1 gap-1 rounded-lg border border-border/60 bg-bg-primary/50 p-1">
          <PriceBox label={t('catalog.price_per_request')} value={model.sell_per_request} />
        </div>
      )}

      <div className="mt-3 grid grid-cols-4 gap-0.5 rounded-lg border border-border/60 bg-bg-primary/50 px-1 py-1.5">
        <StatCell
          label={t('catalog.stat_req')}
          value={reqLabel}
          tooltip={t('catalog.tooltip_stat_req')}
        />
        <StatCell
          label={t('catalog.stat_ok')}
          value={okLabel}
          tooltip={t('catalog.tooltip_stat_ok')}
          valueClassName={hasOkRate ? 'text-success' : 'text-text-tertiary'}
        />
        <StatCell
          label={t('catalog.stat_latency')}
          value={latencyLabel}
          tooltip={t('catalog.tooltip_stat_latency')}
        />
        <StatCell
          label={t('catalog.stat_tps')}
          value={tpsLabel}
          tooltip={t('catalog.tooltip_stat_tps')}
        />
      </div>
    </article>
  )
}