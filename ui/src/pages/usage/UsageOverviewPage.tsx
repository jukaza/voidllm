import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { StatCard } from '../../components/ui/StatCard'
import { AreaChart } from '../../components/ui/charts'
import { useMyUsage } from '../../hooks/useUsage'
import { formatNumber, formatTokens, formatCost } from '../../lib/utils'

function getLast7Days(): { from: string; to: string } {
  const now = new Date()
  const from = new Date(now.getTime() - 7 * 24 * 3_600_000)
  return { from: from.toISOString(), to: now.toISOString() }
}

function getLast24h(): { from: string; to: string } {
  const now = new Date()
  const from = new Date(now.getTime() - 24 * 3_600_000)
  return { from: from.toISOString(), to: now.toISOString() }
}

// ---------------------------------------------------------------------------
// Icons
// ---------------------------------------------------------------------------

function ActivityIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
    </svg>
  )
}

function SparklesIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3l1.88 5.76a1 1 0 00.95.69H21l-5.12 3.72a1 1 0 00-.36 1.12L17.4 20 12 16.28 6.6 20l1.88-5.71a1 1 0 00-.36-1.12L3 9.45h6.17a1 1 0 00.95-.69L12 3z" />
    </svg>
  )
}

function DollarIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <line x1="12" y1="1" x2="12" y2="23" />
      <path d="M17 5H9.5a3.5 3.5 0 000 7h5a3.5 3.5 0 010 7H6" />
    </svg>
  )
}

function ArrowRightIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <line x1="5" y1="12" x2="19" y2="12" />
      <polyline points="12 5 19 12 12 19" />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// UsageOverviewPage
// ---------------------------------------------------------------------------

export default function UsageOverviewPage() {
  const { from: from24h, to: to24h } = useMemo(() => getLast24h(), [])
  const { from: from7d, to: to7d } = useMemo(() => getLast7Days(), [])

  // LLM: 24h totals + 7d daily trend
  const llmTotals = useMyUsage(from24h, to24h, 'model')
  const llmTrend = useMyUsage(from7d, to7d, 'day')

  const llmSummary = useMemo(() => {
    if (!llmTotals.data?.data) return { requests: 0, tokens: 0, cost: 0 }
    return llmTotals.data.data.reduce(
      (acc, d) => ({
        requests: acc.requests + d.total_requests,
        tokens: acc.tokens + d.total_tokens,
        cost: acc.cost + d.cost_estimate,
      }),
      { requests: 0, tokens: 0, cost: 0 },
    )
  }, [llmTotals.data])

  const llmLoading = llmTotals.isLoading

  return (
    <div className="grid grid-cols-1 gap-6">
      {/* LLM panel */}
      <div className="bg-bg-secondary rounded-xl border border-border p-6 flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-text-primary">LLM Usage</h3>
          <Link
            to="/usage/llm"
            className="flex items-center gap-1 text-xs text-accent hover:text-accent/80 transition-colors no-underline"
          >
            View Details
            <ArrowRightIcon />
          </Link>
        </div>

        <div className="grid grid-cols-3 gap-3">
          <StatCard
            label="Requests 24h"
            value={llmLoading ? '...' : formatNumber(llmSummary.requests)}
            icon={<ActivityIcon />}
            iconColor="purple"
            className="p-3"
          />
          <StatCard
            label="Tokens 24h"
            value={llmLoading ? '...' : formatTokens(llmSummary.tokens)}
            icon={<SparklesIcon />}
            iconColor="blue"
            className="p-3"
          />
          <StatCard
            label="Cost 24h"
            value={llmLoading ? '...' : formatCost(llmSummary.cost)}
            icon={<DollarIcon />}
            iconColor="green"
            className="p-3"
          />
        </div>

        <div>
          <p className="text-xs text-text-tertiary mb-3">Requests - last 7 days</p>
          <AreaChart
            data={(llmTrend.data?.data ?? []).map((d) => ({
              label: d.group_key.length > 10 ? d.group_key.slice(5) : d.group_key,
              value: d.total_requests,
            }))}
            height={140}
            formatValue={formatNumber}
          />
        </div>
      </div>
    </div>
  )
}
