import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { Banner } from '../components/ui/Banner'
import { Dialog } from '../components/ui/Dialog'
import { Button } from '../components/ui/Button'
import { Markdown } from '../components/ui/Markdown'
import { useDashboardStats } from '../hooks/useDashboardStats'
import type { BudgetWarning } from '../hooks/useDashboardStats'
import { useUpdateCheck } from '../hooks/useUpdateCheck'
import { useMe } from '../hooks/useMe'
import { useMyWallet } from '../hooks/useWallet'
import { financeRangeISO, useFinanceSummary } from '../hooks/useFinance'
import { useUsageLive } from '../hooks/useUsageLive'
import { formatNumber, formatCost, formatTokens } from '../lib/utils'
import { useTranslation } from '../lib/i18n'
import { useSiteConfig } from '../hooks/useSiteConfig'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function BudgetWarningBanners({ warnings }: { warnings: BudgetWarning[] }) {
  if (warnings.length === 0) return null
  return (
    <div className="space-y-2">
      {warnings.map((w) => (
        <Banner
          key={`${w.scope}-${w.window}`}
          variant={w.percent_used > 0.9 ? 'error' : 'warning'}
          title={`${w.window === 'daily' ? 'Daily' : 'Monthly'} token budget: ${formatNumber(w.usage)} / ${formatNumber(w.limit)} (${Math.round(w.percent_used * 100)}% used)`}
        />
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Icons
// ---------------------------------------------------------------------------

const iconProps = {
  width: 16,
  height: 16,
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 2,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
}

function IconActivity() {
  return (
    <svg {...iconProps}>
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
    </svg>
  )
}

function IconZap() {
  return (
    <svg {...iconProps}>
      <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
    </svg>
  )
}

function IconKey() {
  return (
    <svg {...iconProps}>
      <circle cx="7.5" cy="15.5" r="5.5" />
      <path d="M21 2l-9.6 9.6" />
      <path d="M15.5 7.5l3 3L22 7l-3-3" />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function MetricPill({
  icon,
  label,
  value,
  loading,
}: {
  icon: React.ReactNode
  label: string
  value: string
  loading?: boolean
}) {
  return (
    <div className="flex items-center gap-3 rounded-xl border border-white/5 bg-bg-primary/60 px-4 py-3">
      <span className="rounded-lg bg-accent/10 p-2 text-accent">{icon}</span>
      <div className="min-w-0">
        <div className="text-lg font-semibold tabular-nums text-text-primary">
          {loading ? '...' : value}
        </div>
        <div className="truncate text-xs text-text-tertiary">{label}</div>
      </div>
    </div>
  )
}

function HeroLinkButton({
  to,
  children,
  primary,
}: {
  to: string
  children: React.ReactNode
  primary?: boolean
}) {
  return (
    <Link
      to={to}
      className={
        primary
          ? 'inline-flex items-center justify-center rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-white no-underline transition-all duration-200 hover:bg-accent/90 hover:shadow-[0_0_20px_var(--accent-glow)]'
          : 'inline-flex items-center justify-center rounded-lg border border-border bg-bg-primary/80 px-4 py-2 text-sm font-medium text-text-secondary no-underline transition-colors hover:border-accent/30 hover:text-text-primary'
      }
    >
      {children}
    </Link>
  )
}

// ---------------------------------------------------------------------------
// DashboardPage
// ---------------------------------------------------------------------------

export default function DashboardPage() {
  const { data: me } = useMe()
  const { data: stats, isLoading: statsLoading } = useDashboardStats()
  const { data: wallet, isLoading: walletLoading } = useMyWallet()
  const { data: updateInfo } = useUpdateCheck()
  const { data: site } = useSiteConfig()
  const { t } = useTranslation()
  const systemName = site?.system_name ?? 'VoidLLM'
  const [showUpdateDialog, setShowUpdateDialog] = useState(false)

  const isAdmin = me?.is_system_admin ?? false
  const { live, connected } = useUsageLive()
  const financeRange = useMemo(() => financeRangeISO(7), [])
  const { data: financeSummary } = useFinanceSummary(financeRange.from, financeRange.to)

  const availableVersion = updateInfo?.available_version
  const [manualDismiss, setManualDismiss] = useState(false)
  const updateDismissed =
    manualDismiss ||
    !availableVersion ||
    localStorage.getItem(`update_dismissed_${availableVersion}`) === 'true'

  function dismissUpdate() {
    if (updateInfo?.available_version) {
      localStorage.setItem(`update_dismissed_${updateInfo.available_version}`, 'true')
    }
    setManualDismiss(true)
    setShowUpdateDialog(false)
  }

  const showGettingStarted =
    !statsLoading && ((stats?.active_keys ?? 0) === 0 || (stats?.requests_24h ?? 0) === 0)
  const lowBalance = wallet != null && wallet.balance <= 0
  const pendingTopupCount = financeSummary?.totals.pending_topup_count ?? 0
  const displayName = me?.display_name || me?.email?.split('@')[0] || ''

  return (
    <>
      <PageHeader
        title={displayName ? t('dashboard.welcome', { name: displayName }) : t('sidebar.dashboard')}
        description={t('dashboard.overview_desc')}
      />

      <div className="space-y-6">
        {(stats?.budget_warnings?.length ?? 0) > 0 && (
          <BudgetWarningBanners warnings={stats?.budget_warnings ?? []} />
        )}

        {lowBalance && (
          <Link to="/wallet" className="block no-underline">
            <Banner variant="warning" title={t('dashboard.low_balance')} />
          </Link>
        )}

        {isAdmin && pendingTopupCount > 0 && (
          <Link to="/finance/topups?status=pending" className="block no-underline">
            <Banner
              variant="info"
              title={t('dashboard.pending_topups', { count: pendingTopupCount })}
            />
          </Link>
        )}

        {updateInfo?.needs_update && !updateDismissed && (
          <div onClick={() => setShowUpdateDialog(true)} className="cursor-pointer">
            <Banner
              variant="info"
              title={`${systemName} ${updateInfo.available_version} is available (current: ${updateInfo.current_version})`}
              onDismiss={(e) => {
                e.stopPropagation()
                dismissUpdate()
              }}
            />
          </div>
        )}

        {/* Hero + metrics */}
        <div className="relative overflow-hidden rounded-2xl border border-accent/15 bg-bg-secondary">
          <div
            className="pointer-events-none absolute -right-16 -top-16 h-64 w-64 rounded-full opacity-40"
            style={{ background: 'radial-gradient(circle, rgba(139,92,246,0.25) 0%, transparent 70%)' }}
            aria-hidden="true"
          />

          <div className="relative p-6 lg:p-8">
            <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
              <div>
                <p className="text-xs font-medium uppercase tracking-wider text-accent">
                  {t('dashboard.period_24h')}
                </p>
                <p className="mt-3 text-4xl font-bold tabular-nums tracking-tight text-text-primary lg:text-5xl">
                  {walletLoading ? '...' : formatCost(wallet?.balance ?? 0)}
                </p>
                <p className="mt-1 text-sm text-text-tertiary">{t('dashboard.balance')}</p>
              </div>

              <div className="flex flex-wrap gap-2 lg:justify-end">
                <HeroLinkButton to="/playground" primary>
                  {t('dashboard.action_playground')}
                </HeroLinkButton>
                <HeroLinkButton to="/keys">
                  {t('dashboard.action_keys')}
                </HeroLinkButton>
                {lowBalance && (
                  <HeroLinkButton to="/wallet">
                    {t('dashboard.hero_cta_topup')}
                  </HeroLinkButton>
                )}
              </div>
            </div>

            {isAdmin && connected && live && (
              <Link
                to="/analytics"
                className="mt-5 inline-flex flex-wrap items-center gap-3 rounded-lg border border-emerald-500/20 bg-emerald-500/5 px-3 py-2 text-xs no-underline transition-colors hover:border-emerald-500/35"
              >
                <span className="inline-flex items-center gap-1.5 font-medium text-emerald-400">
                  <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-400" />
                  {t('dashboard.live_status')}
                </span>
                <span className="text-text-secondary">{t('analytics.rpm')}: {formatNumber(live.rpm)}</span>
                <span className="text-text-secondary">{t('analytics.tpm')}: {formatNumber(live.tpm)}</span>
                <span className="text-text-secondary">{t('analytics.active')}: {formatNumber(live.active_count)}</span>
              </Link>
            )}

            <div className="mt-6 grid grid-cols-1 gap-3 sm:grid-cols-3">
              <MetricPill
                icon={<IconKey />}
                label={t('dashboard.keys')}
                value={formatNumber(stats?.active_keys ?? 0)}
                loading={statsLoading}
              />
              <MetricPill
                icon={<IconActivity />}
                label={t('dashboard.requests')}
                value={formatNumber(stats?.requests_24h ?? 0)}
                loading={statsLoading}
              />
              <MetricPill
                icon={<IconZap />}
                label={t('dashboard.tokens')}
                value={formatTokens(stats?.tokens_24h ?? 0)}
                loading={statsLoading}
              />
            </div>
          </div>
        </div>

        {showGettingStarted && (
          <div className="rounded-2xl border border-accent/20 bg-accent/5 p-5">
            <h2 className="text-sm font-semibold text-text-primary">{t('dashboard.getting_started')}</h2>
            <div className="mt-4 grid gap-3 sm:grid-cols-3">
              {[
                { step: 1, label: t('dashboard.step_create_key'), path: '/keys' },
                { step: 2, label: t('dashboard.step_test_playground'), path: '/playground' },
                { step: 3, label: t('dashboard.step_view_analytics'), path: '/analytics' },
              ].map((item) => (
                <Link
                  key={item.step}
                  to={item.path}
                  className="flex items-center gap-3 rounded-lg border border-border/60 bg-bg-secondary/80 px-3 py-2.5 no-underline transition-colors hover:border-accent/30"
                >
                  <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-accent/15 text-[11px] font-bold text-accent">
                    {item.step}
                  </span>
                  <span className="text-sm text-text-primary">{item.label}</span>
                </Link>
              ))}
            </div>
          </div>
        )}
      </div>

      {updateInfo != null && (
        <Dialog
          open={showUpdateDialog}
          onClose={() => setShowUpdateDialog(false)}
          title={`${systemName} ${updateInfo.available_version ?? ''}`}
          footer={
            <div className="flex gap-3">
              {updateInfo.release_url != null && (
                <a
                  href={updateInfo.release_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-bold text-white transition-all duration-200 hover:bg-accent/90"
                >
                  View on GitHub
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                  </svg>
                </a>
              )}
              <Button variant="secondary" onClick={dismissUpdate}>
                Dismiss
              </Button>
            </div>
          }
        >
          <p className="mb-4 text-xs text-text-tertiary">
            You are running {updateInfo.current_version}
          </p>
          {updateInfo.release_notes != null && (
            <div className="text-sm text-text-secondary">
              <Markdown>{updateInfo.release_notes}</Markdown>
            </div>
          )}
        </Dialog>
      )}
    </>
  )
}