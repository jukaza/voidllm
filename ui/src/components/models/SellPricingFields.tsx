import { Input } from '../ui/Input'
import { Toggle } from '../ui/Toggle'
import { useTranslation } from '../../lib/i18n'
import type { BillingMode } from '../../hooks/useModels'

const VND_INPUT = {
  type: 'number' as const,
  min: 0,
  step: 1,
  suffix: '₫',
  inputMode: 'numeric' as const,
}

interface SellPricingFieldsProps {
  billingMode: BillingMode
  billMinPerRequest: boolean
  onBillMinPerRequestChange: (value: boolean) => void
  sellInputPer1m: string
  onSellInputPer1mChange: (value: string) => void
  sellOutputPer1m: string
  onSellOutputPer1mChange: (value: string) => void
  sellCachedInputPer1m: string
  onSellCachedInputPer1mChange: (value: string) => void
  sellCacheWritePer1m: string
  onSellCacheWritePer1mChange: (value: string) => void
  sellMinPerRequest: string
  onSellMinPerRequestChange: (value: string) => void
  sellPerRequest: string
  onSellPerRequestChange: (value: string) => void
  disabled?: boolean
}

export function SellPricingFields({
  billingMode,
  billMinPerRequest,
  onBillMinPerRequestChange,
  sellInputPer1m,
  onSellInputPer1mChange,
  sellOutputPer1m,
  onSellOutputPer1mChange,
  sellCachedInputPer1m,
  onSellCachedInputPer1mChange,
  sellCacheWritePer1m,
  onSellCacheWritePer1mChange,
  sellMinPerRequest,
  onSellMinPerRequestChange,
  sellPerRequest,
  onSellPerRequestChange,
  disabled = false,
}: SellPricingFieldsProps) {
  const { t } = useTranslation()

  if (billingMode === 'request') {
    return (
      <Input
        label={t('models.sell_per_request')}
        {...VND_INPUT}
        value={sellPerRequest}
        onChange={(e) => onSellPerRequestChange(e.target.value)}
        placeholder="1000"
        description={t('models.price_per_request_hint')}
        disabled={disabled}
      />
    )
  }

  return (
    <div className="space-y-4">
      <p className="text-xs text-text-tertiary">{t('models.price_per_1m_hint')}</p>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <Input
          label={t('models.sell_input')}
          {...VND_INPUT}
          value={sellInputPer1m}
          onChange={(e) => onSellInputPer1mChange(e.target.value)}
          placeholder="60000"
          disabled={disabled}
        />
        <Input
          label={t('models.sell_output')}
          {...VND_INPUT}
          value={sellOutputPer1m}
          onChange={(e) => onSellOutputPer1mChange(e.target.value)}
          placeholder="250000"
          disabled={disabled}
        />
        <Input
          label={t('models.sell_cached')}
          {...VND_INPUT}
          value={sellCachedInputPer1m}
          onChange={(e) => onSellCachedInputPer1mChange(e.target.value)}
          placeholder="30000"
          disabled={disabled}
        />
        <Input
          label={t('models.sell_cache_write')}
          {...VND_INPUT}
          value={sellCacheWritePer1m}
          onChange={(e) => onSellCacheWritePer1mChange(e.target.value)}
          placeholder="37500"
          disabled={disabled}
        />
      </div>
      <div className="rounded-lg border border-border/50 bg-bg-secondary/40 p-3 space-y-3">
        <Toggle
          checked={billMinPerRequest}
          onChange={onBillMinPerRequestChange}
          label={t('models.bill_min_per_request')}
          disabled={disabled}
          size="sm"
        />
        <p className="text-xs text-text-tertiary leading-relaxed">{t('models.bill_min_per_request_desc')}</p>
        {billMinPerRequest && (
          <Input
            label={t('models.sell_min_per_request')}
            {...VND_INPUT}
            value={sellMinPerRequest}
            onChange={(e) => onSellMinPerRequestChange(e.target.value)}
            placeholder="500"
            disabled={disabled}
          />
        )}
      </div>
    </div>
  )
}