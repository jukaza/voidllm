import { useState, useMemo } from 'react'
import { Navigate } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { StatCard } from '../components/ui/StatCard'
import { useMe } from '../hooks/useMe'
import { useLicense, useActivateLicense } from '../hooks/useLicense'
import type { LicenseInfo } from '../hooks/useLicense'
import { useToast } from '../hooks/useToast'
import { formatDate } from '../lib/utils'
import { useTranslation } from '../lib/i18n'

// Global arrays removed and redefined dynamically inside components to support localization.

// ---------------------------------------------------------------------------
// Icons
// ---------------------------------------------------------------------------

function IconTag() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z" />
      <line x1="7" y1="7" x2="7.01" y2="7" />
    </svg>
  )
}

function IconCheckCircle() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
      <polyline points="22 4 12 14.01 9 11.01" />
    </svg>
  )
}

function IconCalendar() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
      <line x1="16" y1="2" x2="16" y2="6" />
      <line x1="8" y1="2" x2="8" y2="6" />
      <line x1="3" y1="10" x2="21" y2="10" />
    </svg>
  )
}

function IconCheck() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polyline points="20 6 9 17 4 12" />
    </svg>
  )
}

function IconX() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

function planBadgeVariant(plan: string): 'muted' | 'default' | 'success' {
  if (plan === 'enterprise') return 'success'
  if (plan === 'pro') return 'default'
  return 'muted'
}

function planLabel(plan: string, t: any): string {
  if (plan === 'enterprise') return 'Enterprise'
  if (plan === 'pro') return 'Pro'
  return t('license.current_plan') === 'Gói hiện tại' ? 'Bản Community' : 'Community'
}

const statusBadgeVariant = (status: string): 'success' | 'error' | 'muted' => {
  if (status === 'active') return 'success'
  if (status === 'expired') return 'error'
  return 'muted'
}

function statusLabel(status: string, t: any): string {
  if (status === 'active') return t('license.current_plan') === 'Gói hiện tại' ? 'Đang hoạt động' : 'Active'
  if (status === 'expired') return t('license.current_plan') === 'Gói hiện tại' ? 'Hết hạn' : 'Expired'
  return t('license.current_plan') === 'Gói hiện tại' ? 'Bản Community' : 'Community'
}

function limitLabel(n: number, t: any): string {
  return n < 0 ? (t('license.never') === 'Không bao giờ' ? 'Không giới hạn' : 'Unlimited') : String(n)
}

function FeatureRow({ label, enabled }: { label: string; enabled: boolean }) {
  return (
    <div className={['flex items-center gap-2 text-sm', enabled ? 'text-text-secondary' : 'text-text-tertiary'].join(' ')}>
      <span className={enabled ? 'text-success' : 'text-text-tertiary'} aria-hidden="true">
        {enabled ? <IconCheck /> : <IconX />}
      </span>
      {label}
    </div>
  )
}

// ---------------------------------------------------------------------------
// CurrentPlanPanel
// ---------------------------------------------------------------------------

interface CurrentPlanPanelProps {
  license: LicenseInfo
  licenseKey: string
  onLicenseKeyChange: (v: string) => void
  onActivate: () => void
  activating: boolean
  activateError: string | null
}

function CurrentPlanPanel({
  license,
  licenseKey,
  onLicenseKeyChange,
  onActivate,
  activating,
  activateError,
}: CurrentPlanPanelProps) {
  const { t, language } = useTranslation()
  const isCommunity = license.edition === 'community'

  const communityCapabilities = useMemo(() => {
    return language === 'vi' ? [
      'Không giới hạn người dùng',
      'Proxy đầy đủ cho tất cả nhà cung cấp',
      'Theo dõi & phân tích lượng sử dụng',
      'Kiểm soát quyền truy cập mô hình',
      'Phân quyền RBAC (4 vai trò tích hợp)',
      'Hệ thống mời thành viên',
      'Khu vực Playground',
      'Tài liệu API',
    ] : [
      'Unlimited users',
      'Full proxy with all providers',
      'Usage tracking + analytics',
      'Model access control',
      'RBAC (4 built-in roles)',
      'Invite system',
      'Playground',
      'API documentation',
    ]
  }, [language])

  const FEATURE_LABELS: Record<string, string> = useMemo(() => {
    return {
      multi_org:        language === 'vi' ? 'Quản lý đa tổ chức' : 'Multi-org management',
      cost_reports:     language === 'vi' ? 'Báo cáo chi phí & cảnh báo ngân sách' : 'Cost reports + budget alerts',
      audit_logs:       language === 'vi' ? 'Nhật ký kiểm toán (Audit logs)' : 'Audit logs',
      sso_oidc:         language === 'vi' ? 'Tích hợp SSO / OIDC' : 'SSO / OIDC integration',
      custom_roles:     language === 'vi' ? 'Vai trò tùy chỉnh' : 'Custom roles',
      otel_tracing:     language === 'vi' ? 'Theo dõi OpenTelemetry' : 'OpenTelemetry tracing',
      fallback_chains:  language === 'vi' ? 'Chuỗi dự phòng mô hình' : 'Model fallback chains',
    }
  }, [language])

  return (
    <div className="rounded-lg border border-border bg-bg-secondary p-6">
      {/* Plan heading */}
      <div className="flex items-center gap-3 mb-1">
        <h2 className="text-lg font-semibold text-text-primary">{t('license.current_plan')}</h2>
        <Badge variant={planBadgeVariant(license.edition)}>{planLabel(license.edition, t)}</Badge>
        <Badge variant={statusBadgeVariant(license.valid ? 'active' : 'expired')}>{statusLabel(license.valid ? 'active' : 'expired', t)}</Badge>
      </div>

      {/* Expiry */}
      {license.expires_at != null && (
        <p className="text-xs text-text-tertiary mb-4">
          {license.valid ? (language === 'vi' ? 'Hết hạn' : 'Expires') : (language === 'vi' ? 'Đã hết hạn' : 'Expired')} {formatDate(license.expires_at)}
        </p>
      )}

      {/* Limits */}
      <div className="flex gap-4 mb-6 mt-4">
        <div className="flex-1 rounded-md bg-bg-tertiary px-3 py-2">
          <div className="text-xs text-text-tertiary mb-0.5">{t('license.max_orgs')}</div>
          <div className="text-sm font-semibold text-text-primary">{limitLabel(license.max_orgs, t)}</div>
        </div>
        <div className="flex-1 rounded-md bg-bg-tertiary px-3 py-2">
          <div className="text-xs text-text-tertiary mb-0.5">{t('license.max_teams')}</div>
          <div className="text-sm font-semibold text-text-primary">{limitLabel(license.max_teams, t)}</div>
        </div>
      </div>

      {/* Community capabilities (always enabled) */}
      <div className="space-y-2 mb-4">
        {communityCapabilities.map((f: string) => (
          <div key={f} className="flex items-center gap-2 text-sm text-text-secondary">
            <span className="text-success" aria-hidden="true"><IconCheck /></span>
            {f}
          </div>
        ))}
      </div>

      {/* Licensed feature flags */}
      {Object.keys(FEATURE_LABELS).length > 0 && (
        <div className="space-y-2 mb-6 border-t border-border pt-4">
          {Object.entries(FEATURE_LABELS).map(([key, label]) => (
            <FeatureRow
              key={key}
              label={label}
              enabled={license.features.includes(key)}
            />
          ))}
        </div>
      )}

      {/* Customer ID (admin-visible) */}
      {license.customer_id != null && (
        <p className="text-xs text-text-tertiary mb-4">
          {t('license.customer_id')}: <span className="font-mono text-text-secondary">{license.customer_id}</span>
        </p>
      )}

      {/* License key input */}
      <div className="border-t border-border pt-4">
        {!isCommunity && (
          <p className="text-xs text-text-tertiary mb-3">
            Bản quyền hoạt động: <span className="font-mono text-success">{planLabel(license.edition, t)}</span>
          </p>
        )}
        <Input
          label={isCommunity ? t('license.key') : t('license.key_replace')}
          type="password"
          value={licenseKey}
          onChange={(e) => onLicenseKeyChange(e.target.value)}
          placeholder="eyJhbGciOiJFZERTQSJ9..."
          description={isCommunity
            ? t('license.key_desc_community')
            : t('license.key_desc_paid')}
          disabled={activating}
          error={activateError ?? undefined}
          autoComplete="off"
        />
        <Button
          variant="primary"
          size="sm"
          className="mt-2"
          disabled={!licenseKey.trim()}
          loading={activating}
          onClick={onActivate}
        >
          {t('license.activate')}
        </Button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// LoadingSkeleton
// ---------------------------------------------------------------------------

function LoadingSkeleton() {
  return (
    <div className="rounded-lg border border-border bg-bg-secondary p-6 space-y-4 animate-pulse">
      <div className="flex items-center gap-3">
        <div className="h-5 w-28 rounded bg-bg-tertiary" />
        <div className="h-5 w-20 rounded bg-bg-tertiary" />
      </div>
      <div className="flex gap-4">
        <div className="flex-1 h-12 rounded bg-bg-tertiary" />
        <div className="flex-1 h-12 rounded bg-bg-tertiary" />
      </div>
      {[...Array(6)].map((_, i) => (
        <div key={i} className="h-4 rounded bg-bg-tertiary" style={{ width: `${70 + (i % 3) * 10}%` }} />
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// LicensePage
// ---------------------------------------------------------------------------

export default function LicensePage() {
  const { data: me } = useMe()
  const { data: license, isLoading } = useLicense()
  const { t, language } = useTranslation()
  const [licenseKey, setLicenseKey] = useState('')
  const [activateError, setActivateError] = useState<string | null>(null)
  const activateLicense = useActivateLicense()
  const { toast } = useToast()

  const proFeatures = useMemo(() => {
    return language === 'vi' ? [
      'Quản lý đa tổ chức',
      'Báo cáo chi phí & cảnh báo ngân sách',
      'Xuất dữ liệu sử dụng (CSV/JSON)',
      'Phân tích chéo tổ chức',
      'Lưu trữ dữ liệu không giới hạn',
      'Hỗ trợ email ưu tiên (48h)',
    ] : [
      'Multi-org management',
      'Cost reports + budget alerts',
      'Usage export (CSV/JSON)',
      'Cross-org analytics',
      'Unlimited data retention',
      'Priority email support (48h)',
    ]
  }, [language])

  const enterpriseFeatures = useMemo(() => {
    return language === 'vi' ? [
      'Tích hợp SSO / OIDC',
      'Nhật ký kiểm toán (Audit logs)',
      'Vai trò tùy chỉnh',
      'Theo dõi OpenTelemetry',
      'Chuỗi dự phòng mô hình',
      'Giới hạn tốc độ phân tán (Redis)',
      'Lưu trữ dữ liệu không giới hạn',
      'Hỗ trợ qua kênh Slack riêng (24h)',
    ] : [
      'SSO / OIDC integration',
      'Audit logs',
      'Custom roles',
      'OpenTelemetry tracing',
      'Model fallback chains',
      'Distributed rate limiting (Redis)',
      'Unlimited data retention',
      'Dedicated Slack support (24h)',
    ]
  }, [language])

  if (me && !me.is_system_admin) {
    return <Navigate to="/" replace />
  }

  const plan = license?.edition ?? 'community'
  const isCommunity = plan === 'community'
  const isPro = plan === 'pro'

  function handleActivate() {
    setActivateError(null)
    activateLicense.mutate(licenseKey.trim(), {
      onSuccess: () => {
        setLicenseKey('')
        toast({
          variant: 'success',
          message: t('license.restart_notice'),
        })
      },
      onError: (err) => {
        const msg = err instanceof Error ? err.message : t('license.fail_notice')
        setActivateError(msg)
        toast({ variant: 'error', message: msg })
      },
    })
  }

  return (
    <>
      <PageHeader
        title={t('license.title')}
        description={t('license.desc')}
      />

      {/* Stat row */}
      {license != null && (
        <div className="grid grid-cols-3 gap-4 mb-6">
          <StatCard
            label="Gói cước"
            value={planLabel(license.edition, t)}
            icon={<IconTag />}
            iconColor="purple"
          />
          <StatCard
            label="Trạng thái"
            value={statusLabel(license.valid ? 'active' : 'expired', t)}
            icon={<IconCheckCircle />}
            iconColor={license.valid ? 'green' : 'red'}
          />
          <StatCard
            label="Hết hạn"
            value={license.expires_at != null ? formatDate(license.expires_at) : (language === 'vi' ? 'Không bao giờ' : 'Never')}
            icon={<IconCalendar />}
            iconColor="blue"
          />
        </div>
      )}

      <div className="flex gap-6 items-start">
        {/* Left panel — Current Plan (40%) */}
        <div className="w-[40%] shrink-0">
          {isLoading || license == null ? (
            <LoadingSkeleton />
          ) : (
            <CurrentPlanPanel
              license={license}
              licenseKey={licenseKey}
              onLicenseKeyChange={setLicenseKey}
              onActivate={handleActivate}
              activating={activateLicense.isPending}
              activateError={activateError}
            />
          )}
        </div>

        {/* Right panel — Upgrade CTAs (60%) — shown only for non-enterprise plans */}
        {!(!isCommunity && !isPro) && (
          <div className="flex-1 space-y-4">
            {/* Pro card — shown for community users */}
            {isCommunity && (
              <div className="rounded-lg border border-accent/30 bg-bg-secondary p-6">
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-lg font-semibold text-text-primary">Pro</h3>
                  <span className="text-sm text-accent font-semibold">$299/tháng</span>
                </div>
                <p className="text-xs text-text-tertiary mb-4">{t('license.pro.desc')}</p>

                <div className="space-y-2 mb-4">
                  {proFeatures.map((f: string) => (
                    <div key={f} className="flex items-center gap-2 text-sm text-text-secondary">
                      <span className="text-success" aria-hidden="true"><IconCheck /></span>
                      {f}
                    </div>
                  ))}
                </div>

                <Button
                  variant="primary"
                  onClick={() => window.open('https://voidllm.ai/pricing', '_blank')}
                >
                  {t('license.pro.upgrade')}
                </Button>
              </div>
            )}

            {/* Enterprise card — shown for community and pro users */}
            <div className="rounded-lg border border-border bg-bg-secondary p-6">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-lg font-semibold text-text-primary">Enterprise</h3>
                <span className="text-sm text-text-secondary font-semibold">$799/tháng</span>
              </div>
              <p className="text-xs text-text-tertiary mb-4">
                {isCommunity
                  ? t('license.ent.desc_community')
                  : t('license.ent.desc_pro')}
              </p>

              <div className="space-y-2 mb-4">
                {enterpriseFeatures.map((f: string) => (
                  <div key={f} className="flex items-center gap-2 text-sm text-text-secondary">
                    <span className="text-success" aria-hidden="true"><IconCheck /></span>
                    {f}
                  </div>
                ))}
              </div>

              <Button
                variant="secondary"
                onClick={() => window.open('https://voidllm.ai/pricing', '_blank')}
              >
                {t('license.ent.contact')}
              </Button>
            </div>
          </div>
        )}
      </div>
    </>
  )
}
