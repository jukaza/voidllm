import {
  ResponsiveContainer,
  AreaChart as RechartsAreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
  Legend,
} from 'recharts'
import type { TooltipContentProps } from 'recharts'
import type { ValueType, NameType } from 'recharts/types/component/DefaultTooltipContent'

export interface MultiAreaChartSeries {
  key: string
  color: string
  label: string
}

export interface MultiAreaChartProps {
  data: Record<string, string | number>[]
  xKey?: string
  series: MultiAreaChartSeries[]
  height?: number
  formatValue?: (n: number) => string
}

function renderTooltip(
  props: TooltipContentProps<ValueType, NameType>,
  formatValue?: (n: number) => string,
) {
  const { active, payload, label } = props
  if (!active || !payload || payload.length === 0) return null
  return (
    <div className="rounded-lg border border-border bg-bg-secondary px-3 py-2 shadow-lg">
      <p className="text-[11px] text-text-tertiary mb-1">{label}</p>
      {payload.map((entry) => {
        const raw = entry.value
        const num = typeof raw === 'number' ? raw : 0
        const display = formatValue ? formatValue(num) : num.toLocaleString()
        return (
          <p key={String(entry.dataKey)} className="text-xs text-text-primary tabular-nums">
            <span style={{ color: entry.color }}>{entry.name}: </span>
            {display}
          </p>
        )
      })}
    </div>
  )
}

export function MultiAreaChart({
  data,
  xKey = 'label',
  series,
  height = 300,
  formatValue,
}: MultiAreaChartProps) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <RechartsAreaChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" vertical={false} />
        <XAxis
          dataKey={xKey}
          tick={{ fill: '#8494a8', fontSize: 11 }}
          axisLine={false}
          tickLine={false}
        />
        <YAxis
          tick={{ fill: '#8494a8', fontSize: 11 }}
          axisLine={false}
          tickLine={false}
          width={56}
          tickFormatter={(v) => (formatValue ? formatValue(Number(v)) : String(v))}
        />
        <Tooltip content={(p) => renderTooltip(p, formatValue)} />
        <Legend wrapperStyle={{ fontSize: 11 }} />
        {series.map((s) => (
          <Area
            key={s.key}
            type="monotone"
            dataKey={s.key}
            name={s.label}
            stroke={s.color}
            fill={s.color}
            fillOpacity={0.15}
            strokeWidth={2}
            dot={false}
          />
        ))}
      </RechartsAreaChart>
    </ResponsiveContainer>
  )
}