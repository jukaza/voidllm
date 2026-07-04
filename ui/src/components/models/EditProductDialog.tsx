import { useState } from 'react'
import { Dialog } from '../ui/Dialog'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { Select } from '../ui/Select'
import { Toggle } from '../ui/Toggle'
import { IconPicker } from '../ui/IconPicker'
import { ComboRouteEditor } from './ComboRouteEditor'
import { resolveModelIcon } from '../../lib/provider-icons'
import type { BillingMode, ModelResponse, UpdateModelParams } from '../../hooks/useModels'
import { billingModeFromModel, useUpdateModel } from '../../hooks/useModels'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'

const MODEL_TYPE_OPTIONS = [
  { value: 'chat', label: 'Chat' },
  { value: 'embedding', label: 'Embedding' },
  { value: 'reranking', label: 'Reranking' },
  { value: 'completion', label: 'Completion' },
  { value: 'image', label: 'Image Generation' },
  { value: 'audio_transcription', label: 'Audio Transcription' },
  { value: 'tts', label: 'Text to Speech' },
]

interface EditProductDialogProps {
  model: ModelResponse
  onClose: () => void
}

export function EditProductDialog({ model, onClose }: EditProductDialogProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const updateModel = useUpdateModel()

  const [name, setName] = useState(model.name)
  const [type, setType] = useState(model.type || 'chat')
  const [aliases, setAliases] = useState((model.aliases ?? []).join(', '))
  const [billingMode, setBillingMode] = useState<BillingMode>(billingModeFromModel(model))
  const [billMinPerRequest, setBillMinPerRequest] = useState(model.bill_min_per_request === true)
  const [sellInputPer1m, setSellInputPer1m] = useState(
    model.sell_input_per_1m != null ? String(model.sell_input_per_1m) : '',
  )
  const [sellOutputPer1m, setSellOutputPer1m] = useState(
    model.sell_output_per_1m != null ? String(model.sell_output_per_1m) : '',
  )
  const [sellCachedInputPer1m, setSellCachedInputPer1m] = useState(
    model.sell_cached_input_per_1m != null ? String(model.sell_cached_input_per_1m) : '',
  )
  const [sellCacheWritePer1m, setSellCacheWritePer1m] = useState(
    model.sell_cache_write_per_1m != null ? String(model.sell_cache_write_per_1m) : '',
  )
  const [sellPerRequest, setSellPerRequest] = useState(
    model.sell_per_request != null ? String(model.sell_per_request) : '',
  )
  const [sellMinPerRequest, setSellMinPerRequest] = useState(
    model.sell_min_per_request != null ? String(model.sell_min_per_request) : '',
  )
  const [logo, setLogo] = useState(model.logo ?? '')
  const [isPublic, setIsPublic] = useState(model.is_public === true)
  const [timeout, setTimeout] = useState(model.timeout ?? '')
  const [rpmLimit, setRpmLimit] = useState(
    model.rpm_limit != null && model.rpm_limit > 0 ? String(model.rpm_limit) : '',
  )

  function handleSubmit(e: React.FormEvent | React.MouseEvent) {
    e.preventDefault()

    const params: UpdateModelParams = {}

    if (name.trim() !== model.name) params.name = name.trim()
    if (type !== (model.type || 'chat')) params.type = type

    const newAliases = aliases.split(',').map((a) => a.trim()).filter(Boolean)
    const sortedNew = [...newAliases].sort()
    const sortedOld = [...(model.aliases ?? [])].sort()
    if (JSON.stringify(sortedNew) !== JSON.stringify(sortedOld)) {
      params.aliases = newAliases
    }

    const prevMode = billingModeFromModel(model)
    if (billingMode !== prevMode) {
      params.bill_per_token = billingMode === 'token'
      params.bill_per_request = billingMode === 'request'
    }
    if (billingMode === 'token') {
      if (billMinPerRequest !== (model.bill_min_per_request === true)) {
        params.bill_min_per_request = billMinPerRequest
      }
      if (sellMinPerRequest.trim()) {
        const parsed = parseFloat(sellMinPerRequest)
        if (!isNaN(parsed) && parsed !== model.sell_min_per_request) {
          params.sell_min_per_request = parsed
        }
      } else if (model.sell_min_per_request != null) {
        params.sell_min_per_request = 0
      }
    } else if (model.bill_min_per_request) {
      params.bill_min_per_request = false
    }
    if (isPublic !== (model.is_public === true)) params.is_public = isPublic

    if (sellInputPer1m.trim()) {
      const parsed = parseFloat(sellInputPer1m)
      if (!isNaN(parsed) && parsed !== model.sell_input_per_1m) params.sell_input_per_1m = parsed
    }
    if (sellOutputPer1m.trim()) {
      const parsed = parseFloat(sellOutputPer1m)
      if (!isNaN(parsed) && parsed !== model.sell_output_per_1m) params.sell_output_per_1m = parsed
    }
    if (sellCachedInputPer1m.trim()) {
      const parsed = parseFloat(sellCachedInputPer1m)
      if (!isNaN(parsed) && parsed !== model.sell_cached_input_per_1m) params.sell_cached_input_per_1m = parsed
    }
    if (sellCacheWritePer1m.trim()) {
      const parsed = parseFloat(sellCacheWritePer1m)
      if (!isNaN(parsed) && parsed !== model.sell_cache_write_per_1m) params.sell_cache_write_per_1m = parsed
    }
    if (sellPerRequest.trim()) {
      const parsed = parseFloat(sellPerRequest)
      if (!isNaN(parsed) && parsed !== model.sell_per_request) params.sell_per_request = parsed
    }

    const trimmedTimeout = timeout.trim()
    if (trimmedTimeout !== (model.timeout ?? '')) {
      params.timeout = trimmedTimeout || undefined
    }

    const trimmedLogo = logo.trim()
    if (trimmedLogo !== (model.logo ?? '')) {
      params.logo = trimmedLogo
    }

    const rpmParsed = rpmLimit.trim() === '' ? 0 : Number.parseInt(rpmLimit, 10)
    if (rpmLimit.trim() !== '' && (Number.isNaN(rpmParsed) || rpmParsed < 0)) {
      toast({ variant: 'error', message: t('common.invalid_number') })
      return
    }
    const prevRpm = model.rpm_limit ?? 0
    if (rpmParsed !== prevRpm) {
      params.rpm_limit = rpmParsed
    }

    if (Object.keys(params).length === 0) {
      onClose()
      return
    }

    updateModel.mutate(
      { modelId: model.id, params },
      {
        onSuccess: () => {
          toast({ variant: 'success', message: t('models.product_updated') })
          onClose()
        },
        onError: (err) => {
          toast({
            variant: 'error',
            message: err instanceof Error ? err.message : t('models.product_update_failed'),
          })
        },
      },
    )
  }

  const isPending = updateModel.isPending
  const isYaml = model.source === 'yaml'

  return (
    <Dialog open onClose={onClose} title={t('models.edit_product')}>
      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        {isYaml ? (
          <p className="text-xs text-amber-600 dark:text-amber-400 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2">
            {t('models.yaml_readonly_hint')}
          </p>
        ) : null}

        <Input
          label={t('models.product_name')}
          value={name}
          onChange={(e) => setName(e.target.value)}
          disabled={isPending || isYaml}
        />
        <Select
          label={t('models.col_type')}
          options={MODEL_TYPE_OPTIONS}
          value={type}
          onChange={setType}
          disabled={isPending || isYaml}
        />
        <Input
          label={t('models.col_aliases')}
          value={aliases}
          onChange={(e) => setAliases(e.target.value)}
          description={t('models.aliases_hint')}
          disabled={isPending || isYaml}
        />

        {!isYaml && (
          <IconPicker
            value={logo}
            onChange={setLogo}
            label={t('models.logo')}
            previewKey={resolveModelIcon(null, name)}
          />
        )}

        <div className="rounded-lg border border-white/5 p-4 space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-text-secondary">{t('models.sell_pricing')}</p>
              <p className="text-xs text-text-tertiary">{t('models.sell_pricing_desc')}</p>
            </div>
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
              disabled={isPending || isYaml}
            />
          </div>
          {billingMode === 'token' ? (
            <>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
                <Input
                  label={t('models.sell_input')}
                  type="number"
                  value={sellInputPer1m}
                  onChange={(e) => setSellInputPer1m(e.target.value)}
                  disabled={isPending || isYaml}
                />
                <Input
                  label={t('models.sell_output')}
                  type="number"
                  value={sellOutputPer1m}
                  onChange={(e) => setSellOutputPer1m(e.target.value)}
                  disabled={isPending || isYaml}
                />
                <Input
                  label={t('models.sell_cached')}
                  type="number"
                  value={sellCachedInputPer1m}
                  onChange={(e) => setSellCachedInputPer1m(e.target.value)}
                  disabled={isPending || isYaml}
                />
                <Input
                  label={t('models.sell_cache_write')}
                  type="number"
                  value={sellCacheWritePer1m}
                  onChange={(e) => setSellCacheWritePer1m(e.target.value)}
                  disabled={isPending || isYaml}
                />
              </div>
              <div className="rounded-lg border border-border/50 p-3 space-y-2">
                <Toggle
                  checked={billMinPerRequest}
                  onChange={setBillMinPerRequest}
                  label={t('models.bill_min_per_request')}
                  disabled={isPending || isYaml}
                  size="sm"
                />
                <p className="text-[10px] text-text-tertiary">{t('models.bill_min_per_request_desc')}</p>
                {billMinPerRequest && (
                  <Input
                    label={t('models.sell_min_per_request')}
                    type="number"
                    value={sellMinPerRequest}
                    onChange={(e) => setSellMinPerRequest(e.target.value)}
                    disabled={isPending || isYaml}
                  />
                )}
              </div>
            </>
          ) : (
            <Input
              label={t('models.sell_per_request')}
              type="number"
              value={sellPerRequest}
              onChange={(e) => setSellPerRequest(e.target.value)}
              disabled={isPending || isYaml}
            />
          )}
        </div>

        <Toggle
          checked={isPublic}
          onChange={setIsPublic}
          label={t('models.public_storefront')}
          disabled={isPending || isYaml}
        />

        <Input
          label="Timeout"
          value={timeout}
          onChange={(e) => setTimeout(e.target.value)}
          placeholder="e.g. 30s, 2m"
          description={t('models.timeout_hint')}
          disabled={isPending || isYaml}
        />

        <Input
          label={t('models.rpm_limit')}
          type="number"
          min={0}
          value={rpmLimit}
          onChange={(e) => setRpmLimit(e.target.value)}
          placeholder={t('providers.rpm_unlimited')}
          description={t('models.rpm_limit_desc')}
          disabled={isPending || isYaml}
        />

        {!isYaml && (
          <ComboRouteEditor
            modelId={model.id}
            routingStrategy={model.routing_strategy}
            stickyRoundRobinLimit={model.sticky_round_robin_limit}
          />
        )}

        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={onClose} disabled={isPending}>
            {t('common.cancel')}
          </Button>
          {!isYaml && (
            <Button onClick={handleSubmit} loading={isPending}>
              {t('models.save_changes')}
            </Button>
          )}
        </div>
      </form>
    </Dialog>
  )
}