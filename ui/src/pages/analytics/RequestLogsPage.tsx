import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Table } from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { useUsageLogs, type UsageLogEntry } from '../../hooks/useUsageLogs'
import { useMe } from '../../hooks/useMe'
import { LatencyBadge } from '../../components/ui/LatencyBadge'
import { TokenBreakdown } from '../../components/ui/TokenBreakdown'
import { formatCost, formatDate, truncateRequestId } from '../../lib/utils'
import { UsageLogDrawer } from './components/UsageLogDrawer'
import { useTranslation } from '../../lib/i18n'

const TIME_RANGES = ['24h', '7d', '30d'] as const
type TimeRange = (typeof TIME_RANGES)[number]

const RANGE_HOURS: Record<TimeRange, number> = {
  '24h': 24,
  '7d': 168,
  '30d': 720,
}

function getTimeRange(range: TimeRange): { from: string; to: string } {
  const now = new Date()
  const from = new Date(now.getTime() - RANGE_HOURS[range] * 3_600_000)
  return { from: from.toISOString(), to: now.toISOString() }
}

export default function RequestLogsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { data: me } = useMe()
  const { t } = useTranslation()
  const isAdmin = me?.is_system_admin ?? false

  const [range, setRange] = useState<TimeRange>('7d')
  const [model, setModel] = useState('')
  const [requestId, setRequestId] = useState(searchParams.get('request_id') ?? '')
  const [selectedId, setSelectedId] = useState<string | null>(
    searchParams.get('request_id'),
  )

  const { from, to } = useMemo(() => getTimeRange(range), [range])

  const { data, isLoading, refetch } = useUsageLogs({
    from,
    to,
    model: model || undefined,
    request_id: requestId || undefined,
    limit: 100,
  })

  const columns: Column<UsageLogEntry>[] = useMemo(() => {
    const cols: Column<UsageLogEntry>[] = [
      {
        key: 'created_at',
        header: t('analytics.time'),
        render: (row) => (
          <span className="text-text-secondary text-xs whitespace-nowrap">
            {formatDate(row.created_at)}
          </span>
        ),
      },
      {
        key: 'model_name',
        header: 'Model',
        render: (row) => <span className="text-text-primary">{row.model_name}</span>,
      },
    ]
    cols.push(
      {
        key: 'request_id',
        header: t('analytics.request_id'),
        render: (row) => (
          <span
            className="font-mono text-xs text-text-tertiary"
            title={row.request_id}
          >
            {truncateRequestId(row.request_id)}
          </span>
        ),
      },
      {
        key: 'tokens',
        header: 'Tokens',
        render: (row) => (
          <TokenBreakdown
            promptTokens={row.prompt_tokens}
            completionTokens={row.completion_tokens}
            cachedTokens={row.cached_tokens}
            cacheWriteTokens={row.cache_write_tokens}
            compact
          />
        ),
      },
      {
        key: 'latency',
        header: t('analytics.latency'),
        align: 'right',
        render: (row) => (
          <LatencyBadge
            durationMs={row.duration_ms}
            ttftMs={row.ttft_ms}
            isStream={row.is_stream}
          />
        ),
      },
      {
        key: 'revenue',
        header: isAdmin ? t('analytics.revenue') : t('analytics.payment'),
        align: 'right',
        render: (row) => (
          <span className="text-text-primary tabular-nums">
            {row.revenue != null ? formatCost(row.revenue) : '—'}
          </span>
        ),
      },
      {
        key: 'status_code',
        header: 'Status',
        align: 'right',
        render: (row) => (
          <span
            className={
              row.status_code >= 400
                ? 'text-red-400 font-medium'
                : 'text-emerald-400 font-medium'
            }
          >
            {row.status_code}
          </span>
        ),
      },
      {
        key: 'actions',
        header: '',
        align: 'right',
        render: (row) => (
          <Button
            variant="ghost"
            size="sm"
            onClick={(e) => {
              e.stopPropagation()
              setSelectedId(row.request_id)
              setSearchParams({ request_id: row.request_id })
            }}
          >
            {t('common.view')}
          </Button>
        ),
      },
    )
    return cols
  }, [isAdmin, t, setSearchParams])

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end gap-3">
        <div>
          <label className="text-xs text-text-tertiary block mb-1">{t('analytics.period')}</label>
          <select
            className="bg-bg-secondary border border-border rounded-lg px-3 py-2 text-sm"
            value={range}
            onChange={(e) => setRange(e.target.value as TimeRange)}
          >
            {TIME_RANGES.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </select>
        </div>
        <div className="min-w-[160px]">
          <label className="text-xs text-text-tertiary block mb-1">Model</label>
          <Input value={model} onChange={(e) => setModel(e.target.value)} placeholder="gpt-4o" />
        </div>
        <div className="min-w-[200px]">
          <label className="text-xs text-text-tertiary block mb-1">{t('analytics.request_id')}</label>
          <Input
            value={requestId}
            onChange={(e) => setRequestId(e.target.value)}
            placeholder="req_..."
          />
        </div>
        <Button variant="secondary" onClick={() => refetch()}>
          {t('common.refresh')}
        </Button>
      </div>

      <Table
        columns={columns}
        data={data?.data ?? []}
        keyExtractor={(row) => row.id}
        loading={isLoading}
        emptyMessage={t('analytics.no_logs')}
      />

      <UsageLogDrawer
        requestId={selectedId}
        onClose={() => {
          setSelectedId(null)
          setSearchParams({})
        }}
      />
    </div>
  )
}