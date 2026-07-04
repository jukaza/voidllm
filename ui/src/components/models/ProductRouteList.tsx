import type { RouteStepDraft } from './ComboRouteEditor'
import { formatStepLabel } from './upstream-model-utils'
import { cn } from '../../lib/utils'

interface ProductRouteListProps {
  steps: RouteStepDraft[]
  onMoveUp: (index: number) => void
  onMoveDown: (index: number) => void
  onRemove: (index: number) => void
  disabled?: boolean
}

function IconArrowUp() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M12 19V5" />
      <path d="M5 12l7-7 7 7" />
    </svg>
  )
}

function IconArrowDown() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M12 5v14" />
      <path d="M19 12l-7 7-7-7" />
    </svg>
  )
}

function IconClose() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M18 6L6 18" />
      <path d="M6 6l12 12" />
    </svg>
  )
}

export function ProductRouteList({
  steps,
  onMoveUp,
  onMoveDown,
  onRemove,
  disabled = false,
}: ProductRouteListProps) {
  if (steps.length === 0) return null

  return (
    <div className="flex max-h-[280px] min-w-0 flex-col gap-1 overflow-y-auto">
      {steps.map((step, index) => (
        <div
          key={`${step.provider_id}-${step.upstream_model}-${index}`}
          className="group flex min-w-0 items-center gap-1.5 rounded-md bg-white/[0.02] px-2 py-1.5 transition-colors hover:bg-white/[0.04]"
        >
          <span className="text-[10px] font-medium text-text-tertiary w-3 text-center shrink-0 tabular-nums">
            {index + 1}
          </span>
          <div
            className="min-w-0 flex-1 truncate rounded px-1.5 py-0.5 font-mono text-xs text-text-primary"
            title={formatStepLabel(step)}
          >
            {formatStepLabel(step)}
          </div>
          <div className="flex shrink-0 items-center gap-0.5">
            <button
              type="button"
              onClick={() => onMoveUp(index)}
              disabled={disabled || index === 0}
              className={cn(
                'p-0.5 rounded transition-colors',
                index === 0
                  ? 'text-text-tertiary/20 cursor-not-allowed'
                  : 'text-text-tertiary hover:text-accent hover:bg-accent/10',
              )}
              title="Move up"
            >
              <IconArrowUp />
            </button>
            <button
              type="button"
              onClick={() => onMoveDown(index)}
              disabled={disabled || index === steps.length - 1}
              className={cn(
                'p-0.5 rounded transition-colors',
                index === steps.length - 1
                  ? 'text-text-tertiary/20 cursor-not-allowed'
                  : 'text-text-tertiary hover:text-accent hover:bg-accent/10',
              )}
              title="Move down"
            >
              <IconArrowDown />
            </button>
            <button
              type="button"
              onClick={() => onRemove(index)}
              disabled={disabled}
              className="p-0.5 rounded text-text-tertiary hover:text-error hover:bg-error/10 transition-colors"
              title="Remove"
            >
              <IconClose />
            </button>
          </div>
        </div>
      ))}
    </div>
  )
}