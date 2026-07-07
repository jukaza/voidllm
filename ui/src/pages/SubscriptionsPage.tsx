import { useEffect, useMemo, useRef, useState } from 'react'
import { PageHeader } from '../components/ui/PageHeader'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Textarea } from '../components/ui/Textarea'
import { Dialog, ConfirmDialog } from '../components/ui/Dialog'
import { Badge } from '../components/ui/Badge'
import { Toggle } from '../components/ui/Toggle'
import { ModelLimitPicker, type ModelLimitOption } from '../components/keys/ModelLimitPicker'
import { useTranslation } from '../lib/i18n'
import { useToast } from '../hooks/useToast'
import { useModels } from '../hooks/useModels'
import { formatCost, formatNumber, cn } from '../lib/utils'
import {
  useSubscriptionPackages,
  useSubscriptionPlans,
  useCreateSubscriptionPackage,
  useUpdateSubscriptionPackage,
  useDeleteSubscriptionPackage,
  useUploadPackageCover,
  useCreateSubscriptionPlan,
  useUpdateSubscriptionPlan,
  useDeleteSubscriptionPlan,
  useGrantUserSubscription,
  type SubscriptionPackage,
  type SubscriptionPlan,
} from '../hooks/useSubscriptions'

const COVER_PRESETS = ['aurora', 'sunset', 'ocean', 'ember', 'violet'] as const

const presetGradients: Record<string, string> = {
  aurora: 'from-violet-600/80 via-fuchsia-500/60 to-cyan-400/70',
  sunset: 'from-orange-500/80 via-rose-500/70 to-amber-400/60',
  ocean: 'from-blue-700/80 via-cyan-500/60 to-teal-400/70',
  ember: 'from-red-700/80 via-orange-600/70 to-yellow-500/50',
  violet: 'from-indigo-700/80 via-violet-600/70 to-purple-400/60',
}

function PackageCover({ pkg }: { pkg: SubscriptionPackage }) {
  if (pkg.cover_type === 'upload' || pkg.cover_type === 'url') {
    const src = pkg.cover_type === 'upload' ? pkg.cover_value : pkg.cover_value
    return (
      <div className="relative h-36 w-full overflow-hidden rounded-t-xl bg-bg-tertiary">
        {src ? (
          <img src={src} alt="" className="h-full w-full object-cover" />
        ) : (
          <div className={cn('h-full w-full bg-gradient-to-br', presetGradients.aurora)} />
        )}
        <div className="absolute inset-0 bg-gradient-to-t from-bg-secondary/90 via-transparent to-transparent" />
      </div>
    )
  }
  const preset = COVER_PRESETS.includes(pkg.cover_value as (typeof COVER_PRESETS)[number])
    ? pkg.cover_value
    : 'aurora'
  return (
    <div
      className={cn(
        'relative h-36 w-full overflow-hidden rounded-t-xl bg-gradient-to-br',
        presetGradients[preset] ?? presetGradients.aurora,
      )}
    >
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,rgba(255,255,255,0.15),transparent_55%)]" />
      <div className="absolute inset-0 bg-gradient-to-t from-bg-secondary/80 via-transparent to-transparent" />
    </div>
  )
}

export default function SubscriptionsPage() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data: packagesData, isLoading } = useSubscriptionPackages(true)
  const { data: plansData } = useSubscriptionPlans()
  const { data: modelsData } = useModels()
  const deletePackage = useDeleteSubscriptionPackage()
  const deletePlanMut = useDeleteSubscriptionPlan()

  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [editPackage, setEditPackage] = useState<SubscriptionPackage | null>(null)
  const [createPackageOpen, setCreatePackageOpen] = useState(false)
  const [editPlan, setEditPlan] = useState<SubscriptionPlan | null>(null)
  const [createPlanFor, setCreatePlanFor] = useState<string | null>(null)
  const [deletePkg, setDeletePkg] = useState<SubscriptionPackage | null>(null)
  const [deletePlan, setDeletePlan] = useState<SubscriptionPlan | null>(null)
  const [grantPlan, setGrantPlan] = useState<SubscriptionPlan | null>(null)

  const packages = packagesData?.data ?? []
  const allPlans = plansData?.data ?? []

  const selected = packages.find((p) => p.id === selectedId) ?? packages[0]
  const selectedPlans = useMemo(
    () => (selected ? allPlans.filter((pl) => pl.package_id === selected.id) : []),
    [allPlans, selected],
  )

  const modelOptions = useMemo<ModelLimitOption[]>(
    () =>
      (modelsData?.data ?? [])
        .filter((m) => m.is_active)
        .map((m) => ({ name: m.name, logo: m.logo })),
    [modelsData?.data],
  )

  return (
    <>
      <PageHeader
        title={t('subscriptions.title')}
        description={t('subscriptions.desc')}
        actions={
          <Button onClick={() => setCreatePackageOpen(true)}>{t('subscriptions.add_package')}</Button>
        }
      />

      {isLoading ? (
        <div className="rounded-xl border border-border bg-bg-secondary p-16 text-center text-sm text-text-tertiary">
          …
        </div>
      ) : packages.length === 0 ? (
        <div className="rounded-xl border border-dashed border-border bg-bg-secondary/50 p-16 text-center">
          <p className="text-sm text-text-secondary">{t('subscriptions.empty')}</p>
          <Button className="mt-4" onClick={() => setCreatePackageOpen(true)}>
            {t('subscriptions.add_package')}
          </Button>
        </div>
      ) : (
        <div className="grid gap-6 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1.4fr)]">
          <div className="space-y-3">
            <p className="text-xs font-medium uppercase tracking-wider text-text-tertiary px-1">
              {t('subscriptions.packages')}
            </p>
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
              {packages.map((pkg) => (
                <button
                  key={pkg.id}
                  type="button"
                  onClick={() => setSelectedId(pkg.id)}
                  className={cn(
                    'text-left rounded-xl border overflow-hidden transition-all hover:border-accent/40 hover:shadow-md',
                    (selected?.id === pkg.id || (!selectedId && pkg.id === packages[0]?.id))
                      ? 'border-accent/50 ring-1 ring-accent/20'
                      : 'border-border bg-bg-secondary',
                  )}
                >
                  <PackageCover pkg={pkg} />
                  <div className="p-4 space-y-1">
                    <div className="flex items-center justify-between gap-2">
                      <h3 className="font-semibold text-text-primary truncate">{pkg.name}</h3>
                      {!pkg.enabled && <Badge variant="muted">{t('subscriptions.disabled')}</Badge>}
                    </div>
                    {pkg.description && (
                      <p className="text-xs text-text-tertiary line-clamp-2">{pkg.description}</p>
                    )}
                    <p className="text-[11px] text-text-tertiary pt-1">
                      {allPlans.filter((pl) => pl.package_id === pkg.id).length}{' '}
                      {t('subscriptions.plan_count')}
                    </p>
                  </div>
                </button>
              ))}
            </div>
          </div>

          {selected && (
            <div className="space-y-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h2 className="text-lg font-semibold text-text-primary">{selected.name}</h2>
                  <p className="text-sm text-text-tertiary">{selected.description || '—'}</p>
                </div>
                <div className="flex gap-2">
                  <Button variant="secondary" size="sm" onClick={() => setEditPackage(selected)}>
                    {t('subscriptions.edit_package')}
                  </Button>
                  <Button size="sm" onClick={() => setCreatePlanFor(selected.id)}>
                    {t('subscriptions.add_plan')}
                  </Button>
                  <Button variant="destructive" size="sm" onClick={() => setDeletePkg(selected)}>
                    {t('common.delete')}
                  </Button>
                </div>
              </div>

              <div className="rounded-xl border border-border bg-bg-secondary overflow-hidden">
                <div className="grid grid-cols-[1.2fr_0.8fr_0.7fr_0.6fr_auto] gap-2 px-4 py-2.5 border-b border-border text-[11px] font-medium uppercase tracking-wide text-text-tertiary">
                  <span>{t('subscriptions.col_plan')}</span>
                  <span>{t('subscriptions.col_price')}</span>
                  <span>{t('subscriptions.col_slots')}</span>
                  <span>{t('subscriptions.col_quota')}</span>
                  <span />
                </div>
                {selectedPlans.length === 0 ? (
                  <p className="px-4 py-8 text-sm text-text-tertiary text-center">
                    {t('subscriptions.no_plans')}
                  </p>
                ) : (
                  selectedPlans.map((plan) => (
                    <PlanRow
                      key={plan.id}
                      plan={plan}
                      onEdit={() => setEditPlan(plan)}
                      onDelete={() => setDeletePlan(plan)}
                      onGrant={() => setGrantPlan(plan)}
                    />
                  ))
                )}
              </div>
            </div>
          )}
        </div>
      )}

      <PackageFormDialog
        open={createPackageOpen || editPackage !== null}
        pkg={editPackage}
        onClose={() => {
          setCreatePackageOpen(false)
          setEditPackage(null)
        }}
      />

      <PlanFormDialog
        open={createPlanFor !== null || editPlan !== null}
        packageId={createPlanFor ?? editPlan?.package_id ?? ''}
        plan={editPlan}
        modelOptions={modelOptions}
        onClose={() => {
          setCreatePlanFor(null)
          setEditPlan(null)
        }}
      />

      <GrantDialog plan={grantPlan} onClose={() => setGrantPlan(null)} />

      <ConfirmDialog
        open={deletePkg !== null}
        onClose={() => setDeletePkg(null)}
        title={t('subscriptions.delete_package_title')}
        description={t('subscriptions.delete_package_msg')}
        confirmLabel={t('common.delete')}
        onConfirm={() => {
          if (!deletePkg) return
          deletePackage.mutate(deletePkg.id, {
            onSuccess: () => {
              toast({ variant: 'success', message: t('subscriptions.deleted') })
              setDeletePkg(null)
              setSelectedId(null)
            },
            onError: (e) => toast({ variant: 'error', message: e.message }),
          })
        }}
      />

      <ConfirmDialog
        open={deletePlan !== null}
        onClose={() => setDeletePlan(null)}
        title={t('subscriptions.delete_plan_title')}
        description={t('subscriptions.delete_plan_msg')}
        confirmLabel={t('common.delete')}
        onConfirm={() => {
          if (!deletePlan) return
          deletePlanMut.mutate(deletePlan.id, {
            onSuccess: () => {
              toast({ variant: 'success', message: t('subscriptions.deleted') })
              setDeletePlan(null)
            },
            onError: (e) => toast({ variant: 'error', message: e.message }),
          })
        }}
      />
    </>
  )
}

function PlanRow({
  plan,
  onEdit,
  onDelete,
  onGrant,
}: {
  plan: SubscriptionPlan
  onEdit: () => void
  onDelete: () => void
  onGrant: () => void
}) {
  const { t } = useTranslation()
  const slotLabel =
    plan.max_concurrent_bindings > 0
      ? `${plan.active_bindings}/${plan.max_concurrent_bindings}`
      : t('subscriptions.slots_unlimited')

  const quotaParts: string[] = []
  if (plan.daily_token_limit > 0) quotaParts.push(`${formatNumber(plan.daily_token_limit)} tok/d`)
  if (plan.monthly_token_limit > 0) quotaParts.push(`${formatNumber(plan.monthly_token_limit)} tok/mo`)
  if (plan.daily_request_limit > 0) quotaParts.push(`${formatNumber(plan.daily_request_limit)} req/d`)

  return (
    <div className="grid grid-cols-[1.2fr_0.8fr_0.7fr_0.6fr_auto] gap-2 items-center px-4 py-3 border-b border-border/60 last:border-0 hover:bg-bg-primary/30">
      <div>
        <p className="text-sm font-medium text-text-primary">{plan.name}</p>
        <p className="text-[11px] text-text-tertiary">
          {plan.allowed_models.length} {t('subscriptions.models')} · {plan.validity_days}d
        </p>
      </div>
      <span className="text-sm tabular-nums text-accent">{formatCost(plan.price)}</span>
      <span className="text-xs tabular-nums text-text-secondary">{slotLabel}</span>
      <span className="text-[11px] text-text-tertiary truncate">
        {quotaParts.join(' · ') || '∞'}
      </span>
      <div className="flex gap-1 justify-end">
        <Button variant="ghost" size="sm" onClick={onGrant}>
          {t('subscriptions.grant')}
        </Button>
        <Button variant="ghost" size="sm" onClick={onEdit}>
          {t('common.edit')}
        </Button>
        <Button variant="ghost" size="sm" onClick={onDelete}>
          {t('common.delete')}
        </Button>
      </div>
    </div>
  )
}

function PackageFormDialog({
  open,
  pkg,
  onClose,
}: {
  open: boolean
  pkg: SubscriptionPackage | null
  onClose: () => void
}) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const create = useCreateSubscriptionPackage()
  const update = useUpdateSubscriptionPackage()
  const upload = useUploadPackageCover()
  const fileRef = useRef<HTMLInputElement>(null)

  const [name, setName] = useState(pkg?.name ?? '')
  const [description, setDescription] = useState(pkg?.description ?? '')
  const [coverType, setCoverType] = useState<SubscriptionPackage['cover_type']>(pkg?.cover_type ?? 'default')
  const [coverValue, setCoverValue] = useState(pkg?.cover_value ?? 'aurora')
  const [enabled, setEnabled] = useState(pkg?.enabled ?? true)

  const isEdit = pkg !== null
  const pending = create.isPending || update.isPending

  useEffect(() => {
    setName(pkg?.name ?? '')
    setDescription(pkg?.description ?? '')
    setCoverType(pkg?.cover_type ?? 'default')
    setCoverValue(pkg?.cover_value ?? 'aurora')
    setEnabled(pkg?.enabled ?? true)
  }, [pkg, open])

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    const body = {
      name: name.trim(),
      description: description.trim(),
      cover_type: coverType,
      cover_value: coverType === 'default' ? coverValue : coverValue.trim(),
      enabled,
    }
    if (isEdit && pkg) {
      update.mutate(
        { id: pkg.id, ...body },
        {
          onSuccess: () => {
            toast({ variant: 'success', message: t('subscriptions.saved') })
            onClose()
          },
          onError: (e) => toast({ variant: 'error', message: e.message }),
        },
      )
    } else {
      create.mutate(body, {
        onSuccess: () => {
          toast({ variant: 'success', message: t('subscriptions.created') })
          onClose()
        },
        onError: (e) => toast({ variant: 'error', message: e.message }),
      })
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={isEdit ? t('subscriptions.edit_package') : t('subscriptions.add_package')}
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input label={t('subscriptions.field_name')} value={name} onChange={(e) => setName(e.target.value)} required />
        <Textarea
          label={t('subscriptions.field_description')}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={3}
        />
        <div className="space-y-2">
          <p className="text-xs font-medium text-text-secondary">{t('subscriptions.field_cover')}</p>
          <div className="flex flex-wrap gap-2">
            {(['default', 'url', 'upload'] as const).map((type) => (
              <button
                key={type}
                type="button"
                onClick={() => setCoverType(type)}
                className={cn(
                  'px-3 py-1.5 rounded-md text-xs border transition-colors',
                  coverType === type
                    ? 'border-accent bg-accent/10 text-accent'
                    : 'border-border text-text-tertiary hover:border-border-strong',
                )}
              >
                {t(`subscriptions.cover_${type}`)}
              </button>
            ))}
          </div>
          {coverType === 'default' && (
            <div className="flex flex-wrap gap-2 pt-1">
              {COVER_PRESETS.map((preset) => (
                <button
                  key={preset}
                  type="button"
                  onClick={() => setCoverValue(preset)}
                  className={cn(
                    'h-8 w-14 rounded-md bg-gradient-to-br border-2',
                    presetGradients[preset],
                    coverValue === preset ? 'border-white' : 'border-transparent opacity-70',
                  )}
                  title={preset}
                />
              ))}
            </div>
          )}
          {coverType === 'url' && (
            <Input
              value={coverValue}
              onChange={(e) => setCoverValue(e.target.value)}
              placeholder="https://…"
            />
          )}
          {coverType === 'upload' && isEdit && pkg && (
            <div className="flex items-center gap-2">
              <input
                ref={fileRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={(e) => {
                  const file = e.target.files?.[0]
                  if (!file) return
                  upload.mutate(
                    { id: pkg.id, file },
                    {
                      onSuccess: () => toast({ variant: 'success', message: t('subscriptions.cover_uploaded') }),
                      onError: (err) => toast({ variant: 'error', message: err.message }),
                    },
                  )
                }}
              />
              <Button type="button" variant="secondary" size="sm" onClick={() => fileRef.current?.click()} loading={upload.isPending}>
                {t('subscriptions.upload_cover')}
              </Button>
            </div>
          )}
        </div>
        <Toggle checked={enabled} onChange={setEnabled} label={t('subscriptions.field_enabled')} />
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={onClose} type="button">
            {t('common.cancel')}
          </Button>
          <Button type="submit" loading={pending}>
            {t('common.save')}
          </Button>
        </div>
      </form>
    </Dialog>
  )
}

function PlanFormDialog({
  open,
  packageId,
  plan,
  modelOptions,
  onClose,
}: {
  open: boolean
  packageId: string
  plan: SubscriptionPlan | null
  modelOptions: ModelLimitOption[]
  onClose: () => void
}) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const create = useCreateSubscriptionPlan()
  const update = useUpdateSubscriptionPlan()

  const [name, setName] = useState(plan?.name ?? '')
  const [description, setDescription] = useState(plan?.description ?? '')
  const [price, setPrice] = useState(plan ? String(plan.price) : '')
  const [validityDays, setValidityDays] = useState(plan ? String(plan.validity_days) : '30')
  const [maxSlots, setMaxSlots] = useState(plan ? String(plan.max_concurrent_bindings) : '0')
  const [dailyTokens, setDailyTokens] = useState(plan?.daily_token_limit ? String(plan.daily_token_limit) : '')
  const [monthlyTokens, setMonthlyTokens] = useState(plan?.monthly_token_limit ? String(plan.monthly_token_limit) : '')
  const [dailyReq, setDailyReq] = useState(plan?.daily_request_limit ? String(plan.daily_request_limit) : '')
  const [monthlyReq, setMonthlyReq] = useState(plan?.monthly_request_limit ? String(plan.monthly_request_limit) : '')
  const [rpm, setRpm] = useState(plan?.requests_per_minute ? String(plan.requests_per_minute) : '')
  const [rpd, setRpd] = useState(plan?.requests_per_day ? String(plan.requests_per_day) : '')
  const [models, setModels] = useState<string[]>(plan?.allowed_models ?? [])
  const [policy, setPolicy] = useState<'block' | 'fallback_wallet'>(plan?.quota_exceeded_policy ?? 'fallback_wallet')
  const [forSale, setForSale] = useState(plan?.for_sale ?? true)

  const isEdit = plan !== null
  const pending = create.isPending || update.isPending

  useEffect(() => {
    if (!open) return
    setName(plan?.name ?? '')
    setDescription(plan?.description ?? '')
    setPrice(plan != null ? String(plan.price) : '')
    setValidityDays(plan != null ? String(plan.validity_days) : '30')
    setMaxSlots(plan != null ? String(plan.max_concurrent_bindings) : '0')
    setDailyTokens(plan != null && plan.daily_token_limit > 0 ? String(plan.daily_token_limit) : '')
    setMonthlyTokens(plan != null && plan.monthly_token_limit > 0 ? String(plan.monthly_token_limit) : '')
    setDailyReq(plan != null && plan.daily_request_limit > 0 ? String(plan.daily_request_limit) : '')
    setMonthlyReq(plan != null && plan.monthly_request_limit > 0 ? String(plan.monthly_request_limit) : '')
    setRpm(plan != null && plan.requests_per_minute > 0 ? String(plan.requests_per_minute) : '')
    setRpd(plan != null && plan.requests_per_day > 0 ? String(plan.requests_per_day) : '')
    setModels(plan?.allowed_models ?? [])
    setPolicy(plan?.quota_exceeded_policy ?? 'fallback_wallet')
    setForSale(plan?.for_sale ?? true)
  }, [plan, open])

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim() || !packageId) return
    const body = {
      package_id: packageId,
      name: name.trim(),
      description: description.trim(),
      price: parseFloat(price) || 0,
      validity_days: parseInt(validityDays, 10) || 30,
      max_concurrent_bindings: parseInt(maxSlots, 10) || 0,
      daily_token_limit: parseInt(dailyTokens, 10) || 0,
      monthly_token_limit: parseInt(monthlyTokens, 10) || 0,
      daily_request_limit: parseInt(dailyReq, 10) || 0,
      monthly_request_limit: parseInt(monthlyReq, 10) || 0,
      requests_per_minute: parseInt(rpm, 10) || 0,
      requests_per_day: parseInt(rpd, 10) || 0,
      allowed_models: models,
      quota_exceeded_policy: policy,
      for_sale: forSale,
    }
    if (isEdit && plan) {
      const { package_id: _pkg, ...patch } = body
      update.mutate(
        { id: plan.id, ...patch },
        {
          onSuccess: () => {
            toast({ variant: 'success', message: t('subscriptions.saved') })
            onClose()
          },
          onError: (e) => toast({ variant: 'error', message: e.message }),
        },
      )
    } else {
      create.mutate(body, {
        onSuccess: () => {
          toast({ variant: 'success', message: t('subscriptions.created') })
          onClose()
        },
        onError: (e) => toast({ variant: 'error', message: e.message }),
      })
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={isEdit ? t('subscriptions.edit_plan') : t('subscriptions.add_plan')}
      panelClassName="max-w-2xl"
    >
      {isEdit && plan && (
        <p className="mb-4 text-sm text-text-secondary">
          <span className="font-medium text-text-primary">{plan.name}</span>
          {' · '}
          {formatCost(plan.price)}
          {' · '}
          {plan.validity_days}d
        </p>
      )}
      <form onSubmit={handleSubmit} className="space-y-4 max-h-[70vh] overflow-y-auto pr-1">
        <div className="grid grid-cols-2 gap-4">
          <Input label={t('subscriptions.field_name')} value={name} onChange={(e) => setName(e.target.value)} required />
          <Input label={t('subscriptions.field_price')} type="number" value={price} onChange={(e) => setPrice(e.target.value)} />
        </div>
        <Textarea label={t('subscriptions.field_description')} value={description} onChange={(e) => setDescription(e.target.value)} rows={2} />
        <div className="grid grid-cols-2 gap-4">
          <Input label={t('subscriptions.field_validity')} type="number" value={validityDays} onChange={(e) => setValidityDays(e.target.value)} />
          <Input label={t('subscriptions.field_slots')} type="number" value={maxSlots} onChange={(e) => setMaxSlots(e.target.value)} placeholder="0 = ∞" />
        </div>
        <div className="grid grid-cols-2 gap-4">
          <Input label={t('subscriptions.field_daily_tokens')} type="number" value={dailyTokens} onChange={(e) => setDailyTokens(e.target.value)} placeholder="∞" />
          <Input label={t('subscriptions.field_monthly_tokens')} type="number" value={monthlyTokens} onChange={(e) => setMonthlyTokens(e.target.value)} placeholder="∞" />
        </div>
        <div className="grid grid-cols-2 gap-4">
          <Input label={t('subscriptions.field_daily_requests')} type="number" value={dailyReq} onChange={(e) => setDailyReq(e.target.value)} placeholder="∞" />
          <Input label={t('subscriptions.field_monthly_requests')} type="number" value={monthlyReq} onChange={(e) => setMonthlyReq(e.target.value)} placeholder="∞" />
        </div>
        <div className="grid grid-cols-2 gap-4">
          <Input label={t('keys.create.rpm')} type="number" value={rpm} onChange={(e) => setRpm(e.target.value)} placeholder="∞" />
          <Input label={t('keys.create.rpd')} type="number" value={rpd} onChange={(e) => setRpd(e.target.value)} placeholder="∞" />
        </div>
        <div className="space-y-2">
          <p className="text-xs text-text-tertiary">{t('subscriptions.field_models')}</p>
          <ModelLimitPicker models={modelOptions} selected={models} onChange={setModels} />
        </div>
        <div className="flex gap-4">
          <Toggle checked={forSale} onChange={setForSale} label={t('subscriptions.field_for_sale')} />
        </div>
        <div className="space-y-2">
          <p className="text-xs font-medium text-text-secondary">{t('subscriptions.field_quota_policy')}</p>
          <div className="flex gap-2">
            {(['fallback_wallet', 'block'] as const).map((p) => (
              <button
                key={p}
                type="button"
                onClick={() => setPolicy(p)}
                className={cn(
                  'px-3 py-1.5 rounded-md text-xs border',
                  policy === p ? 'border-accent bg-accent/10 text-accent' : 'border-border text-text-tertiary',
                )}
              >
                {t(`subscriptions.policy_${p}`)}
              </button>
            ))}
          </div>
        </div>
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={onClose} type="button">{t('common.cancel')}</Button>
          <Button type="submit" loading={pending}>{t('common.save')}</Button>
        </div>
      </form>
    </Dialog>
  )
}

function GrantDialog({ plan, onClose }: { plan: SubscriptionPlan | null; onClose: () => void }) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const grant = useGrantUserSubscription()
  const [userId, setUserId] = useState('')
  const [days, setDays] = useState('')

  return (
    <Dialog open={plan !== null} onClose={onClose} title={t('subscriptions.grant_title')}>
      <form
        className="space-y-4"
        onSubmit={(e) => {
          e.preventDefault()
          if (!plan || !userId.trim()) return
          grant.mutate(
            {
              user_id: userId.trim(),
              plan_id: plan.id,
              days: days.trim() ? parseInt(days, 10) : undefined,
            },
            {
              onSuccess: () => {
                toast({ variant: 'success', message: t('subscriptions.granted') })
                onClose()
                setUserId('')
                setDays('')
              },
              onError: (e) => toast({ variant: 'error', message: e.message }),
            },
          )
        }}
      >
        <p className="text-sm text-text-secondary">
          {plan?.name} — {t('subscriptions.grant_hint')}
        </p>
        <Input label={t('subscriptions.field_user_id')} value={userId} onChange={(e) => setUserId(e.target.value)} required />
        <Input label={t('subscriptions.field_grant_days')} type="number" value={days} onChange={(e) => setDays(e.target.value)} placeholder={String(plan?.validity_days ?? 30)} />
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose} type="button">{t('common.cancel')}</Button>
          <Button type="submit" loading={grant.isPending}>{t('subscriptions.grant')}</Button>
        </div>
      </form>
    </Dialog>
  )
}

