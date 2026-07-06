import { useMemo, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { useMe } from '../../hooks/useMe'
import { StatCard } from '../../components/ui/StatCard'
import { Table } from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import { AreaChart } from '../../components/ui/charts'
import { Select } from '../../components/ui/Select'
import { Button } from '../../components/ui/Button'
import { useMyUsage, type UsageDataPoint } from '../../hooks/useUsage'
import { formatCost, formatNumber } from '../../lib/utils'
import { exportData } from '../../lib/export'
import { useTranslation } from '../../lib/i18n'

const TIME_RANGES = ['7d', '30d', '90d'] as const
type TimeRange = (typeof TIME_RANGES)[number]

const RANGE_DAYS: Record<TimeRange, number> = {
  '7d': 7,
  '30d': 30,
  '90d': 90,
}

function getTimeRange(range: TimeRange): { from: string; to: string } {
  const now = new Date()
  const from = new Date(now.getTime() - RANGE_DAYS[range] * 86_400_000)
  return { from: from.toISOString(), to: now.toISOString() }
}

export default function CostPage() {
  const { data: me } = useMe()
  const { t } = useTranslation()
  const isAdmin = me?.is_system_admin ?? false
  const [range, setRange] = useState<TimeRange>('30d')
  const [groupBy, setGroupBy] = useState('model')
  const { from, to } = useMemo(() => getTimeRange(range), [range])

  const { data, isLoading } = useMyUsage(from, to, groupBy)

  const groupOptions = useMemo(
    () => [
      { value: 'model', label: t('analytics.model') },
      { value: 'day', label: t('analytics.group_day') },
      { value: 'key', label: t('analytics.filter_key') },
    ],
    [t],
  )

  const totals = useMemo(() => {
    const rows = data?.data ?? []
    return {
      total_requests: rows.reduce((s, r) => s + r.total_requests, 0),
      cost: rows.reduce((s, r) => s + r.revenue, 0),
    }
  }, [data?.data])

  const chartData = useMemo(() => {
    if (groupBy !== 'day') return []
    return (data?.data ?? []).map((d) => ({
      label: d.group_key?.slice(5, 10) ?? '',
      value: d.revenue,
    }))
  }, [data, groupBy])

  const exportHeaders = useMemo(
    () => [
      { key: 'group_key', label: 'Group' },
      { key: 'total_requests', label: t('analytics.requests') },
      { key: 'revenue', label: t('analytics.cost') },
    ],
    [t],
  )

  if (me && isAdmin) {
    return <Navigate to="/analytics/profit" replace />
  }

  const columns: Column<UsageDataPoint>[] = [
    {
      key: 'group_key',
      header: groupOptions.find((o) => o.value === groupBy)?.label ?? 'Group',
      render: (row) => row.group_label || row.group_key || '—',
    },
    {
      key: 'total_requests',
      header: t('analytics.requests'),
      align: 'right',
      render: (row) => formatNumber(row.total_requests),
    },
    {
      key: 'revenue',
      header: t('analytics.cost'),
      align: 'right',
      render: (row) => formatCost(row.revenue),
    },
  ]

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-end gap-3">
        <Select
          label={t('analytics.period')}
          value={range}
          onChange={(v) => setRange(v as TimeRange)}
          options={TIME_RANGES.map((r) => ({ value: r, label: r }))}
        />
        <Select
          label={t('analytics.group_by')}
          value={groupBy}
          onChange={setGroupBy}
          options={groupOptions}
        />
        <Button
          variant="secondary"
          onClick={() =>
            data &&
            exportData(
              data.data as unknown as Record<string, unknown>[],
              exportHeaders,
              `tavo-cost-${groupBy}-${range}`,
              'csv',
            )
          }
          disabled={!data?.data?.length}
        >
          Export CSV
        </Button>
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-3 gap-4">
        <StatCard label={t('analytics.requests')} value={formatNumber(totals.total_requests)} />
        <StatCard label={t('analytics.cost')} value={formatCost(totals.cost)} />
      </div>

      {groupBy === 'day' && (
        <div className="bg-bg-secondary rounded-xl border border-border p-6">
          <h3 className="text-sm font-semibold text-text-primary mb-4">
            {t('analytics.cost_trend')}
          </h3>
          <AreaChart data={chartData} height={220} formatValue={formatCost} />
        </div>
      )}

      <Table
        columns={columns}
        data={data?.data ?? []}
        keyExtractor={(row) => row.group_key || String(row.total_requests)}
        loading={isLoading}
        emptyMessage={t('analytics.no_data')}
      />
    </div>
  )
}