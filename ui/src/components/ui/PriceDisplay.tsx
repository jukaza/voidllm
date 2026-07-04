import { formatCost } from '../../lib/utils'
import { cn } from '../../lib/utils'

export function PriceCell({
  value,
  className,
}: {
  value: number | null | undefined
  className?: string
}) {
  if (value == null || value <= 0) {
    return <span className="text-text-tertiary">—</span>
  }
  return (
    <span className={cn('text-text-secondary tabular-nums font-medium', className)}>
      {formatCost(value)}
    </span>
  )
}

export function CostPairCell({
  input,
  output,
}: {
  input: number | null | undefined
  output: number | null | undefined
}) {
  if (input == null || output == null) {
    return <span className="text-text-tertiary text-xs">—</span>
  }

  return (
    <div className="flex flex-col gap-0.5 text-xs tabular-nums">
      <div className="flex items-baseline gap-1.5">
        <span className="text-[10px] uppercase tracking-wide text-text-tertiary w-6 shrink-0">in</span>
        <span className="text-text-secondary">{formatCost(input)}</span>
      </div>
      <div className="flex items-baseline gap-1.5">
        <span className="text-[10px] uppercase tracking-wide text-text-tertiary w-6 shrink-0">out</span>
        <span className="text-text-secondary">{formatCost(output)}</span>
      </div>
    </div>
  )
}

export function PriceColumnHeader({
  label,
  unit,
  align = 'right',
}: {
  label: string
  unit?: string
  align?: 'left' | 'right'
}) {
  return (
    <th
      className={cn(
        'px-4 py-3 font-medium',
        align === 'right' ? 'text-right' : 'text-left',
      )}
    >
      <div>{label}</div>
      {unit != null && (
        <div className="text-[10px] font-normal text-text-tertiary/80 mt-0.5">{unit}</div>
      )}
    </th>
  )
}