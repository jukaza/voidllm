import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { BrandMark } from '../components/brand/BrandMark'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { StatCard } from '../components/ui/StatCard'
import { Banner } from '../components/ui/Banner'
import { PillGroup } from '../components/ui/PillGroup'
import { useSiteConfig } from '../hooks/useSiteConfig'
import { useTranslation } from '../lib/i18n'
import { formatCost, formatNumber, formatTokens } from '../lib/utils'
import type { UsageResponse } from '../hooks/useUsage'

type PeriodDays = 7 | 30

async function fetchKeyUsage(apiKey: string, from: string, to: string): Promise<UsageResponse> {
  const params = new URLSearchParams({ from, to, group_by: 'day' })
  const res = await fetch(`/api/v1/usage/me?${params}`, {
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
  })

  if (!res.ok) {
    let message = res.statusText
    try {
      const body = await res.json() as { error?: { message?: string } }
      message = body?.error?.message ?? message
    } catch {
      // ignore parse errors
    }
    throw new Error(message)
  }

  return res.json() as Promise<UsageResponse>
}

function getRange(days: PeriodDays): { from: string; to: string } {
  const to = new Date()
  const from = new Date(to.getTime() - days * 24 * 3_600_000)
  return { from: from.toISOString(), to: to.toISOString() }
}

export default function KeyUsagePage() {
  const { t } = useTranslation()
  const { data: site } = useSiteConfig()

  const [apiKey, setApiKey] = useState('')
  const [keyVisible, setKeyVisible] = useState(false)
  const [periodDays, setPeriodDays] = useState<PeriodDays>(7)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [usage, setUsage] = useState<UsageResponse | null>(null)

  const { from, to } = useMemo(() => getRange(periodDays), [periodDays])

  const summary = useMemo(() => {
    const rows = usage?.data ?? []
    return rows.reduce(
      (acc, row) => ({
        requests: acc.requests + row.total_requests,
        tokens: acc.tokens + row.total_tokens,
        spend: acc.spend + row.revenue,
      }),
      { requests: 0, tokens: 0, spend: 0 },
    )
  }, [usage])

  async function queryUsage() {
    const trimmed = apiKey.trim()
    if (!trimmed) return

    setLoading(true)
    setError(null)
    setUsage(null)

    try {
      const data = await fetchKeyUsage(trimmed, from, to)
      setUsage(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('keyUsage.error'))
    } finally {
      setLoading(false)
    }
  }

  const periodOptions = [
    { value: 7 as PeriodDays, label: t('keyUsage.period_7d') },
    { value: 30 as PeriodDays, label: t('keyUsage.period_30d') },
  ]

  const hasResults = usage !== null && !loading

  return (
    <div className="min-h-screen bg-bg-primary text-text-primary">
      <header className="border-b border-border">
        <div className="max-w-3xl mx-auto px-6 py-4 flex items-center justify-between">
          <Link to="/" className="no-underline">
            <BrandMark />
          </Link>
          <Link to="/" className="text-sm text-text-secondary hover:text-text-primary no-underline">
            {t('keyUsage.back_home')}
          </Link>
        </div>
      </header>

      <main className="max-w-3xl mx-auto px-6 py-12">
        <div className="text-center mb-10">
          <h1 className="text-3xl font-bold tracking-tight">{t('keyUsage.title')}</h1>
          <p className="mt-2 text-text-secondary">{t('keyUsage.subtitle')}</p>
          {site?.server_address && (
            <p className="mt-1 text-xs text-text-tertiary font-mono">{site.server_address}</p>
          )}
        </div>

        <div className="space-y-4 mb-8">
          <div className="flex gap-3">
            <div className="relative flex-1">
              <Input
                type={keyVisible ? 'text' : 'password'}
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder={t('keyUsage.placeholder')}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') void queryUsage()
                }}
              />
              <button
                type="button"
                onClick={() => setKeyVisible((v) => !v)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-text-tertiary hover:text-text-primary"
              >
                {keyVisible ? t('common.close') : t('common.view')}
              </button>
            </div>
            <Button onClick={() => void queryUsage()} loading={loading}>
              {loading ? t('keyUsage.querying') : t('keyUsage.query')}
            </Button>
          </div>
          <p className="text-xs text-text-tertiary text-center">{t('keyUsage.privacy')}</p>

          <div className="flex justify-center">
            <PillGroup
              options={periodOptions}
              value={periodDays}
              onChange={setPeriodDays}
            />
          </div>
        </div>

        {error && (
          <Banner variant="error" title={t('keyUsage.error')} description={error} className="mb-6" />
        )}

        {hasResults && summary.requests === 0 && summary.tokens === 0 && (
          <div className="rounded-xl border border-border bg-bg-secondary p-12 text-center text-sm text-text-tertiary">
            {t('keyUsage.no_data')}
          </div>
        )}

        {hasResults && (summary.requests > 0 || summary.tokens > 0) && (
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <StatCard
              label={t('keyUsage.requests')}
              value={formatNumber(summary.requests)}
            />
            <StatCard
              label={t('keyUsage.tokens')}
              value={formatTokens(summary.tokens)}
            />
            <StatCard
              label={t('keyUsage.spend')}
              value={formatCost(summary.spend)}
            />
          </div>
        )}
      </main>
    </div>
  )
}