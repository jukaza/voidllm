import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { SePayPaymentDialog } from '../components/wallet/SePayPaymentDialog'
import { useTranslation } from '../lib/i18n'
import { useToast } from '../hooks/useToast'
import { formatCost, formatNumber, cn } from '../lib/utils'
import {
  usePublicSubscriptionCatalog,
  useCreateSubscriptionOrder,
  useMySubscriptions,
  type PublicSubscriptionPackage,
  type PublicSubscriptionPlan,
  type SepayOrderLike,
} from '../hooks/useSubscriptions'
import { useQueryClient } from '@tanstack/react-query'
import type { SepayOrder } from '../hooks/usePaymentSettings'

const presetGradients: Record<string, string> = {
  aurora: 'from-violet-600/80 via-fuchsia-500/60 to-cyan-400/70',
  sunset: 'from-orange-500/80 via-rose-500/70 to-amber-400/60',
  ocean: 'from-blue-700/80 via-cyan-500/60 to-teal-400/70',
  ember: 'from-red-700/80 via-orange-600/70 to-yellow-500/50',
  violet: 'from-indigo-700/80 via-violet-600/70 to-purple-400/60',
}

function PackageHero({ pkg }: { pkg: PublicSubscriptionPackage }) {
  const gradient =
    pkg.cover_type === 'default'
      ? presetGradients[pkg.cover_value] ?? presetGradients.aurora
      : null
  if (pkg.cover_type === 'upload' || pkg.cover_type === 'url') {
    const src = pkg.cover_value
    return (
      <div className="relative h-44 overflow-hidden rounded-t-2xl">
        {src ? (
          <img src={src} alt="" className="h-full w-full object-cover" />
        ) : (
          <div className={cn('h-full w-full bg-gradient-to-br', presetGradients.aurora)} />
        )}
        <div className="absolute inset-0 bg-gradient-to-t from-bg-secondary via-bg-secondary/20 to-transparent" />
      </div>
    )
  }
  return (
    <div className={cn('relative h-44 overflow-hidden rounded-t-2xl bg-gradient-to-br', gradient)}>
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,rgba(255,255,255,0.12),transparent_60%)]" />
      <div className="absolute inset-0 bg-gradient-to-t from-bg-secondary via-transparent to-transparent" />
    </div>
  )
}

function PlanCard({
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

  const quota: string[] = []
  if (plan.daily_token_limit > 0) quota.push(`${formatNumber(plan.daily_token_limit)} tok/d`)
  if (plan.monthly_token_limit > 0) quota.push(`${formatNumber(plan.monthly_token_limit)} tok/mo`)
  if (plan.daily_request_limit > 0) quota.push(`${formatNumber(plan.daily_request_limit)} req/d`)

  return (
    <div className="rounded-xl border border-border/80 bg-bg-primary/40 p-4 flex flex-col gap-3 hover:border-accent/30 transition-colors">
      <div className="flex items-start justify-between gap-2">
        <div>
          <h4 className="font-semibold text-text-primary">{plan.name}</h4>
          {plan.description && (
            <p className="text-xs text-text-tertiary mt-1 line-clamp-2">{plan.description}</p>
          )}
        </div>
        <span className="text-lg font-bold text-accent tabular-nums shrink-0">
          {formatCost(plan.price)}
        </span>
      </div>

      <div className="flex flex-wrap gap-1.5">
        <Badge variant="muted">{plan.validity_days}d</Badge>
        {plan.allowed_models?.length > 0 && (
          <Badge variant="info">{plan.allowed_models.length} models</Badge>
        )}
        {plan.slots_remaining !== undefined && (
          <Badge variant={soldOut ? 'warning' : 'default'}>
            {soldOut ? t('plans.sold_out') : `${plan.slots_remaining} ${t('plans.slots_left')}`}
          </Badge>
        )}
      </div>

      {quota.length > 0 && (
        <p className="text-[11px] text-text-tertiary">{quota.join(' · ')}</p>
      )}

      {plan.allowed_models?.length > 0 && (
        <p className="text-[10px] text-text-tertiary line-clamp-2 font-mono">
          {plan.allowed_models.slice(0, 4).join(', ')}
          {plan.allowed_models.length > 4 ? '…' : ''}
        </p>
      )}

      <Button
        className="mt-auto w-full"
        onClick={onBuy}
        loading={buying}
        disabled={soldOut}
      >
        {soldOut ? t('plans.sold_out') : t('plans.buy')}
      </Button>
    </div>
  )
}

export default function PlansPage() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const queryClient = useQueryClient()
  const { data, isLoading } = usePublicSubscriptionCatalog()
  const { data: mySubs } = useMySubscriptions()
  const createOrder = useCreateSubscriptionOrder()

  const [sepayOpen, setSepayOpen] = useState(false)
  const [sepayOrder, setSepayOrder] = useState<SepayOrder | null>(null)
  const [buyingPlanId, setBuyingPlanId] = useState<string | null>(null)

  const packages = data?.data ?? []
  const activeCount = useMemo(
    () => (mySubs?.data ?? []).filter((s) => s.status === 'active').length,
    [mySubs?.data],
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

  function onPaymentSuccess() {
    queryClient.invalidateQueries({ queryKey: ['my-subscriptions'] })
    queryClient.invalidateQueries({ queryKey: ['public-subscription-packages'] })
    queryClient.invalidateQueries({ queryKey: ['my-wallet'] })
    toast({ variant: 'success', message: t('plans.purchase_success') })
  }

  return (
    <>
      <PageHeader
        title={t('plans.title')}
        description={t('plans.desc')}
        actions={
          activeCount > 0 ? (
            <Link to="/keys">
              <Button variant="secondary">{t('plans.attach_key')}</Button>
            </Link>
          ) : undefined
        }
      />

      {activeCount > 0 && (
        <div className="mb-6 rounded-xl border border-accent/25 bg-accent/5 px-4 py-3 text-sm text-text-secondary">
          {t('plans.active_count', { count: activeCount })}{' '}
          <Link to="/keys" className="text-accent hover:underline">
            {t('plans.attach_key_link')}
          </Link>
        </div>
      )}

      {isLoading ? (
        <div className="rounded-2xl border border-border bg-bg-secondary p-20 text-center text-text-tertiary">
          …
        </div>
      ) : packages.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-border p-20 text-center">
          <p className="text-text-secondary">{t('plans.empty')}</p>
        </div>
      ) : (
        <div className="space-y-10">
          {packages.map((pkg) => (
            <section key={pkg.id} className="rounded-2xl border border-border bg-bg-secondary overflow-hidden shadow-sm">
              <PackageHero pkg={pkg} />
              <div className="p-6 space-y-5">
                <div>
                  <h2 className="text-xl font-bold text-text-primary">{pkg.name}</h2>
                  {pkg.description && (
                    <p className="text-sm text-text-tertiary mt-2 max-w-2xl">{pkg.description}</p>
                  )}
                </div>
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  {pkg.plans.map((plan) => (
                    <PlanCard
                      key={plan.id}
                      plan={plan}
                      onBuy={() => handleBuy(plan)}
                      buying={buyingPlanId === plan.id}
                    />
                  ))}
                </div>
              </div>
            </section>
          ))}
        </div>
      )}

      <p className="mt-8 text-center text-xs text-text-tertiary max-w-xl mx-auto">
        {t('plans.footer_hint')}
      </p>

      <SePayPaymentDialog
        open={sepayOpen}
        onClose={() => setSepayOpen(false)}
        order={sepayOrder}
        onSuccess={onPaymentSuccess}
      />
    </>
  )
}