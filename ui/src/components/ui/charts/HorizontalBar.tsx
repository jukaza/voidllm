import type { TokenBarSegment } from '../../../lib/tokenColors'

export interface HorizontalBarItem {
  label: string
  value: number
  detail?: string
  /** When set, renders a stacked bar colored by token type. */
  segments?: TokenBarSegment[]
}

export interface HorizontalBarProps {
  items: HorizontalBarItem[]
  maxValue?: number
  color?: string
}

export function HorizontalBar({ items, maxValue, color }: HorizontalBarProps) {
  const max = maxValue ?? Math.max(...items.map((i) => i.value), 1)

  return (
    <div className="space-y-5">
      {items.map((item, idx) => {
        const pct = max > 0 ? (item.value / max) * 100 : 0
        const hasSegments = (item.segments?.length ?? 0) > 0
        const opacity = hasSegments ? 1 : Math.max(1 - idx * 0.2, 0.2)

        const barStyle: React.CSSProperties = color
          ? { width: `${pct}%`, background: color, opacity }
          : {
              width: `${pct}%`,
              background: 'linear-gradient(90deg, #6366f1, #8b5cf6)',
              opacity,
            }

        return (
          <div key={item.label}>
            <div className="flex items-center justify-between mb-1.5">
              <span className="text-sm text-text-secondary truncate mr-2">{item.label}</span>
              {item.detail != null && (
                <span className="text-xs text-text-tertiary shrink-0 tabular-nums">{item.detail}</span>
              )}
            </div>
            <div className="h-2.5 rounded-full bg-bg-tertiary overflow-hidden">
              {hasSegments ? (
                <div
                  className="flex h-full rounded-full overflow-hidden transition-all duration-500"
                  style={{ width: `${pct}%` }}
                >
                  {item.segments!.map((seg, segIdx) => {
                    const segPct = item.value > 0 ? (seg.value / item.value) * 100 : 0
                    return (
                      <div
                        key={segIdx}
                        className="h-full shrink-0"
                        style={{ width: `${segPct}%`, backgroundColor: seg.color }}
                      />
                    )
                  })}
                </div>
              ) : (
                <div className="h-full rounded-full transition-all duration-500" style={barStyle} />
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}