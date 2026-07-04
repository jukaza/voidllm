import { useEffect, useState } from 'react'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { Select } from '../ui/Select'
import type { SelectOption } from '../ui/Select'
import { useModelRoutes, useReplaceModelRoutes } from '../../hooks/useModelRoutes'
import type { ModelRouteStepInput } from '../../hooks/useModelRoutes'
import { useUpdateModel } from '../../hooks/useModels'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { ProductRouteList } from './ProductRouteList'
import { UpstreamModelSelectModal } from './UpstreamModelSelectModal'
import { stepToKey } from './upstream-model-utils'

export const ROUTING_STRATEGY_OPTIONS: SelectOption[] = [
  { value: 'fallback', label: 'Fallback', description: 'Try steps in order until one succeeds.' },
  { value: 'round-robin', label: 'Round robin', description: 'Rotate across enabled steps.' },
]

export interface RouteStepDraft {
  provider_id: string
  upstream_model: string
  is_enabled: boolean
  provider_name?: string
}

function IconLayers() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" className="text-text-tertiary mx-auto mb-1" aria-hidden="true">
      <path d="M12 2 2 7l10 5 10-5-10-5z" />
      <path d="M2 17l10 5 10-5" />
      <path d="M2 12l10 5 10-5" />
    </svg>
  )
}

export interface ComboRouteStepsEditorProps {
  steps: RouteStepDraft[]
  onStepsChange: (steps: RouteStepDraft[]) => void
  strategy: string
  onStrategyChange: (strategy: string) => void
  stickyLimit: string
  onStickyLimitChange: (value: string) => void
  disabled?: boolean
  showHeader?: boolean
}

export function ComboRouteStepsEditor({
  steps,
  onStepsChange,
  strategy,
  onStrategyChange,
  stickyLimit,
  onStickyLimitChange,
  disabled = false,
  showHeader = true,
}: ComboRouteStepsEditorProps) {
  const { t } = useTranslation()
  const [showPicker, setShowPicker] = useState(false)

  function moveStep(index: number, direction: -1 | 1) {
    const nextIndex = index + direction
    if (nextIndex < 0 || nextIndex >= steps.length) return
    const copy = [...steps]
    ;[copy[index], copy[nextIndex]] = [copy[nextIndex], copy[index]]
    onStepsChange(copy)
  }

  function removeStepKey(key: string) {
    onStepsChange(steps.filter((s) => stepToKey(s) !== key))
  }

  return (
    <>
      <div className="rounded-lg border border-white/5 p-4 space-y-3">
        {showHeader ? (
          <div>
            <p className="text-sm font-medium text-text-secondary">{t('models.col_routes')}</p>
            <p className="text-xs text-text-tertiary mt-0.5">{t('models.routes_edit_hint')}</p>
          </div>
        ) : null}

        <Select
          label={t('models.col_strategy')}
          options={ROUTING_STRATEGY_OPTIONS}
          value={strategy}
          onChange={onStrategyChange}
          disabled={disabled}
        />

        {strategy === 'round-robin' && (
          <Input
            label="Sticky round robin limit"
            type="number"
            value={stickyLimit}
            onChange={(e) => onStickyLimitChange(e.target.value)}
            placeholder="0"
            description="Consecutive requests pinned to the same step before rotating. 0 = rotate every request."
            disabled={disabled}
          />
        )}

        {steps.length === 0 ? (
          <div className="text-center py-4 border border-dashed border-border rounded-lg">
            <IconLayers />
            <p className="text-xs text-text-tertiary">{t('models.routes_empty')}</p>
          </div>
        ) : (
          <ProductRouteList
            steps={steps}
            onMoveUp={(i) => moveStep(i, -1)}
            onMoveDown={(i) => moveStep(i, 1)}
            onRemove={(i) => onStepsChange(steps.filter((_, idx) => idx !== i))}
            disabled={disabled}
          />
        )}

        <button
          type="button"
          onClick={() => setShowPicker(true)}
          disabled={disabled}
          className="w-full py-2 border border-dashed border-border rounded-lg text-xs text-accent font-medium hover:border-accent/50 hover:bg-accent/5 transition-colors flex items-center justify-center gap-1.5 disabled:opacity-50"
        >
          <span className="text-base leading-none">+</span>
          {t('models.add_route_step')}
        </button>
      </div>

      <UpstreamModelSelectModal
        open={showPicker}
        onClose={() => setShowPicker(false)}
        addedSteps={steps}
        onAdd={(step) => {
          const key = stepToKey(step)
          if (steps.some((s) => stepToKey(s) === key)) return
          onStepsChange([...steps, step])
        }}
        onRemove={removeStepKey}
      />
    </>
  )
}

interface ComboRouteEditorProps {
  modelId: string
  routingStrategy?: string
  stickyRoundRobinLimit?: number
}

export function ComboRouteEditor({
  modelId,
  routingStrategy = 'fallback',
  stickyRoundRobinLimit = 0,
}: ComboRouteEditorProps) {
  const { toast } = useToast()
  const { data: routesData, isLoading } = useModelRoutes(modelId)
  const replaceRoutes = useReplaceModelRoutes()
  const updateModel = useUpdateModel()

  const [steps, setSteps] = useState<RouteStepDraft[]>([])
  const [strategy, setStrategy] = useState(routingStrategy || 'fallback')
  const [stickyLimit, setStickyLimit] = useState(
    stickyRoundRobinLimit > 0 ? String(stickyRoundRobinLimit) : '',
  )
  const [initialized, setInitialized] = useState(false)

  useEffect(() => {
    if (!routesData || initialized) return
    setSteps(
      routesData.data.map((s) => ({
        provider_id: s.provider_id,
        upstream_model: s.upstream_model,
        is_enabled: s.is_enabled,
        provider_name: s.provider_name,
      })),
    )
    setInitialized(true)
  }, [routesData, initialized])

  useEffect(() => {
    setStrategy(routingStrategy || 'fallback')
    setStickyLimit(stickyRoundRobinLimit > 0 ? String(stickyRoundRobinLimit) : '')
  }, [routingStrategy, stickyRoundRobinLimit])

  const isPending = replaceRoutes.isPending || updateModel.isPending

  async function handleSave() {
    const routeSteps: ModelRouteStepInput[] = steps.map((s) => ({
      provider_id: s.provider_id,
      upstream_model: s.upstream_model,
      is_enabled: s.is_enabled,
    }))

    const parsedSticky = stickyLimit.trim() ? parseInt(stickyLimit, 10) : 0
    const stickyValue = !isNaN(parsedSticky) && parsedSticky > 0 ? parsedSticky : 0

    const modelParams: {
      routing_strategy?: string
      sticky_round_robin_limit?: number
    } = {}
    if (strategy !== (routingStrategy || 'fallback')) {
      modelParams.routing_strategy = strategy
    }
    const currentSticky = stickyRoundRobinLimit > 0 ? stickyRoundRobinLimit : 0
    if (stickyValue !== currentSticky) {
      modelParams.sticky_round_robin_limit = stickyValue
    }

    try {
      await replaceRoutes.mutateAsync({ modelId, steps: routeSteps })
      if (Object.keys(modelParams).length > 0) {
        await updateModel.mutateAsync({ modelId, params: modelParams })
      }
      toast({ variant: 'success', message: 'Combo routes saved' })
    } catch (err) {
      toast({
        variant: 'error',
        message: err instanceof Error ? err.message : 'Failed to save combo routes',
      })
    }
  }

  if (isLoading && !initialized) {
    return <p className="text-sm text-text-tertiary">Loading routes…</p>
  }

  return (
    <div className="space-y-3">
      <ComboRouteStepsEditor
        steps={steps}
        onStepsChange={setSteps}
        strategy={strategy}
        onStrategyChange={setStrategy}
        stickyLimit={stickyLimit}
        onStickyLimitChange={setStickyLimit}
        disabled={isPending}
      />
      <div className="flex justify-end">
        <Button size="sm" onClick={() => void handleSave()} loading={isPending}>
          Save routes
        </Button>
      </div>
    </div>
  )
}