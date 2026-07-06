import React, { useMemo, useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { useTranslation } from '../lib/i18n'
import { Table } from '../components/ui/Table'
import type { Column } from '../components/ui/Table'
import { Dialog, ConfirmDialog } from '../components/ui/Dialog'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Textarea } from '../components/ui/Textarea'
import { Toggle } from '../components/ui/Toggle'
import { ExpiryField, defaultExpiryDateInput, dateInputToISO } from '../components/keys/ExpiryField'
import { ModelLimitPicker, type ModelLimitOption } from '../components/keys/ModelLimitPicker'
import { KeySubscriptionField } from '../components/keys/KeySubscriptionField'
import { Banner } from '../components/ui/Banner'
import { KeyHint } from '../components/ui/KeyHint'
import { forgetApiKey, rememberApiKey } from '../lib/apiKeySecrets'
import { KeyCopyButton } from '../components/keys/KeyCopyButton'
import { TimeAgo } from '../components/ui/TimeAgo'
import { CopyButton } from '../components/ui/CopyButton'
import { StatCard } from '../components/ui/StatCard'
import { useMe } from '../hooks/useMe'
import { useModels } from '../hooks/useModels'
import { useMyWallet } from '../hooks/useWallet'
import {
  useAPIKeys,
  useCreateAPIKey,
  useDeleteAPIKey,
  useUpdateAPIKey,
  useRotateAPIKey,
  usePatchAPIKeyStatus,
  normalizeKeyStatus,
} from '../hooks/useAPIKeys'
import type { APIKeyResponse, APIKeyStatus, CreateAPIKeyParams, UpdateAPIKeyParams } from '../hooks/useAPIKeys'
import { useToast } from '../hooks/useToast'
import { formatCost, formatNumber, formatTokens } from '../lib/utils'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const statusBadgeVariant: Record<APIKeyStatus, 'default' | 'info' | 'warning' | 'muted'> = {
  active: 'default',
  disabled: 'muted',
  expired: 'warning',
  quota_exhausted: 'warning',
}

function parseOptionalInt(value: string): number | undefined {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const n = parseInt(trimmed, 10)
  return isNaN(n) ? undefined : n
}

function parseOptionalFloat(value: string): number | undefined {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const n = parseFloat(trimmed)
  return isNaN(n) ? undefined : n
}

function LimitBar({ used, limit, label }: { used: number; limit: number; label: string }) {
  if (limit <= 0) return null
  const pct = Math.min(100, Math.round((used / limit) * 100))
  return (
    <div className="space-y-0.5" title={`${formatNumber(used)} / ${formatNumber(limit)}`}>
      <div className="flex justify-between text-[10px] text-text-tertiary">
        <span>{label}</span>
        <span>{pct}%</span>
      </div>
      <div className="h-1.5 rounded-full bg-bg-tertiary overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${pct >= 90 ? 'bg-warning' : 'bg-accent'}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}

function SpendCapBar({ used, cap }: { used: number; cap: number }) {
  if (cap <= 0) {
    return <span className="text-xs text-text-tertiary">—</span>
  }
  const pct = Math.min(100, Math.round((used / cap) * 100))
  return (
    <div className="min-w-[120px] space-y-1">
      <div className="text-xs text-text-secondary tabular-nums">
        {formatCost(used)} / {formatCost(cap)}
      </div>
      <div className="h-1.5 rounded-full bg-bg-tertiary overflow-hidden">
        <div
          className={`h-full rounded-full ${pct >= 90 ? 'bg-warning' : 'bg-success'}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}

function LimitsCell({ row }: { row: APIKeyResponse }) {
  const { t } = useTranslation()
  const live = row.limits_live
  const hasLimits =
    row.daily_token_limit > 0 ||
    row.monthly_token_limit > 0 ||
    row.requests_per_minute > 0 ||
    row.requests_per_day > 0

  if (!hasLimits) {
    return <span className="text-xs text-text-tertiary">{t('keys.limit.none')}</span>
  }

  return (
    <div className="space-y-1.5 min-w-[140px]">
      {row.daily_token_limit > 0 && (
        <LimitBar
          used={live?.daily_tokens ?? 0}
          limit={row.daily_token_limit}
          label={t('keys.limit.daily_tokens', { limit: formatTokens(row.daily_token_limit) })}
        />
      )}
      {row.monthly_token_limit > 0 && (
        <LimitBar
          used={live?.monthly_tokens ?? 0}
          limit={row.monthly_token_limit}
          label={t('keys.limit.monthly_tokens', { limit: formatTokens(row.monthly_token_limit) })}
        />
      )}
      {row.requests_per_minute > 0 && (
        <LimitBar
          used={live?.requests_per_minute ?? 0}
          limit={row.requests_per_minute}
          label={t('keys.limit.rpm', { limit: row.requests_per_minute })}
        />
      )}
      {row.requests_per_day > 0 && (
        <LimitBar
          used={live?.requests_per_day ?? 0}
          limit={row.requests_per_day}
          label={t('keys.limit.rpd', { limit: row.requests_per_day })}
        />
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Icons
// ---------------------------------------------------------------------------

function IconKey({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
    </svg>
  )
}

function IconCheck({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M20 6L9 17l-5-5" />
    </svg>
  )
}

function IconClock({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="10" />
      <path d="M12 6v6l4 2" />
    </svg>
  )
}

function IconPencil({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
    </svg>
  )
}

function IconRefresh({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M1 4v6h6" />
      <path d="M23 20v-6h-6" />
      <path d="M20.49 9A9 9 0 0 0 5.64 5.64L1 10m22 4l-4.64 4.36A9 9 0 0 1 3.51 15" />
    </svg>
  )
}

function IconTrash({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polyline points="3 6 5 6 21 6" />
      <path d="M19 6l-1 14H6L5 6" />
      <path d="M10 11v6M14 11v6" />
      <path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
    </svg>
  )
}

function IconChart({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M3 3v18h18" />
      <path d="M7 16l4-8 4 5 5-9" />
    </svg>
  )
}

function IconCheckCircle({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
      <polyline points="22 4 12 14.01 9 11.01" />
    </svg>
  )
}

function IconChevronDown({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M6 9l6 6 6-6" />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Advanced limits fields (shared by create/edit)
// ---------------------------------------------------------------------------

interface AdvancedLimitsFieldsProps {
  dailyTokenLimit: string
  setDailyTokenLimit: (v: string) => void
  monthlyTokenLimit: string
  setMonthlyTokenLimit: (v: string) => void
  requestsPerMinute: string
  setRequestsPerMinute: (v: string) => void
  requestsPerDay: string
  setRequestsPerDay: (v: string) => void
  spendCap: string
  setSpendCap: (v: string) => void
  ipWhitelist: string
  setIpWhitelist: (v: string) => void
  ipBlacklist: string
  setIpBlacklist: (v: string) => void
  modelLimitsEnabled: boolean
  setModelLimitsEnabled: (v: boolean) => void
  modelLimits: string[]
  setModelLimits: (v: string[]) => void
  modelOptions: ModelLimitOption[]
  disabled?: boolean
}

function AdvancedLimitsFields({
  dailyTokenLimit,
  setDailyTokenLimit,
  monthlyTokenLimit,
  setMonthlyTokenLimit,
  requestsPerMinute,
  setRequestsPerMinute,
  requestsPerDay,
  setRequestsPerDay,
  spendCap,
  setSpendCap,
  ipWhitelist,
  setIpWhitelist,
  ipBlacklist,
  setIpBlacklist,
  modelLimitsEnabled,
  setModelLimitsEnabled,
  modelLimits,
  setModelLimits,
  modelOptions,
  disabled,
}: AdvancedLimitsFieldsProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <Input
          label={t('keys.create.daily_tokens')}
          type="number"
          value={dailyTokenLimit}
          onChange={(e) => setDailyTokenLimit(e.target.value)}
          placeholder={t('keys.placeholder.unlimited')}
          disabled={disabled}
        />
        <Input
          label={t('keys.create.monthly_tokens')}
          type="number"
          value={monthlyTokenLimit}
          onChange={(e) => setMonthlyTokenLimit(e.target.value)}
          placeholder={t('keys.placeholder.unlimited')}
          disabled={disabled}
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <Input
          label={t('keys.create.rpm')}
          type="number"
          value={requestsPerMinute}
          onChange={(e) => setRequestsPerMinute(e.target.value)}
          placeholder={t('keys.placeholder.unlimited')}
          disabled={disabled}
        />
        <Input
          label={t('keys.create.rpd')}
          type="number"
          value={requestsPerDay}
          onChange={(e) => setRequestsPerDay(e.target.value)}
          placeholder={t('keys.placeholder.unlimited')}
          disabled={disabled}
        />
      </div>
      <Input
        label={t('keys.create.spend_cap')}
        type="number"
        value={spendCap}
        onChange={(e) => setSpendCap(e.target.value)}
        placeholder={t('keys.placeholder.unlimited')}
        disabled={disabled}
      />
      <Textarea
        label={t('keys.create.ip_whitelist')}
        value={ipWhitelist}
        onChange={(e) => setIpWhitelist(e.target.value)}
        placeholder={t('keys.create.ip_hint')}
        rows={3}
        disabled={disabled}
      />
      <Textarea
        label={t('keys.create.ip_blacklist')}
        value={ipBlacklist}
        onChange={(e) => setIpBlacklist(e.target.value)}
        placeholder={t('keys.create.ip_hint')}
        rows={3}
        disabled={disabled}
      />
      <Toggle
        checked={modelLimitsEnabled}
        onChange={setModelLimitsEnabled}
        label={t('keys.create.model_limits_enabled')}
        disabled={disabled}
      />
      {modelLimitsEnabled && (
        <div className="space-y-2">
          <p className="text-xs text-text-tertiary">{t('keys.create.model_limits_hint')}</p>
          <ModelLimitPicker
            models={modelOptions}
            selected={modelLimits}
            onChange={setModelLimits}
            disabled={disabled}
          />
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// CreateKeyDialog
// ---------------------------------------------------------------------------

interface CreateKeyDialogProps {
  open: boolean
  onClose: () => void
  onCreated: (data: { id: string; key: string }) => void
  userId?: string
}

function CreateKeyDialog({ open, onClose, onCreated, userId }: CreateKeyDialogProps) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [expiresAt, setExpiresAt] = useState<string | null>(dateInputToISO(defaultExpiryDateInput(90)))
  const [nameError, setNameError] = useState<string | undefined>()
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [dailyTokenLimit, setDailyTokenLimit] = useState('')
  const [monthlyTokenLimit, setMonthlyTokenLimit] = useState('')
  const [requestsPerMinute, setRequestsPerMinute] = useState('')
  const [requestsPerDay, setRequestsPerDay] = useState('')
  const [spendCap, setSpendCap] = useState('')
  const [ipWhitelist, setIpWhitelist] = useState('')
  const [ipBlacklist, setIpBlacklist] = useState('')
  const [modelLimitsEnabled, setModelLimitsEnabled] = useState(false)
  const [modelLimits, setModelLimits] = useState<string[]>([])

  const { data: me } = useMe()
  const { data: modelsData } = useModels()
  const createKey = useCreateAPIKey()
  const { toast } = useToast()

  const modelOptions = useMemo<ModelLimitOption[]>(
    () =>
      (modelsData?.data ?? [])
        .filter((m) => m.is_active)
        .map((m) => ({ name: m.name, logo: m.logo }))
        .sort((a, b) => a.name.localeCompare(b.name)),
    [modelsData?.data],
  )

  function resetForm() {
    setName('')
    setExpiresAt(dateInputToISO(defaultExpiryDateInput(90)))
    setNameError(undefined)
    setShowAdvanced(false)
    setDailyTokenLimit('')
    setMonthlyTokenLimit('')
    setRequestsPerMinute('')
    setRequestsPerDay('')
    setSpendCap('')
    setIpWhitelist('')
    setIpBlacklist('')
    setModelLimitsEnabled(false)
    setModelLimits([])
  }

  function handleClose() {
    resetForm()
    onClose()
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()

    const trimmedName = name.trim()
    if (!trimmedName) {
      setNameError(t('keys.create.name_required'))
      return
    }
    setNameError(undefined)

    const targetUserId = userId ?? me?.id
    if (!targetUserId) return

    const params: CreateAPIKeyParams = {
      name: trimmedName,
      key_type: 'user_key',
      user_id: targetUserId,
      expires_at: expiresAt ?? undefined,
    }

    const parsedDaily = parseOptionalInt(dailyTokenLimit)
    if (parsedDaily !== undefined) params.daily_token_limit = parsedDaily
    const parsedMonthly = parseOptionalInt(monthlyTokenLimit)
    if (parsedMonthly !== undefined) params.monthly_token_limit = parsedMonthly
    const parsedRpm = parseOptionalInt(requestsPerMinute)
    if (parsedRpm !== undefined) params.requests_per_minute = parsedRpm
    const parsedRpd = parseOptionalInt(requestsPerDay)
    if (parsedRpd !== undefined) params.requests_per_day = parsedRpd
    const parsedSpend = parseOptionalFloat(spendCap)
    if (parsedSpend !== undefined) params.spend_cap = parsedSpend
    if (ipWhitelist.trim()) params.ip_whitelist = ipWhitelist.trim()
    if (ipBlacklist.trim()) params.ip_blacklist = ipBlacklist.trim()
    params.model_limits_enabled = modelLimitsEnabled
    if (modelLimitsEnabled && modelLimits.length > 0) params.model_limits = modelLimits

    createKey.mutate(params, {
      onSuccess: (data) => {
        handleClose()
        if (data.key) {
          rememberApiKey(data.id, data.key)
          onCreated({ id: data.id, key: data.key })
        }
      },
      onError: (err) => {
        toast({
          variant: 'error',
          message: err instanceof Error ? err.message : t('keys.create_failed'),
        })
      },
    })
  }

  return (
    <Dialog open={open} onClose={handleClose} title={t('keys.create.title')}>
      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <Input
          label={t('keys.create.name')}
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('keys.placeholder.name')}
          error={nameError}
          disabled={createKey.isPending}
        />
        <ExpiryField
          label={t('keys.create.expires')}
          value={expiresAt}
          onChange={setExpiresAt}
          disabled={createKey.isPending}
          description={t('keys.expiry.hint')}
        />

        <div className="border-t border-border pt-4">
          <button
            type="button"
            className="flex w-full items-center justify-between"
            onClick={() => setShowAdvanced((v) => !v)}
            disabled={createKey.isPending}
          >
            <span className="text-[10px] font-medium tracking-widest uppercase text-text-tertiary">
              {t('keys.create.advanced')}
            </span>
            <IconChevronDown
              className={[
                'h-3.5 w-3.5 text-text-tertiary transition-transform duration-150',
                showAdvanced ? 'rotate-180' : '',
              ].join(' ')}
            />
          </button>
          {showAdvanced && (
            <div className="mt-4">
              <AdvancedLimitsFields
                dailyTokenLimit={dailyTokenLimit}
                setDailyTokenLimit={setDailyTokenLimit}
                monthlyTokenLimit={monthlyTokenLimit}
                setMonthlyTokenLimit={setMonthlyTokenLimit}
                requestsPerMinute={requestsPerMinute}
                setRequestsPerMinute={setRequestsPerMinute}
                requestsPerDay={requestsPerDay}
                setRequestsPerDay={setRequestsPerDay}
                spendCap={spendCap}
                setSpendCap={setSpendCap}
                ipWhitelist={ipWhitelist}
                setIpWhitelist={setIpWhitelist}
                ipBlacklist={ipBlacklist}
                setIpBlacklist={setIpBlacklist}
                modelLimitsEnabled={modelLimitsEnabled}
                setModelLimitsEnabled={setModelLimitsEnabled}
                modelLimits={modelLimits}
                setModelLimits={setModelLimits}
                modelOptions={modelOptions}
                disabled={createKey.isPending}
              />
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={handleClose} disabled={createKey.isPending}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSubmit} loading={createKey.isPending}>
            {t('keys.create.submit')}
          </Button>
        </div>
      </form>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// KeyCreatedDialog
// ---------------------------------------------------------------------------

interface KeyCreatedDialogProps {
  keyValue: string | null
  onClose: () => void
}

function KeyCreatedDialog({ keyValue, onClose }: KeyCreatedDialogProps) {
  const { t } = useTranslation()

  return (
    <Dialog open={keyValue !== null} onClose={onClose} title={t('keys.created.title')} closeOnBackdrop={false}>
      <div className="space-y-5">
        <div className="flex justify-center">
          <span className="flex h-14 w-14 items-center justify-center rounded-full bg-success/10">
            <IconCheckCircle className="h-7 w-7 text-success" />
          </span>
        </div>
        <div className="flex items-start gap-2.5 rounded-lg border border-warning/25 bg-warning/5 px-3 py-2.5">
          <span className="text-xs leading-relaxed text-warning">{t('keys.created.warning')}</span>
        </div>
        <div className="rounded-lg border border-border bg-bg-primary px-4 py-3">
          <p className="break-all font-mono text-sm leading-relaxed text-text-primary">{keyValue}</p>
        </div>
        <div className="flex items-center justify-end gap-2">
          <CopyButton text={keyValue ?? ''} label={t('keys.created.copy')} copiedLabel={t('keys.created.copied')} />
          <Button onClick={onClose}>{t('keys.created.done')}</Button>
        </div>
      </div>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// EditKeyDialog
// ---------------------------------------------------------------------------

interface EditKeyDialogProps {
  apiKey: APIKeyResponse
  onClose: () => void
}

function EditKeyDialog({ apiKey, onClose }: EditKeyDialogProps) {
  const { t } = useTranslation()
  const [name, setName] = useState(apiKey.name)
  const [enabled, setEnabled] = useState(normalizeKeyStatus(apiKey.status) === 'active')
  const [expiresAt, setExpiresAt] = useState<string | null>(apiKey.expires_at ?? null)
  const [dailyTokenLimit, setDailyTokenLimit] = useState(
    apiKey.daily_token_limit > 0 ? String(apiKey.daily_token_limit) : '',
  )
  const [monthlyTokenLimit, setMonthlyTokenLimit] = useState(
    apiKey.monthly_token_limit > 0 ? String(apiKey.monthly_token_limit) : '',
  )
  const [requestsPerMinute, setRequestsPerMinute] = useState(
    apiKey.requests_per_minute > 0 ? String(apiKey.requests_per_minute) : '',
  )
  const [requestsPerDay, setRequestsPerDay] = useState(
    apiKey.requests_per_day > 0 ? String(apiKey.requests_per_day) : '',
  )
  const [spendCap, setSpendCap] = useState((apiKey.spend_cap ?? 0) > 0 ? String(apiKey.spend_cap) : '')
  const [ipWhitelist, setIpWhitelist] = useState(apiKey.ip_whitelist ?? '')
  const [ipBlacklist, setIpBlacklist] = useState(apiKey.ip_blacklist ?? '')
  const [modelLimitsEnabled, setModelLimitsEnabled] = useState(apiKey.model_limits_enabled ?? false)
  const [modelLimits, setModelLimits] = useState<string[]>(apiKey.model_limits ?? [])
  const [nameError, setNameError] = useState<string | undefined>()

  const { data: modelsData } = useModels()
  const updateKey = useUpdateAPIKey()
  const { toast } = useToast()

  const modelOptions = useMemo<ModelLimitOption[]>(() => {
    const fromApi = (modelsData?.data ?? [])
      .filter((m) => m.is_active)
      .map((m) => ({ name: m.name, logo: m.logo }))
    const byName = new Map(fromApi.map((m) => [m.name, m]))
    for (const name of modelLimits) {
      if (!byName.has(name)) byName.set(name, { name, logo: undefined })
    }
    return [...byName.values()].sort((a, b) => a.name.localeCompare(b.name))
  }, [modelsData?.data, modelLimits])

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()

    const trimmedName = name.trim()
    if (!trimmedName) {
      setNameError(t('keys.create.name_required'))
      return
    }
    setNameError(undefined)

    const params: UpdateAPIKeyParams = {}
    const nextStatus: APIKeyStatus = enabled ? 'active' : 'disabled'

    if (trimmedName !== apiKey.name) params.name = trimmedName
    if (nextStatus !== normalizeKeyStatus(apiKey.status)) params.status = nextStatus

    const prevExpires = apiKey.expires_at ?? null
    if (expiresAt !== prevExpires) {
      params.expires_at = expiresAt
    }

    const parsedDaily = dailyTokenLimit.trim() ? parseInt(dailyTokenLimit, 10) : 0
    if (!isNaN(parsedDaily) && parsedDaily !== apiKey.daily_token_limit) {
      params.daily_token_limit = parsedDaily
    }

    const parsedMonthly = monthlyTokenLimit.trim() ? parseInt(monthlyTokenLimit, 10) : 0
    if (!isNaN(parsedMonthly) && parsedMonthly !== apiKey.monthly_token_limit) {
      params.monthly_token_limit = parsedMonthly
    }

    const parsedRpm = requestsPerMinute.trim() ? parseInt(requestsPerMinute, 10) : 0
    if (!isNaN(parsedRpm) && parsedRpm !== apiKey.requests_per_minute) {
      params.requests_per_minute = parsedRpm
    }

    const parsedRpd = requestsPerDay.trim() ? parseInt(requestsPerDay, 10) : 0
    if (!isNaN(parsedRpd) && parsedRpd !== apiKey.requests_per_day) {
      params.requests_per_day = parsedRpd
    }

    const parsedSpend = spendCap.trim() ? parseFloat(spendCap) : 0
    if (!isNaN(parsedSpend) && parsedSpend !== (apiKey.spend_cap ?? 0)) {
      params.spend_cap = parsedSpend
    }

    if (ipWhitelist.trim() !== (apiKey.ip_whitelist ?? '')) {
      params.ip_whitelist = ipWhitelist.trim()
    }
    if (ipBlacklist.trim() !== (apiKey.ip_blacklist ?? '')) {
      params.ip_blacklist = ipBlacklist.trim()
    }
    if (modelLimitsEnabled !== apiKey.model_limits_enabled) {
      params.model_limits_enabled = modelLimitsEnabled
    }
    const prevModels = apiKey.model_limits ?? []
    const modelsChanged =
      modelLimits.length !== prevModels.length ||
      modelLimits.some((m, i) => m !== prevModels[i])
    if (modelsChanged) {
      params.model_limits = modelLimits
    }

    if (Object.keys(params).length === 0) {
      onClose()
      return
    }

    updateKey.mutate(
      { keyId: apiKey.id, params },
      {
        onSuccess: () => {
          toast({ variant: 'success', message: t('keys.updated') })
          onClose()
        },
        onError: (err) => {
          toast({
            variant: 'error',
            message: err instanceof Error ? err.message : t('keys.update_failed'),
          })
        },
      },
    )
  }

  return (
    <Dialog open onClose={onClose} title={t('keys.edit.title')}>
      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <Input
          label={t('keys.create.name')}
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('keys.placeholder.name')}
          error={nameError}
          disabled={updateKey.isPending}
        />
        <Toggle
          checked={enabled}
          onChange={setEnabled}
          label={t('keys.edit.enabled')}
          disabled={updateKey.isPending || normalizeKeyStatus(apiKey.status) === 'expired'}
        />
        <ExpiryField
          label={t('keys.edit.expires')}
          value={expiresAt}
          onChange={setExpiresAt}
          disabled={updateKey.isPending}
          description={t('keys.expiry.hint')}
        />
        <AdvancedLimitsFields
          dailyTokenLimit={dailyTokenLimit}
          setDailyTokenLimit={setDailyTokenLimit}
          monthlyTokenLimit={monthlyTokenLimit}
          setMonthlyTokenLimit={setMonthlyTokenLimit}
          requestsPerMinute={requestsPerMinute}
          setRequestsPerMinute={setRequestsPerMinute}
          requestsPerDay={requestsPerDay}
          setRequestsPerDay={setRequestsPerDay}
          spendCap={spendCap}
          setSpendCap={setSpendCap}
          ipWhitelist={ipWhitelist}
          setIpWhitelist={setIpWhitelist}
          ipBlacklist={ipBlacklist}
          setIpBlacklist={setIpBlacklist}
          modelLimitsEnabled={modelLimitsEnabled}
          setModelLimitsEnabled={setModelLimitsEnabled}
          modelLimits={modelLimits}
          setModelLimits={setModelLimits}
          modelOptions={modelOptions}
          disabled={updateKey.isPending}
        />
        <KeySubscriptionField
          keyId={apiKey.id}
          modelLimitsEnabled={modelLimitsEnabled}
          modelLimits={modelLimits}
          disabled={updateKey.isPending}
        />
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={onClose} disabled={updateKey.isPending}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSubmit} loading={updateKey.isPending}>
            {t('keys.edit.save')}
          </Button>
        </div>
      </form>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// KeysPage
// ---------------------------------------------------------------------------

export default function KeysPage() {
  const { t } = useTranslation()

  const [cursor, setCursor] = useState<string | undefined>()
  const [prevCursors, setPrevCursors] = useState<string[]>([])
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [createdKey, setCreatedKey] = useState<string | null>(null)
  const [revokeKeyId, setRevokeKeyId] = useState<string | null>(null)
  const [editKey, setEditKey] = useState<APIKeyResponse | null>(null)
  const [rotateKeyId, setRotateKeyId] = useState<string | null>(null)
  const [rotatedKey, setRotatedKey] = useState<string | null>(null)

  useEffect(() => {
    return () => setCreatedKey(null)
  }, [])

  const { data: keys, isLoading } = useAPIKeys({ cursor })
  const { data: wallet } = useMyWallet()
  const deleteKey = useDeleteAPIKey()
  const rotateKey = useRotateAPIKey()
  const patchStatus = usePatchAPIKeyStatus()
  const { toast } = useToast()

  const allKeys = useMemo(() => keys?.data ?? [], [keys?.data])
  const lowBalance = wallet != null && wallet.balance <= 0

  const statusLabel = (status: APIKeyStatus) => {
    const map: Record<APIKeyStatus, string> = {
      active: t('keys.status.active'),
      disabled: t('keys.status.disabled'),
      expired: t('keys.status.expired'),
      quota_exhausted: t('keys.status.quota_exhausted'),
    }
    return map[status] ?? status
  }

  const [totalKeys, activeKeys, expiringSoon] = useMemo(() => {
    const now = Date.now()
    const sevenDaysMs = 7 * 24 * 60 * 60 * 1000
    const total = allKeys.length
    const active = allKeys.filter((k) => normalizeKeyStatus(k.status) === 'active' && (!k.expires_at || new Date(k.expires_at).getTime() > now)).length
    const expiring = allKeys.filter((k) => {
      if (!k.expires_at) return false
      const exp = new Date(k.expires_at).getTime()
      return exp > now && exp - now <= sevenDaysMs
    }).length
    return [total, active, expiring] as const
  }, [allKeys])

  function toggleKeyStatus(row: APIKeyResponse) {
    const status = normalizeKeyStatus(row.status)
    const next: APIKeyStatus = status === 'active' ? 'disabled' : 'active'
    patchStatus.mutate(
      { keyId: row.id, status: next },
      {
        onSuccess: () => toast({ variant: 'success', message: t('keys.updated') }),
        onError: (err) => {
          toast({
            variant: 'error',
            message: err instanceof Error ? err.message : t('keys.update_failed'),
          })
        },
      },
    )
  }

  const columns: Column<APIKeyResponse>[] = [
    {
      key: 'name',
      header: t('keys.table.name'),
      render: (row) => <span className="text-text-primary font-medium">{row.name}</span>,
    },
    {
      key: 'key_hint',
      header: t('keys.table.key'),
      render: (row) => <KeyCopyButton keyId={row.id} hint={row.key_hint} />,
    },
    {
      key: 'status',
      header: t('keys.table.status'),
      render: (row) => {
        const status = normalizeKeyStatus(row.status)
        return (
          <Badge variant={statusBadgeVariant[status] ?? 'muted'}>
            {statusLabel(status)}
          </Badge>
        )
      },
    },
    {
      key: 'limits',
      header: t('keys.table.limits'),
      render: (row) => <LimitsCell row={row} />,
    },
    {
      key: 'spend_cap',
      header: t('keys.table.spend'),
      render: (row) => <SpendCapBar used={row.spend_used ?? 0} cap={row.spend_cap ?? 0} />,
    },
    {
      key: 'expires_at',
      header: t('keys.table.expires'),
      render: (row) => (
        <TimeAgo date={row.expires_at ?? ''} fallback={t('keys.expires.never')} />
      ),
    },
    {
      key: 'last_used_at',
      header: t('keys.table.last_used'),
      render: (row) =>
        row.last_used_at ? <TimeAgo date={row.last_used_at} /> : <span className="text-text-tertiary">—</span>,
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (row) => {
        if (row.key_type === 'session_key') return null
        const busy = deleteKey.isPending || rotateKey.isPending || patchStatus.isPending
        const status = normalizeKeyStatus(row.status)
        return (
          <div className="flex items-center justify-end gap-0.5">
            <Toggle
              checked={status === 'active'}
              onChange={() => toggleKeyStatus(row)}
              disabled={busy || status === 'expired' || status === 'quota_exhausted'}
              size="sm"
              title={status === 'active' ? t('keys.action.disable') : t('keys.action.enable')}
            />
            <Link
              to={`/analytics?key_id=${row.id}`}
              title={t('keys.action.analytics')}
              className="inline-flex items-center justify-center rounded-md p-1.5 text-text-secondary hover:text-text-primary hover:opacity-80 transition-all"
            >
              <IconChart className="h-4 w-4" />
            </Link>
            <Button
              variant="ghost"
              size="sm"
              className="!px-1.5"
              title={t('keys.action.edit')}
              onClick={() => setEditKey(row)}
              disabled={busy}
            >
              <IconPencil className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="!px-1.5"
              title={t('keys.action.rotate')}
              onClick={() => setRotateKeyId(row.id)}
              disabled={busy}
            >
              <IconRefresh className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="!px-1.5 text-error hover:text-error"
              title={t('keys.action.revoke')}
              onClick={() => setRevokeKeyId(row.id)}
              disabled={busy}
            >
              <IconTrash className="h-4 w-4" />
            </Button>
          </div>
        )
      },
    },
  ]

  function handleRevoke() {
    if (!revokeKeyId) return
    forgetApiKey(revokeKeyId)
    deleteKey.mutate(revokeKeyId, {
      onSuccess: () => {
        toast({ variant: 'success', message: t('keys.revoked') })
        setRevokeKeyId(null)
      },
      onError: (err) => {
        toast({
          variant: 'error',
          message: err instanceof Error ? err.message : t('keys.revoke_failed'),
        })
        setRevokeKeyId(null)
      },
    })
  }

  function handleRotate() {
    if (!rotateKeyId) return
    rotateKey.mutate(rotateKeyId, {
      onSuccess: (data) => {
        setRotateKeyId(null)
        if (data.key) {
          rememberApiKey(data.id, data.key)
          setRotatedKey(data.key)
        }
      },
      onError: (err) => {
        toast({
          variant: 'error',
          message: err instanceof Error ? err.message : t('keys.rotate_failed'),
        })
        setRotateKeyId(null)
      },
    })
  }

  const showEmptyState = !isLoading && allKeys.length === 0 && !keys?.has_more

  return (
    <>
      <PageHeader
        title={t('keys.title')}
        description={t('keys.desc')}
        actions={<Button onClick={() => setShowCreateDialog(true)}>{t('keys.add')}</Button>}
      />

      {lowBalance && (
        <Link to="/wallet" className="block no-underline mb-6">
          <Banner variant="warning" title={t('keys.wallet_empty')} />
        </Link>
      )}

      <div className="grid grid-cols-3 gap-4 mb-6">
        <StatCard
          label={t('keys.stats.total')}
          value={totalKeys}
          iconColor="purple"
          icon={<IconKey className="h-4 w-4" />}
        />
        <StatCard
          label={t('keys.stats.active')}
          value={activeKeys}
          iconColor="green"
          icon={<IconCheck className="h-4 w-4" />}
        />
        <StatCard
          label={t('keys.stats.expiring')}
          value={expiringSoon}
          iconColor="yellow"
          icon={<IconClock className="h-4 w-4" />}
        />
      </div>

      {showEmptyState ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-border bg-bg-secondary py-16">
          <span className="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-bg-tertiary">
            <IconKey className="h-7 w-7 text-text-tertiary" />
          </span>
          <h3 className="mb-1 text-base font-medium text-text-primary">{t('keys.empty.title')}</h3>
          <p className="mb-6 text-sm text-text-secondary">{t('keys.empty.desc')}</p>
          <Button onClick={() => setShowCreateDialog(true)}>{t('keys.add')}</Button>
        </div>
      ) : (
        <Table<APIKeyResponse>
          columns={columns}
          data={allKeys}
          keyExtractor={(row) => row.id}
          loading={isLoading}
          emptyMessage={t('keys.empty.table')}
          pagination={{
            cursor: cursor ?? null,
            hasMore: keys?.has_more ?? false,
            hasPrevious: prevCursors.length > 0,
            onNext: () => {
              if (keys?.next_cursor) {
                setPrevCursors((prev) => [...prev, cursor ?? ''])
                setCursor(keys.next_cursor)
              }
            },
            onPrevious: () => {
              const prev = prevCursors[prevCursors.length - 1]
              setPrevCursors((p) => p.slice(0, -1))
              setCursor(prev || undefined)
            },
          }}
        />
      )}

      <CreateKeyDialog
        open={showCreateDialog}
        onClose={() => setShowCreateDialog(false)}
        onCreated={({ key }) => setCreatedKey(key)}
      />

      <KeyCreatedDialog keyValue={createdKey} onClose={() => setCreatedKey(null)} />

      {editKey !== null && <EditKeyDialog apiKey={editKey} onClose={() => setEditKey(null)} />}

      <ConfirmDialog
        open={rotateKeyId !== null}
        onClose={() => setRotateKeyId(null)}
        onConfirm={handleRotate}
        title={t('keys.rotate.title')}
        description={t('keys.rotate.desc')}
        confirmLabel={t('keys.rotate.confirm')}
        loading={rotateKey.isPending}
      />

      <KeyCreatedDialog keyValue={rotatedKey} onClose={() => setRotatedKey(null)} />

      <ConfirmDialog
        open={revokeKeyId !== null}
        onClose={() => setRevokeKeyId(null)}
        onConfirm={handleRevoke}
        title={t('keys.revoke.title')}
        description={t('keys.revoke.desc')}
        confirmLabel={t('keys.revoke.confirm')}
        loading={deleteKey.isPending}
      />
    </>
  )
}