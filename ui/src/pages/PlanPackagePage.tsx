import { useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { PackageCover } from '../components/subscriptions/PackageCover'
import { SePayPaymentDialog } from '../components/wallet/SePayPaymentDialog'
import { useTranslation } from '../lib/i18n'
import { useToast } from '../hooks/useToast'
import { formatCost, formatNumber, cn } from '../lib/utils'
import {
  usePublicSubscriptionCatalog,
  useCreateSubscriptionOrder,
  type PublicSubscriptionPlan,
  type SepayOrderLike,
} from '../hooks/useSubscriptions'
import { useQueryClient } from '@tanstack/react-query'
import type { SepayOrder } from '../hooks/usePaymentSettings'

function PlanOptionCard({
  plan,
  onBuy,
  buying,
}: {
  plan: PublicSubscriptionPlan
  onBuy: () => void
  buying: boolean
}) {
  const { t } = useTranslation()
  const soldOut = plan.slots_remaining !== undefined && plan.slots_remaining <= 0

  return (
    <div className="rounded-xl border border-border bg-bg-primary/50 p-5 flex flex-col gap-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-lg font-bold text-text-primary">{plan.name}</h3>
          {plan.description && (
            <p className="text-sm text-text-tertiary mt-1">{plan.description}</p>
          )}
        </div>
        <span className="text-xl font-bold text-emerald-400 tabular-nums shrink-0">
          {formatCost(plan.price)}
        </span>
      </div>

      <div className="grid gap-2 sm:grid-cols-2 text-sm">
        <div className="rounded-lg bg-bg-secondary px-3 py-2">
          <p className="text-[10px] uppercase text-text-tertiary">{t('plans.validity')}</p>
          <p className="font-semibold text-text-primary">{plan.validity_days} {t('plans.days')}</p>
        </div>
        {plan.daily_token_limit > 0 && (
          <div className="rounded-lg bg-bg-secondary px-3 py-2">
            <p className="text-[10px] uppercase text-text-tertiary">{t('plans.daily_tokens')}</p>
            <p className="font-semibold text-text-primary tabular-nums">{formatNumber(plan.daily_token_limit)}</p>
          </div>
        )}
        {plan.monthly_token_limit > 0 && (
          <div className="rounded-lg bg-bg-secondary px-3 py-2">
            <p className="text-[10px] uppercase text-text-tertiary">{t('plans.monthly_tokens')}</p>
            <p className="font-semibold text-text-primary tabular-nums">{formatNumber(plan.monthly_token_limit)}</p>
          </div>
        )}
        {plan.daily_request_limit > 0 && (
          <div className="rounded-lg bg-bg-secondary px-3 py-2">
            <p className="text-[10px] uppercase text-text-tertiary">{t('plans.daily_requests')}</p>
            <p className="font-semibold text-text-primary tabular-nums">{formatNumber(plan.daily_request_limit)}</p>
          </div>
        )}
        {plan.monthly_request_limit > 0 && (
          <div className="rounded-lg bg-bg-secondary px-3 py-2">
            <p className="text-[10px] uppercase text-text-tertiary">{t('plans.monthly_requests')}</p>
            <p className="font-semibold text-text-primary tabular-nums">{formatNumber(plan.monthly_request_limit)}</p>
          </div>
        )}
      </div>

      <div className="flex flex-wrap gap-1.5">
        {plan.allowed_models?.length > 0 && (
          <Badge variant="info">{plan.allowed_models.length} models</Badge>
        )}
        {plan.slots_remaining !== undefined && (
          <Badge variant={soldOut ? 'warning' : 'default'}>
            {soldOut ? t('plans.sold_out') : `${plan.slots_remaining} ${t('plans.slots_left')}`}
          </Badge>
        )}
      </div>

      {plan.allowed_models?.length > 0 && (
        <div className="rounded-lg border border-border/60 bg-bg-secondary/50 p-3">
          <p className="text-[10px] font-semibold uppercase text-text-tertiary mb-2">{t('plans.included_models')}</p>
          <div className="flex flex-wrap gap-1.5">
            {plan.allowed_models.map((m) => (
              <span key={m} className="rounded-md bg-bg-primary px-2 py-0.5 font-mono text-[10px] text-text-secondary">
                {m}
              </span>
            ))}
          </div>
        </div>
      )}

      <Button className="w-full mt-auto" onClick={onBuy} loading={buying} disabled={soldOut}>
        {soldOut ? t('plans.sold_out') : t('plans.buy')}
      </Button>
    </div>
  )
}

export default function PlanPackagePage() {
  const { packageId } = useParams<{ packageId: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { toast } = useToast()
  const queryClient = useQueryClient()
  const { data, isLoading } = usePublicSubscriptionCatalog()
  const createOrder = useCreateSubscriptionOrder()

  const [sepayOpen, setSepayOpen] = useState(false)
  const [sepayOrder, setSepayOrder] = useState<SepayOrder | null>(null)
  const [buyingPlanId, setBuyingPlanId] = useState<string | null>(null)

  const pkg = useMemo(
    () => (data?.data ?? []).find((p) => p.id === packageId),
    [data?.data, packageId],
  )

  function toSepayOrder(order: SepayOrderLike): SepayOrder {
    return {
      trade_no: order.trade_no ?? '',
      pay_amount: order.pay_amount,
      credit_amount: order.credit_amount ?? 0,
      bonus_amount: order.bonus_amount ?? 0,
      bank_code: order.bank_code ?? '',
      bank_name: order.bank_name ?? '',
      account_number: order.account_number ?? '',
      account_name: order.account_name ?? '',
      qr_url: order.qr_url ?? '',
      expires_at: order.expires_at ?? '',
      status: order.status,
    }
  }

  function onPaymentSuccess() {
    queryClient.invalidateQueries({ queryKey: ['my-subscriptions'] })
    queryClient.invalidateQueries({ queryKey: ['public-subscription-packages'] })
    queryClient.invalidateQueries({ queryKey: ['my-wallet'] })
    toast({ variant: 'success', message: t('plans.purchase_success') })
    navigate('/my-subscriptions')
  }

  function handleBuy(plan: PublicSubscriptionPlan) {
    setBuyingPlanId(plan.id)
    createOrder.mutate(plan.id, {
      onSuccess: (order) => {
        if (order.payment_method === 'wallet') {
          onPaymentSuccess()
          setBuyingPlanId(null)
          return
        }
        setSepayOrder(toSepayOrder(order))
        setSepayOpen(true)
        setBuyingPlanId(null)
      },
      onError: (e) => {
        toast({ variant: 'error', message: e.message })
        setBuyingPlanId(null)
      },
    })
  }

  if (isLoading) {
    return <div className="p-20 text-center text-text-tertiary">…</div>
  }

  if (!pkg) {
    return (
      <div className="space-y-4 p-10 text-center">
        <p className="text-text-secondary">{t('plans.package_not_found')}</p>
        <Link to="/plans">
          <Button variant="secondary">{t('plans.back_to_store')}</Button>
        </Link>
      </div>
    )
  }

  return (
    <>
      <div className="mb-4">
        <Link to="/plans" className="text-sm text-accent hover:underline">
          ← {t('plans.back_to_store')}
        </Link>
      </div>

      <div className="overflow-hidden rounded-2xl border border-border bg-bg-secondary mb-8">
        <PackageCover pkg={pkg} />
        <div className="p-6 space-y-3">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-wide text-text-tertiary">
                {t('plans.official')}
              </p>
              <h1 className="text-2xl font-bold text-text-primary mt-1">{pkg.name}</h1>
              {pkg.description && (
                <p className="text-sm text-text-tertiary mt-2 max-w-3xl">{pkg.description}</p>
              )}
            </div>
            <div className="text-right">
              <p className="text-xs text-text-tertiary">{t('plans.from_price')}</p>
              <p className="text-2xl font-bold text-emerald-400 tabular-nums">
                {formatCost(pkg.min_price ?? pkg.plans[0]?.price ?? 0)}
              </p>
            </div>
          </div>
          <div className="flex flex-wrap gap-3 text-sm text-text-tertiary">
            <span>{t('plans.models_count', { count: pkg.model_count ?? 0 })}</span>
            <span>•</span>
            <span>
              {(pkg.subscriber_count ?? 0) > 0
                ? t('plans.subscribers_count', { count: pkg.subscriber_count ?? 0 })
                : t('plans.no_subscribers')}
            </span>
          </div>
        </div>
      </div>

      <PageHeader title={t('plans.choose_plan')} description={t('plans.choose_plan_desc')} />

      <div className={cn('grid gap-5', pkg.plans.length > 1 ? 'lg:grid-cols-2' : 'max-w-xl')}>
        {pkg.plans.map((plan) => (
          <PlanOptionCard
            key={plan.id}
            plan={plan}
            onBuy={() => handleBuy(plan)}
            buying={buyingPlanId === plan.id}
          />
        ))}
      </div>

      <SePayPaymentDialog
        open={sepayOpen}
        onClose={() => setSepayOpen(false)}
        order={sepayOrder}
        onSuccess={onPaymentSuccess}
      />
    </>
  )
}