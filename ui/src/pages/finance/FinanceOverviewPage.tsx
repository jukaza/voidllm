import { useMemo, useState } from 'react'
import { StatCard } from '../../components/ui/StatCard'
import { MultiAreaChart } from '../../components/ui/charts'
import { PillGroup } from '../../components/ui/PillGroup'
import { financeRangeISO, useFinanceSummary, type FinanceRangeDays } from '../../hooks/useFinance'
import { formatCost } from '../../lib/utils'
import { useTranslation } from '../../lib/i18n'

const PERIODS: FinanceRangeDays[] = [7, 30, 90]

function formatDayLabel(day: string): string {
  const parts = day.split('-')
  if (parts.length === 3) return `${parts[2]}/${parts[1]}`
  return day
}

export default function FinanceOverviewPage() {
  const { t } = useTranslation()
  const [days, setDays] = useState<FinanceRangeDays>(7)
  const { from, to } = useMemo(() => financeRangeISO(days), [days])
  const { data, isLoading } = useFinanceSummary(from, to)

  const chartData = useMemo(
    () =>
      (data?.daily ?? []).map((d) => ({
        label: formatDayLabel(d.day),
        inflow: d.topup_inflow,
        outflow: d.usage_outflow,
      })),
    [data],
  )

  const totals = data?.totals

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <PillGroup
          label={t('finance.period')}
          value={days}
          onChange={setDays}
          options={PERIODS.map((d) => ({ value: d, label: `${d}d` }))}
        />
        {data?.timezone && (
          <span className="text-xs text-text-tertiary">{t('finance.timezone_note', { tz: data.timezone })}</span>
        )}
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <StatCard
          label={t('finance.wallet_liability')}
          value={isLoading ? '...' : formatCost(totals?.wallet_liability ?? 0)}
        />
        <StatCard
          label={t('finance.topup_inflow')}
          value={isLoading ? '...' : formatCost(totals?.topup_inflow ?? 0)}
        />
        <StatCard
          label={t('finance.usage_outflow')}
          value={isLoading ? '...' : formatCost(totals?.usage_outflow ?? 0)}
        />
        <StatCard
          label={t('finance.bonus_granted')}
          value={isLoading ? '...' : formatCost(totals?.topup_bonus ?? 0)}
        />
        <StatCard
          label={t('finance.net_flow')}
          value={
            isLoading
              ? '...'
              : formatCost(
                  (totals?.topup_inflow ?? 0) -
                    (totals?.usage_outflow ?? 0) +
                    (totals?.adjustment_net ?? 0) +
                    (totals?.refund_total ?? 0),
                )
          }
        />
        <StatCard
          label={t('finance.pending_orders')}
          value={isLoading ? '...' : String(totals?.pending_topup_count ?? 0)}
        />
      </div>

      <div className="rounded-lg border border-border bg-bg-secondary p-4">
        <h3 className="text-sm font-semibold text-text-primary mb-4">{t('finance.chart_title')}</h3>
        {chartData.length === 0 && !isLoading ? (
          <p className="text-sm text-text-tertiary py-8 text-center">{t('finance.no_data')}</p>
        ) : (
          <MultiAreaChart
            data={chartData}
            formatValue={formatCost}
            series={[
              { key: 'inflow', color: '#10b981', label: t('finance.topup_inflow') },
              { key: 'outflow', color: '#f59e0b', label: t('finance.usage_outflow') },
            ]}
          />
        )}
      </div>
    </div>
  )
}