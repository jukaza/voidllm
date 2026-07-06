import { useMemo } from 'react'
// intersectModels reserved for future plan-model overlap warnings
import { Badge } from '../ui/Badge'
import { Button } from '../ui/Button'
import { Banner } from '../ui/Banner'
import { useTranslation } from '../../lib/i18n'
import { useToast } from '../../hooks/useToast'
import {
  useMySubscriptions,
  useKeySubscriptionBinding,
  useBindKeySubscription,
  useReleaseKeySubscription,
} from '../../hooks/useSubscriptions'
interface KeySubscriptionFieldProps {
  keyId: string
  modelLimitsEnabled: boolean
  modelLimits: string[]
  disabled?: boolean
}

export function KeySubscriptionField({
  keyId,
  modelLimitsEnabled,
  modelLimits,
  disabled,
}: KeySubscriptionFieldProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data: subsData } = useMySubscriptions()
  const { data: bindingData, isLoading } = useKeySubscriptionBinding(keyId)
  const bind = useBindKeySubscription()
  const release = useReleaseKeySubscription()

  const binding = bindingData?.data ?? null
  const subs = subsData?.data ?? []

  const activeSubOptions = useMemo(
    () => subs.filter((s) => s.status === 'active'),
    [subs],
  )

  const selectedSub = binding
    ? subs.find((s) => s.id === binding.user_subscription_id)
    : null

  function handleBind(subscriptionId: string) {
    bind.mutate(
      { keyId, userSubscriptionId: subscriptionId },
      {
        onSuccess: () => toast({ variant: 'success', message: t('keys.subscription.bound') }),
        onError: (e) => toast({ variant: 'error', message: e.message }),
      },
    )
  }

  function handleRelease() {
    release.mutate(keyId, {
      onSuccess: () => toast({ variant: 'success', message: t('keys.subscription.released') }),
      onError: (e) => toast({ variant: 'error', message: e.message }),
    })
  }

  return (
    <div className="rounded-lg border border-border/80 bg-bg-primary/50 p-4 space-y-3">
      <div className="flex items-center justify-between gap-2">
        <div>
          <p className="text-sm font-medium text-text-primary">{t('keys.subscription.title')}</p>
          <p className="text-xs text-text-tertiary mt-0.5">{t('keys.subscription.hint')}</p>
        </div>
        {binding && <Badge variant="info">{t('keys.subscription.active')}</Badge>}
      </div>

      {isLoading ? (
        <p className="text-xs text-text-tertiary">…</p>
      ) : binding ? (
        <div className="space-y-3">
          <div className="rounded-md border border-accent/20 bg-accent/5 px-3 py-2.5 text-sm">
            <p className="font-medium text-text-primary">
              {binding.package_name || selectedSub?.package_name || '—'}
              <span className="text-text-tertiary font-normal"> · </span>
              {binding.plan_name || selectedSub?.plan_name}
            </p>
            {binding.expires_at && (
              <p className="text-xs text-text-tertiary mt-1">
                {t('keys.subscription.expires')}: {new Date(binding.expires_at).toLocaleDateString()}
              </p>
            )}
          </div>
          <Button
            variant="secondary"
            size="sm"
            onClick={handleRelease}
            loading={release.isPending}
            disabled={disabled}
          >
            {t('keys.subscription.release')}
          </Button>
        </div>
      ) : activeSubOptions.length === 0 ? (
        <Banner variant="info" title={t('keys.subscription.title')} description={t('keys.subscription.none_available')} />
      ) : (
        <div className="space-y-2">
          {activeSubOptions.map((sub) => (
            <SubscriptionPickRow
              key={sub.id}
              label={`${sub.package_name ?? ''} · ${sub.plan_name ?? sub.plan_id}`}
              expires={sub.expires_at}
              modelLimitsEnabled={modelLimitsEnabled}
              modelLimits={modelLimits}
              onSelect={() => handleBind(sub.id)}
              loading={bind.isPending}
              disabled={disabled}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function SubscriptionPickRow({
  label,
  expires,
  modelLimitsEnabled,
  modelLimits,
  onSelect,
  loading,
  disabled,
}: {
  label: string
  expires: string
  modelLimitsEnabled: boolean
  modelLimits: string[]
  onSelect: () => void
  loading?: boolean
  disabled?: boolean
}) {
  const { t } = useTranslation()
  // Plan models unknown client-side without extra fetch — warn only on key limits.
  const overlapWarning =
    modelLimitsEnabled &&
    modelLimits.length > 0 &&
    t('keys.subscription.key_limit_warning')

  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2">
      <div className="min-w-0">
        <p className="text-sm text-text-primary truncate">{label}</p>
        <p className="text-[11px] text-text-tertiary">
          {t('keys.subscription.expires')}: {new Date(expires).toLocaleDateString()}
        </p>
        {overlapWarning && (
          <p className="text-[11px] text-warning mt-0.5">{overlapWarning}</p>
        )}
      </div>
      <Button size="sm" onClick={onSelect} loading={loading} disabled={disabled}>
        {t('keys.subscription.attach')}
      </Button>
    </div>
  )
}