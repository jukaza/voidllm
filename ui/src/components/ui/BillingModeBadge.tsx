import { useTranslation } from '../../lib/i18n'
import type { BillingMode } from '../../hooks/useModels'
import { cn } from '../../lib/utils'

function TokenIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M4 7h16M4 12h10M4 17h6" />
    </svg>
  )
}

function RequestIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
    </svg>
  )
}

interface BillingModeBadgeProps {
  mode: BillingMode
  className?: string
}

/** Compact billing mode chip for tables and lists. */
export function BillingModeBadge({ mode, className }: BillingModeBadgeProps) {
  const { t } = useTranslation()
  const isToken = mode === 'token'

  return (
    <span
      title={isToken ? t('models.bill_per_token') : t('models.bill_per_request')}
      className={cn(
        'inline-flex max-w-full items-center gap-1 whitespace-nowrap rounded-full border px-2 py-1 text-[11px] font-medium leading-none',
        isToken
          ? 'border-info/25 bg-info/10 text-info'
          : 'border-accent/25 bg-accent/10 text-accent',
        className,
      )}
    >
      {isToken ? (
        <TokenIcon className="h-3 w-3 shrink-0 opacity-80" />
      ) : (
        <RequestIcon className="h-3 w-3 shrink-0 opacity-80" />
      )}
      <span className="truncate">
        {isToken ? t('models.billing_chip_token') : t('models.billing_chip_request')}
      </span>
    </span>
  )
}