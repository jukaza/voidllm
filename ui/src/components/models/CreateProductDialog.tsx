import { useMemo, useState } from 'react'
import { Dialog } from '../ui/Dialog'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { Select } from '../ui/Select'
import { Toggle } from '../ui/Toggle'
import { IconPicker } from '../ui/IconPicker'
import { resolveModelIcon } from '../../lib/provider-icons'
import { ROUTING_STRATEGY_OPTIONS } from './ComboRouteEditor'
import { ProductRouteList } from './ProductRouteList'
import { UpstreamModelSelectModal } from './UpstreamModelSelectModal'
import type { RouteStepDraft } from './ComboRouteEditor'
import type { ModelRouteStepInput } from '../../hooks/useModelRoutes'
import { useCreateModel, useUpdateModel, type BillingMode, type CreateModelParams } from '../../hooks/useModels'
import { useReplaceModelRoutes } from '../../hooks/useModelRoutes'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { PRODUCT_NAME_REGEX, stepToKey } from './upstream-model-utils'
import { SellPricingFields } from './SellPricingFields'

interface CreateProductDialogProps {
  open: boolean
  onClose: () => void
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

export function CreateProductDialog({ open, onClose }: CreateProductDialogProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const createModel = useCreateModel()
  const updateModel = useUpdateModel()
  const replaceRoutes = useReplaceModelRoutes()

  const [name, setName] = useState('')
  const [logo, setLogo] = useState('')
  const [steps, setSteps] = useState<RouteStepDraft[]>([])
  const [showPicker, setShowPicker] = useState(false)
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [billingMode, setBillingMode] = useState<BillingMode>('token')
  const [billMinPerRequest, setBillMinPerRequest] = useState(false)
  const [sellInputPer1m, setSellInputPer1m] = useState('')
  const [sellOutputPer1m, setSellOutputPer1m] = useState('')
  const [sellCachedInputPer1m, setSellCachedInputPer1m] = useState('')
  const [sellCacheWritePer1m, setSellCacheWritePer1m] = useState('')
  const [sellPerRequest, setSellPerRequest] = useState('')
  const [sellMinPerRequest, setSellMinPerRequest] = useState('')
  const [isPublic, setIsPublic] = useState(false)
  const [maxContextTokens, setMaxContextTokens] = useState('')
  const [supportsTools, setSupportsTools] = useState(false)
  const [supportsVision, setSupportsVision] = useState(false)
  const [aliases, setAliases] = useState('')
  const [strategy, setStrategy] = useState('fallback')
  const [stickyLimit, setStickyLimit] = useState('')
  const [rpmLimit, setRpmLimit] = useState('')
  const [nameError, setNameError] = useState<string | undefined>()
  const [routesError, setRoutesError] = useState<string | undefined>()

  const isPending = createModel.isPending || updateModel.isPending || replaceRoutes.isPending

  const strategyOptions = useMemo(
    () =>
      ROUTING_STRATEGY_OPTIONS.map((opt) => ({
        ...opt,
        label: opt.value === 'fallback' ? t('models.strategy_fallback') : t('models.strategy_round_robin'),
        description:
          opt.value === 'fallback'
            ? t('models.strategy_fallback_desc')
            : t('models.strategy_round_robin_desc'),
      })),
    [t],
  )

  function reset() {
    setName('')
    setLogo('')
    setSteps([])
    setShowPicker(false)
    setShowAdvanced(false)
    setBillingMode('token')
    setBillMinPerRequest(false)
    setSellInputPer1m('')
    setSellOutputPer1m('')
    setSellCachedInputPer1m('')
    setSellCacheWritePer1m('')
    setSellPerRequest('')
    setSellMinPerRequest('')
    setIsPublic(false)
    setMaxContextTokens('')
    setSupportsTools(false)
    setSupportsVision(false)
    setAliases('')
    setStrategy('fallback')
    setStickyLimit('')
    setRpmLimit('')
    setNameError(undefined)
    setRoutesError(undefined)
  }

  function handleClose() {
    reset()
    onClose()
  }

  function validateName(value: string): boolean {
    const trimmed = value.trim()
    if (!trimmed) {
      setNameError(t('models.product_name_required'))
      return false
    }
    if (!PRODUCT_NAME_REGEX.test(trimmed)) {
      setNameError(t('models.product_name_invalid'))
      return false
    }
    setNameError(undefined)
    return true
  }

  function validate(): boolean {
    const okName = validateName(name)
    const okRoutes = steps.length > 0
    setRoutesError(okRoutes ? undefined : t('models.product_routes_required'))
    return okName && okRoutes
  }

  function moveStep(index: number, direction: -1 | 1) {
    const next = index + direction
    if (next < 0 || next >= steps.length) return
    setSteps((prev) => {
      const copy = [...prev]
      ;[copy[index], copy[next]] = [copy[next], copy[index]]
      return copy
    })
  }

  function removeStepKey(key: string) {
    setSteps((prev) => prev.filter((s) => stepToKey(s) !== key))
  }

  async function handleSubmit() {
    if (!validate()) return

    const hasSellInput = sellInputPer1m.trim() && !isNaN(parseFloat(sellInputPer1m))
    const hasSellOutput = sellOutputPer1m.trim() && !isNaN(parseFloat(sellOutputPer1m))
    let hasSellCached = sellCachedInputPer1m.trim() && !isNaN(parseFloat(sellCachedInputPer1m))
    let hasSellCacheWrite = sellCacheWritePer1m.trim() && !isNaN(parseFloat(sellCacheWritePer1m))
    const hasSellPerReq = sellPerRequest.trim() && !isNaN(parseFloat(sellPerRequest))

    const hasAnySellPrice =
      hasSellInput || hasSellOutput || hasSellCached || hasSellCacheWrite || hasSellPerReq

    const aliasList = aliases.split(',').map((a) => a.trim()).filter(Boolean)

    const params: CreateModelParams = {
      name: name.trim(),
      type: 'chat',
      strategy: 'priority',
      is_public: isPublic,
    }

    if (aliasList.length > 0) {
      params.aliases = aliasList
    }

    if (logo.trim()) {
      params.logo = logo.trim()
    }

    const ctxParsed = maxContextTokens.trim() === '' ? 0 : Number.parseInt(maxContextTokens, 10)
    if (maxContextTokens.trim() !== '' && (Number.isNaN(ctxParsed) || ctxParsed < 0)) {
      toast({ variant: 'error', message: t('common.invalid_number') })
      return
    }
    if (ctxParsed > 0) {
      params.max_context_tokens = ctxParsed
    }
    if (supportsTools) params.supports_tools = true
    if (supportsVision) params.supports_vision = true

    if (hasAnySellPrice) {
      params.bill_per_token = billingMode === 'token'
      params.bill_per_request = billingMode === 'request'
      if (hasSellInput) params.sell_input_per_1m = parseFloat(sellInputPer1m)
      if (hasSellOutput) params.sell_output_per_1m = parseFloat(sellOutputPer1m)
      if (hasSellCached) params.sell_cached_input_per_1m = parseFloat(sellCachedInputPer1m)
      if (hasSellCacheWrite) params.sell_cache_write_per_1m = parseFloat(sellCacheWritePer1m)
      if (hasSellPerReq) params.sell_per_request = parseFloat(sellPerRequest)
      if (billingMode === 'token' && billMinPerRequest && sellMinPerRequest.trim()) {
        params.bill_min_per_request = true
        params.sell_min_per_request = parseFloat(sellMinPerRequest)
      }
    }

    const routeSteps: ModelRouteStepInput[] = steps.map((s) => ({
      provider_id: s.provider_id,
      upstream_model: s.upstream_model,
      is_enabled: true,
    }))

    const parsedSticky = stickyLimit.trim() ? parseInt(stickyLimit, 10) : 0
    const stickyValue = !isNaN(parsedSticky) && parsedSticky > 0 ? parsedSticky : 0
    const rpmParsed = rpmLimit.trim() === '' ? 0 : Number.parseInt(rpmLimit, 10)
    if (rpmLimit.trim() !== '' && (Number.isNaN(rpmParsed) || rpmParsed < 0)) {
      toast({ variant: 'error', message: t('common.invalid_number') })
      return
    }
    if (rpmParsed > 0) {
      params.rpm_limit = rpmParsed
    }

    try {
      const model = await createModel.mutateAsync(params)
      await replaceRoutes.mutateAsync({ modelId: model.id, steps: routeSteps })
      await updateModel.mutateAsync({
        modelId: model.id,
        params: {
          routing_strategy: strategy,
          sticky_round_robin_limit: stickyValue,
        },
      })
      toast({ variant: 'success', message: t('models.product_created') })
      handleClose()
    } catch (err) {
      toast({
        variant: 'error',
        message: err instanceof Error ? err.message : t('models.product_create_failed'),
      })
    }
  }

  return (
    <>
      <Dialog
        open={open}
        onClose={handleClose}
        title={t('models.add_product')}
        panelClassName="max-w-md"
        footer={
          <div className="flex flex-col gap-2 sm:flex-row">
            <Button variant="secondary" onClick={handleClose} disabled={isPending} className="sm:flex-1">
              {t('common.cancel')}
            </Button>
            <Button onClick={() => void handleSubmit()} loading={isPending} className="sm:flex-1">
              {t('models.add_product')}
            </Button>
          </div>
        }
      >
        <div className="flex flex-col gap-3">
          <div>
            <Input
              label={t('models.product_name')}
              value={name}
              onChange={(e) => {
                setName(e.target.value)
                if (e.target.value.trim()) validateName(e.target.value)
                else setNameError(undefined)
              }}
              placeholder="premium-chat"
              error={nameError}
              disabled={isPending}
              className="font-mono"
            />
            <p className="text-[10px] text-text-tertiary mt-1">{t('models.product_name_hint')}</p>
          </div>

          <IconPicker
            value={logo}
            onChange={setLogo}
            label={t('models.logo')}
            previewKey={resolveModelIcon(null, name)}
          />

          <div>
            <label className="text-sm font-medium text-text-secondary mb-1.5 block">
              {t('models.col_routes')}
            </label>
            <p className="text-[10px] text-text-tertiary mb-2">{t('models.routes_order_hint')}</p>
            {steps.length === 0 ? (
              <div className="text-center py-5 border border-dashed border-border rounded-lg bg-white/[0.01]">
                <IconLayers />
                <p className="text-xs text-text-tertiary">{t('models.routes_empty')}</p>
              </div>
            ) : (
              <ProductRouteList
                steps={steps}
                onMoveUp={(i) => moveStep(i, -1)}
                onMoveDown={(i) => moveStep(i, 1)}
                onRemove={(i) => setSteps((prev) => prev.filter((_, idx) => idx !== i))}
                disabled={isPending}
              />
            )}
            {routesError ? <p className="text-xs text-error mt-1">{routesError}</p> : null}
            <button
              type="button"
              onClick={() => setShowPicker(true)}
              disabled={isPending}
              className="w-full mt-2 py-2.5 border border-dashed border-border rounded-lg text-xs text-accent font-medium hover:border-accent/50 hover:bg-accent/5 transition-colors flex items-center justify-center gap-1.5 disabled:opacity-50"
            >
              <span className="text-base leading-none">+</span>
              {t('models.add_route_step')}
            </button>
          </div>

          <Select
            label={t('models.col_strategy')}
            options={strategyOptions}
            value={strategy}
            onChange={setStrategy}
            disabled={isPending}
          />
          {strategy === 'round-robin' && (
            <Input
              label={t('models.sticky_rr_limit')}
              type="number"
              value={stickyLimit}
              onChange={(e) => setStickyLimit(e.target.value)}
              placeholder="1"
              description={t('models.sticky_rr_hint')}
              disabled={isPending}
            />
          )}

          <details
            className="group rounded-lg border border-border/60"
            open={showAdvanced}
            onToggle={(e) => setShowAdvanced((e.target as HTMLDetailsElement).open)}
          >
            <summary className="cursor-pointer list-none select-none px-3 py-2.5 text-xs font-medium text-text-tertiary hover:text-text-secondary">
              {t('models.advanced_pricing')}
            </summary>
            <div className="px-3 pb-3 space-y-3 border-t border-border/40 pt-3">
              <Select
                label={t('models.billing_mode')}
                options={[
                  { value: 'token', label: t('models.bill_per_token') },
                  { value: 'request', label: t('models.bill_per_request') },
                ]}
                value={billingMode}
                onChange={(v) => {
                  setBillingMode(v as BillingMode)
                  if (v === 'request') setBillMinPerRequest(false)
                }}
                disabled={isPending}
              />
              <SellPricingFields
                billingMode={billingMode}
                billMinPerRequest={billMinPerRequest}
                onBillMinPerRequestChange={setBillMinPerRequest}
                sellInputPer1m={sellInputPer1m}
                onSellInputPer1mChange={setSellInputPer1m}
                sellOutputPer1m={sellOutputPer1m}
                onSellOutputPer1mChange={setSellOutputPer1m}
                sellCachedInputPer1m={sellCachedInputPer1m}
                onSellCachedInputPer1mChange={setSellCachedInputPer1m}
                sellCacheWritePer1m={sellCacheWritePer1m}
                onSellCacheWritePer1mChange={setSellCacheWritePer1m}
                sellMinPerRequest={sellMinPerRequest}
                onSellMinPerRequestChange={setSellMinPerRequest}
                sellPerRequest={sellPerRequest}
                onSellPerRequestChange={setSellPerRequest}
                disabled={isPending}
              />
              <Input
                label={t('models.max_context_tokens')}
                type="number"
                min={0}
                value={maxContextTokens}
                onChange={(e) => setMaxContextTokens(e.target.value)}
                placeholder="128000"
                description={t('models.max_context_hint')}
                disabled={isPending}
              />
              <Toggle
                checked={supportsTools}
                onChange={setSupportsTools}
                label={t('models.supports_tools')}
                disabled={isPending}
                size="sm"
              />
              <Toggle
                checked={supportsVision}
                onChange={setSupportsVision}
                label={t('models.supports_vision')}
                disabled={isPending}
                size="sm"
              />
              <Toggle checked={isPublic} onChange={setIsPublic} label={t('models.public_storefront')} disabled={isPending} size="sm" />
              <Input
                label={t('models.col_aliases')}
                value={aliases}
                onChange={(e) => setAliases(e.target.value)}
                placeholder="gpt-4o-mini, smart-chat"
                description={t('models.col_aliases_hint')}
                disabled={isPending}
              />
              <Input
                label={t('models.rpm_limit')}
                type="number"
                min={0}
                value={rpmLimit}
                onChange={(e) => setRpmLimit(e.target.value)}
                placeholder={t('providers.rpm_unlimited')}
                description={t('models.rpm_limit_desc')}
                disabled={isPending}
              />
            </div>
          </details>
        </div>
      </Dialog>

      <UpstreamModelSelectModal
        open={showPicker}
        onClose={() => setShowPicker(false)}
        addedSteps={steps}
        onAdd={(step) => {
          setSteps((prev) => {
            const key = stepToKey(step)
            if (prev.some((s) => stepToKey(s) === key)) return prev
            return [...prev, step]
          })
          setRoutesError(undefined)
        }}
        onRemove={removeStepKey}
      />
    </>
  )
}