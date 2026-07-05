import { useEffect, useState } from 'react'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Select } from '../../components/ui/Select'
import { Toggle } from '../../components/ui/Toggle'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import {
  useAdminPaymentSettings,
  useUpdatePaymentSettings,
  type BonusType,
  type Campaign,
  type FirstTopupBonus,
  type PaymentSettings,
  type TierBonus,
} from '../../hooks/usePaymentSettings'
import { formatCost } from '../../lib/utils'
import { LiveBadge } from './components/PreviewBadge'
import { SettingsSectionCard } from './components/SettingsSectionCard'
import { SettingsTabFooter } from './components/SettingsTabFooter'

function SectionCard({
  title,
  description,
  children,
  badge,
}: {
  title: string
  description?: string
  children: React.ReactNode
  badge?: React.ReactNode
}) {
  return (
    <SettingsSectionCard title={title} description={description} badge={badge} className="mb-6">
      {children}
    </SettingsSectionCard>
  )
}

function inferBonusType(item: {
  bonus_type?: BonusType
  bonus_percent: number
  bonus_fixed: number
}): BonusType {
  if (item.bonus_type === 'fixed' || item.bonus_type === 'percent') return item.bonus_type
  return item.bonus_fixed > 0 && item.bonus_percent <= 0 ? 'fixed' : 'percent'
}

function normalizeTier(tier: TierBonus): TierBonus {
  const bonus_type = inferBonusType(tier)
  return bonus_type === 'fixed'
    ? { ...tier, bonus_type, bonus_percent: 0 }
    : { ...tier, bonus_type: 'percent', bonus_fixed: 0 }
}

function normalizeCampaign(campaign: Campaign): Campaign {
  const bonus_type = inferBonusType(campaign)
  return bonus_type === 'fixed'
    ? { ...campaign, bonus_type, bonus_percent: 0, max_bonus: 0 }
    : { ...campaign, bonus_type: 'percent', bonus_fixed: 0 }
}

function normalizeFirstTopup(firstTopup: FirstTopupBonus): FirstTopupBonus {
  const bonus_type = inferBonusType(firstTopup)
  return bonus_type === 'fixed'
    ? { ...firstTopup, bonus_type, bonus_percent: 0 }
    : { ...firstTopup, bonus_type: 'percent', bonus_fixed: 0 }
}

function normalizePaymentForm(data: PaymentSettings): PaymentSettings {
  return {
    ...data,
    sepay: {
      ...data.sepay,
      webhook_auth_mode: data.sepay.webhook_auth_mode ?? data.webhook_auth_mode ?? 'api_key',
      webhook_ip_check: data.sepay.webhook_ip_check ?? false,
    },
    tier_bonuses: data.tier_bonuses.map(normalizeTier),
    campaigns: data.campaigns.map(normalizeCampaign),
    first_topup: normalizeFirstTopup(data.first_topup),
  }
}

const emptyTier = (): TierBonus => ({
  min_amount: 100000,
  bonus_type: 'percent',
  bonus_percent: 10,
  bonus_fixed: 0,
  label: '',
})
const emptyCampaign = (): Campaign => ({
  id: `camp-${Date.now()}`,
  name: '',
  enabled: true,
  start_at: new Date().toISOString().slice(0, 16),
  end_at: new Date(Date.now() + 7 * 86400000).toISOString().slice(0, 16),
  bonus_type: 'percent',
  bonus_percent: 10,
  bonus_fixed: 0,
  min_amount: 50000,
  max_bonus: 0,
  first_topup_only: false,
})

function BonusTypeSelect({
  label,
  value,
  onChange,
}: {
  label: string
  value: BonusType
  onChange: (value: BonusType) => void
}) {
  const { t } = useTranslation()
  return (
    <Select
      label={label}
      value={value}
      options={[
        { value: 'percent', label: t('settings.payment_bonus_type_percent') },
        { value: 'fixed', label: t('settings.payment_bonus_type_fixed') },
      ]}
      onChange={(v) => onChange(v as BonusType)}
    />
  )
}

export function PaymentSettingsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data, isLoading, isError, error, refetch } = useAdminPaymentSettings()
  const update = useUpdatePaymentSettings()

  const [form, setForm] = useState<PaymentSettings | null>(null)
  const [webhookToken, setWebhookToken] = useState('')
  const [webhookSecret, setWebhookSecret] = useState('')
  const [newPreset, setNewPreset] = useState('')

  useEffect(() => {
    if (data) setForm(normalizePaymentForm(data))
  }, [data])

  function save() {
    if (!form) return
    const payload: Partial<PaymentSettings> = {
      sepay: {
        ...form.sepay,
        webhook_token: webhookToken || undefined,
        webhook_secret: webhookSecret || undefined,
      },
      amount_presets: form.amount_presets,
      tier_bonuses: form.tier_bonuses.map(normalizeTier),
      campaigns: form.campaigns.map((c) => ({
        ...normalizeCampaign(c),
        start_at: new Date(c.start_at).toISOString(),
        end_at: new Date(c.end_at).toISOString(),
      })),
      first_topup: normalizeFirstTopup(form.first_topup),
      bonus_stack_mode: form.bonus_stack_mode,
    }
    update.mutate(payload, {
      onSuccess: () => {
        setWebhookToken('')
        setWebhookSecret('')
        toast({ variant: 'success', message: t('common.saved') })
      },
      onError: (e) => toast({ variant: 'error', message: e.message }),
    })
  }

  if (isLoading) return <p className="text-sm text-text-tertiary">{t('common.loading')}</p>
  if (isError || !form) {
    return (
      <div>
        <p className="text-sm text-error">{error?.message ?? t('settings.payment_load_error')}</p>
        <Button className="mt-3" variant="secondary" onClick={() => void refetch()}>
          {t('common.refresh')}
        </Button>
      </div>
    )
  }

  const bankOptions = (form.sepay.bank_code ? [{ value: form.sepay.bank_code, label: form.sepay.bank_code }] : []).concat(
    [
      { value: 'MB', label: 'MB Bank' },
      { value: 'VCB', label: 'Vietcombank' },
      { value: 'TCB', label: 'Techcombank' },
      { value: 'BIDV', label: 'BIDV' },
      { value: 'ACB', label: 'ACB' },
      { value: 'VPB', label: 'VPBank' },
    ].filter((b) => b.value !== form.sepay.bank_code),
  )

  const authMode = form.sepay.webhook_auth_mode ?? form.webhook_auth_mode ?? 'api_key'

  return (
    <div>
      <SectionCard title={t('settings.payment_setup_title')} description={t('settings.payment_setup_desc')}>
        <ol className="list-decimal list-inside space-y-2 text-sm text-text-secondary">
          <li>{t('settings.payment_setup_step1')}</li>
          <li>{t('settings.payment_setup_step2')}</li>
          <li>{t('settings.payment_setup_step3')}</li>
          <li>{t('settings.payment_setup_step4')}</li>
          <li>{t('settings.payment_setup_step5')}</li>
          <li>{t('settings.payment_setup_step6')}</li>
        </ol>
        <div className="rounded-md border border-border bg-bg-tertiary px-3 py-2 text-xs text-text-tertiary space-y-2">
          <p>{t('settings.payment_setup_docs')}</p>
          <ul className="list-disc list-inside space-y-1">
            <li>
              <a className="text-accent hover:underline" href="https://docs.sepay.vn/tich-hop-webhooks.html" target="_blank" rel="noreferrer">
                {t('settings.payment_setup_doc_integration')}
              </a>
            </li>
            <li>
              <a className="text-accent hover:underline" href="https://docs.sepay.vn/lap-trinh-webhooks.html" target="_blank" rel="noreferrer">
                {t('settings.payment_setup_doc_programming')}
              </a>
            </li>
            <li>
              <a className="text-accent hover:underline" href="https://developer.sepay.vn/vi/sepay-webhooks/xac-thuc" target="_blank" rel="noreferrer">
                {t('settings.payment_setup_doc_auth')}
              </a>
            </li>
          </ul>
          <p className="text-text-secondary">{t('settings.payment_setup_ip_note')}</p>
          <code className="block break-all text-[11px]">172.236.138.20, 172.233.83.68, 171.244.35.2, 151.158.108.68, 151.158.109.79, 103.255.238.139</code>
        </div>
      </SectionCard>

      <SectionCard
        title={t('settings.payment_sepay_title')}
        description={t('settings.payment_sepay_desc')}
        badge={<LiveBadge />}
      >
        <Toggle
          checked={form.sepay.enabled}
          onChange={(v) => setForm({ ...form, sepay: { ...form.sepay, enabled: v } })}
          label={t('settings.payment_enabled')}
        />
        <Select
          label={t('settings.payment_bank')}
          value={form.sepay.bank_code}
          options={bankOptions}
          onChange={(v) => setForm({ ...form, sepay: { ...form.sepay, bank_code: v } })}
        />
        <Input
          label={t('settings.payment_account_number')}
          value={form.sepay.account_number}
          onChange={(e) => setForm({ ...form, sepay: { ...form.sepay, account_number: e.target.value } })}
        />
        <Input
          label={t('settings.payment_account_name')}
          value={form.sepay.account_name}
          onChange={(e) => setForm({ ...form, sepay: { ...form.sepay, account_name: e.target.value } })}
        />
        <div>
          <Select
            label={t('settings.payment_webhook_auth_mode')}
            value={authMode}
            options={[
              { value: 'api_key', label: t('settings.payment_webhook_auth_api_key') },
              { value: 'hmac', label: t('settings.payment_webhook_auth_hmac') },
            ]}
            onChange={(v) =>
              setForm({
                ...form,
                sepay: { ...form.sepay, webhook_auth_mode: v as 'api_key' | 'hmac' },
              })
            }
          />
          <p className="mt-1 text-xs text-text-tertiary">{t('settings.payment_webhook_auth_mode_hint')}</p>
        </div>
        {authMode === 'api_key' ? (
          <Input
            label={t('settings.payment_webhook_token')}
            type="password"
            value={webhookToken}
            onChange={(e) => setWebhookToken(e.target.value)}
            placeholder={form.webhook_token_configured ? '••••••••' : ''}
            description={t('settings.payment_webhook_token_hint')}
          />
        ) : (
          <Input
            label={t('settings.payment_webhook_secret')}
            type="password"
            value={webhookSecret}
            onChange={(e) => setWebhookSecret(e.target.value)}
            placeholder={form.webhook_secret_configured ? '••••••••' : ''}
            description={t('settings.payment_webhook_secret_hint')}
          />
        )}
        <div>
          <Toggle
            checked={form.sepay.webhook_ip_check ?? false}
            onChange={(v) => setForm({ ...form, sepay: { ...form.sepay, webhook_ip_check: v } })}
            label={t('settings.payment_webhook_ip_check')}
          />
          <p className="mt-1 text-xs text-text-tertiary">{t('settings.payment_webhook_ip_check_hint')}</p>
        </div>
        <div className="rounded-md border border-border bg-bg-tertiary px-3 py-2 text-xs">
          <div className="flex items-center justify-between gap-2 mb-1">
            <div className="text-text-tertiary">{t('settings.payment_webhook_url')}</div>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => {
                void navigator.clipboard.writeText(form.webhook_url)
                toast({ variant: 'success', message: t('common.copied') })
              }}
            >
              {t('common.copy')}
            </Button>
          </div>
          <code className="break-all">{form.webhook_url}</code>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <Input
            label={t('settings.payment_min_amount')}
            type="number"
            value={String(form.sepay.min_amount)}
            onChange={(e) => setForm({ ...form, sepay: { ...form.sepay, min_amount: Number(e.target.value) } })}
          />
          <Input
            label={t('settings.payment_max_amount')}
            type="number"
            value={String(form.sepay.max_amount)}
            onChange={(e) => setForm({ ...form, sepay: { ...form.sepay, max_amount: Number(e.target.value) } })}
          />
          <Input
            label={t('settings.payment_order_ttl')}
            type="number"
            value={String(form.sepay.order_ttl_minutes)}
            onChange={(e) => setForm({ ...form, sepay: { ...form.sepay, order_ttl_minutes: Number(e.target.value) } })}
          />
        </div>
      </SectionCard>

      <SectionCard title={t('settings.payment_presets_title')} description={t('settings.payment_presets_desc')}>
        <div className="flex flex-wrap gap-2">
          {form.amount_presets.map((preset) => (
            <Button
              key={preset}
              size="sm"
              variant="secondary"
              onClick={() =>
                setForm({
                  ...form,
                  amount_presets: form.amount_presets.filter((p) => p !== preset),
                })
              }
            >
              {formatCost(preset)} ×
            </Button>
          ))}
        </div>
        <div className="flex gap-2">
          <Input
            label={t('settings.payment_add_preset')}
            type="number"
            value={newPreset}
            onChange={(e) => setNewPreset(e.target.value)}
          />
          <Button
            className="self-end"
            variant="secondary"
            onClick={() => {
              const v = Number(newPreset)
              if (v > 0 && !form.amount_presets.includes(v)) {
                setForm({ ...form, amount_presets: [...form.amount_presets, v].sort((a, b) => a - b) })
                setNewPreset('')
              }
            }}
          >
            {t('common.add')}
          </Button>
        </div>
      </SectionCard>

      <SectionCard title={t('settings.payment_tiers_title')} description={t('settings.payment_tiers_desc')}>
        <Select
          label={t('settings.payment_stack_mode')}
          value={form.bonus_stack_mode}
          options={[
            { value: 'stack', label: t('settings.payment_stack_mode_stack') },
            { value: 'max', label: t('settings.payment_stack_mode_max') },
          ]}
          onChange={(v) => setForm({ ...form, bonus_stack_mode: v as 'stack' | 'max' })}
        />
        {form.tier_bonuses.map((tier, idx) => (
          <div key={idx} className="grid grid-cols-1 sm:grid-cols-4 gap-3 p-3 rounded-md border border-border">
            <Input
              label={t('settings.payment_tier_min')}
              type="number"
              value={String(tier.min_amount)}
              onChange={(e) => {
                const tiers = [...form.tier_bonuses]
                tiers[idx] = { ...tier, min_amount: Number(e.target.value) }
                setForm({ ...form, tier_bonuses: tiers })
              }}
            />
            <BonusTypeSelect
              label={t('settings.payment_bonus_type')}
              value={tier.bonus_type ?? inferBonusType(tier)}
              onChange={(bonus_type) => {
                const tiers = [...form.tier_bonuses]
                tiers[idx] = normalizeTier({ ...tier, bonus_type })
                setForm({ ...form, tier_bonuses: tiers })
              }}
            />
            {inferBonusType(tier) === 'percent' ? (
              <Input
                label={t('settings.payment_tier_percent')}
                type="number"
                value={String(tier.bonus_percent)}
                onChange={(e) => {
                  const tiers = [...form.tier_bonuses]
                  tiers[idx] = normalizeTier({ ...tier, bonus_percent: Number(e.target.value) })
                  setForm({ ...form, tier_bonuses: tiers })
                }}
              />
            ) : (
              <Input
                label={t('settings.payment_tier_fixed')}
                type="number"
                value={String(tier.bonus_fixed)}
                onChange={(e) => {
                  const tiers = [...form.tier_bonuses]
                  tiers[idx] = normalizeTier({ ...tier, bonus_fixed: Number(e.target.value) })
                  setForm({ ...form, tier_bonuses: tiers })
                }}
              />
            )}
            <Button
              className="self-end"
              variant="destructive"
              size="sm"
              onClick={() =>
                setForm({ ...form, tier_bonuses: form.tier_bonuses.filter((_, i) => i !== idx) })
              }
            >
              {t('common.delete')}
            </Button>
          </div>
        ))}
        <Button variant="secondary" onClick={() => setForm({ ...form, tier_bonuses: [...form.tier_bonuses, emptyTier()] })}>
          {t('settings.payment_add_tier')}
        </Button>
      </SectionCard>

      <SectionCard title={t('settings.payment_campaigns_title')} description={t('settings.payment_campaigns_desc')}>
        {form.campaigns.map((campaign, idx) => (
          <div key={campaign.id} className="space-y-3 p-4 rounded-md border border-border">
            <div className="flex items-center justify-between gap-3">
              <Input
                label={t('settings.payment_campaign_name')}
                value={campaign.name}
                onChange={(e) => {
                  const campaigns = [...form.campaigns]
                  campaigns[idx] = { ...campaign, name: e.target.value }
                  setForm({ ...form, campaigns })
                }}
              />
              <Toggle
                checked={campaign.enabled}
                onChange={(v) => {
                  const campaigns = [...form.campaigns]
                  campaigns[idx] = { ...campaign, enabled: v }
                  setForm({ ...form, campaigns })
                }}
                label={t('settings.payment_campaign_enabled')}
              />
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <Input
                label={t('settings.payment_campaign_start')}
                type="datetime-local"
                value={campaign.start_at.slice(0, 16)}
                onChange={(e) => {
                  const campaigns = [...form.campaigns]
                  campaigns[idx] = { ...campaign, start_at: e.target.value }
                  setForm({ ...form, campaigns })
                }}
              />
              <Input
                label={t('settings.payment_campaign_end')}
                type="datetime-local"
                value={campaign.end_at.slice(0, 16)}
                onChange={(e) => {
                  const campaigns = [...form.campaigns]
                  campaigns[idx] = { ...campaign, end_at: e.target.value }
                  setForm({ ...form, campaigns })
                }}
              />
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <BonusTypeSelect
                label={t('settings.payment_bonus_type')}
                value={campaign.bonus_type ?? inferBonusType(campaign)}
                onChange={(bonus_type) => {
                  const campaigns = [...form.campaigns]
                  campaigns[idx] = normalizeCampaign({ ...campaign, bonus_type })
                  setForm({ ...form, campaigns })
                }}
              />
              {inferBonusType(campaign) === 'percent' ? (
                <Input
                  label={t('settings.payment_tier_percent')}
                  type="number"
                  value={String(campaign.bonus_percent)}
                  onChange={(e) => {
                    const campaigns = [...form.campaigns]
                    campaigns[idx] = normalizeCampaign({ ...campaign, bonus_percent: Number(e.target.value) })
                    setForm({ ...form, campaigns })
                  }}
                />
              ) : (
                <Input
                  label={t('settings.payment_tier_fixed')}
                  type="number"
                  value={String(campaign.bonus_fixed)}
                  onChange={(e) => {
                    const campaigns = [...form.campaigns]
                    campaigns[idx] = normalizeCampaign({ ...campaign, bonus_fixed: Number(e.target.value) })
                    setForm({ ...form, campaigns })
                  }}
                />
              )}
              <Input
                label={t('settings.payment_tier_min')}
                type="number"
                value={String(campaign.min_amount)}
                onChange={(e) => {
                  const campaigns = [...form.campaigns]
                  campaigns[idx] = { ...campaign, min_amount: Number(e.target.value) }
                  setForm({ ...form, campaigns })
                }}
              />
              {inferBonusType(campaign) === 'percent' && (
                <Input
                  label={t('settings.payment_campaign_max_bonus')}
                  type="number"
                  value={String(campaign.max_bonus)}
                  onChange={(e) => {
                    const campaigns = [...form.campaigns]
                    campaigns[idx] = { ...campaign, max_bonus: Number(e.target.value) }
                    setForm({ ...form, campaigns })
                  }}
                />
              )}
            </div>
            <Toggle
              checked={campaign.first_topup_only}
              onChange={(v) => {
                const campaigns = [...form.campaigns]
                campaigns[idx] = { ...campaign, first_topup_only: v }
                setForm({ ...form, campaigns })
              }}
              label={t('settings.payment_campaign_first_only')}
            />
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setForm({ ...form, campaigns: form.campaigns.filter((_, i) => i !== idx) })}
            >
              {t('common.delete')}
            </Button>
          </div>
        ))}
        <Button variant="secondary" onClick={() => setForm({ ...form, campaigns: [...form.campaigns, emptyCampaign()] })}>
          {t('settings.payment_add_campaign')}
        </Button>
      </SectionCard>

      <SectionCard title={t('settings.payment_first_topup_title')} description={t('settings.payment_first_topup_desc')}>
        <Toggle
          checked={form.first_topup.enabled}
          onChange={(v) => setForm({ ...form, first_topup: { ...form.first_topup, enabled: v } })}
          label={t('settings.payment_first_topup_enabled')}
        />
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <BonusTypeSelect
            label={t('settings.payment_bonus_type')}
            value={form.first_topup.bonus_type ?? inferBonusType(form.first_topup)}
            onChange={(bonus_type) =>
              setForm({ ...form, first_topup: normalizeFirstTopup({ ...form.first_topup, bonus_type }) })
            }
          />
          {inferBonusType(form.first_topup) === 'percent' ? (
            <Input
              label={t('settings.payment_tier_percent')}
              type="number"
              value={String(form.first_topup.bonus_percent)}
              onChange={(e) =>
                setForm({
                  ...form,
                  first_topup: normalizeFirstTopup({
                    ...form.first_topup,
                    bonus_percent: Number(e.target.value),
                  }),
                })
              }
            />
          ) : (
            <Input
              label={t('settings.payment_tier_fixed')}
              type="number"
              value={String(form.first_topup.bonus_fixed)}
              onChange={(e) =>
                setForm({
                  ...form,
                  first_topup: normalizeFirstTopup({
                    ...form.first_topup,
                    bonus_fixed: Number(e.target.value),
                  }),
                })
              }
            />
          )}
        </div>
      </SectionCard>

      <SettingsTabFooter mode="live" loading={update.isPending} onSave={save} />
    </div>
  )
}