import { useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Table } from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import { Badge } from '../../components/ui/Badge'
import { Select } from '../../components/ui/Select'
import { Input } from '../../components/ui/Input'
import { Button } from '../../components/ui/Button'
import { PillGroup } from '../../components/ui/PillGroup'
import { TimeAgo } from '../../components/ui/TimeAgo'
import {
  financeRangeISO,
  useAdminFinanceTransactions,
  type FinanceRangeDays,
  type FinanceTransactionItem,
} from '../../hooks/useFinance'
import { parseRangeParam, parseTypeParam } from './financeUrl'
import { exportData } from '../../lib/export'
import { formatCost } from '../../lib/utils'
import { useTranslation } from '../../lib/i18n'

const txTypeBadge: Record<string, 'success' | 'error' | 'info' | 'warning'> = {
  topup: 'success',
  usage: 'info',
  adjustment: 'warning',
  refund: 'success',
}

function customerLabel(row: FinanceTransactionItem): string {
  if (row.user_display_name) return row.user_display_name
  if (row.user_email) return row.user_email
  return row.user_id.slice(0, 8) + '…'
}

export default function FinanceLedgerPage() {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()

  const [txType, setTxType] = useState(() => parseTypeParam(searchParams.get('type')))
  const [userId, setUserId] = useState(searchParams.get('user_id') ?? '')
  const [rangeDays, setRangeDays] = useState<FinanceRangeDays>(() =>
    parseRangeParam(searchParams.get('range'), 30),
  )
  const [cursor, setCursor] = useState('')

  const { from, to } = useMemo(() => financeRangeISO(rangeDays), [rangeDays])

  const { data, isLoading } = useAdminFinanceTransactions({
    from,
    to,
    type: txType || undefined,
    user_id: userId || undefined,
    cursor,
  })

  useEffect(() => {
    const next = new URLSearchParams()
    if (txType) next.set('type', txType)
    if (userId) next.set('user_id', userId)
    next.set('range', `${rangeDays}d`)
    setSearchParams(next, { replace: true })
  }, [txType, userId, rangeDays, setSearchParams])

  const columns: Column<FinanceTransactionItem>[] = [
    {
      key: 'created_at',
      header: t('wallet.col_time'),
      render: (row) => <TimeAgo date={row.created_at} />,
    },
    {
      key: 'customer',
      header: t('marketplace.col_customer'),
      render: (row) => (
        <div className="min-w-0">
          <div className="text-sm text-text-primary truncate">{customerLabel(row)}</div>
          {row.user_email && row.user_display_name && (
            <div className="text-xs text-text-tertiary truncate">{row.user_email}</div>
          )}
        </div>
      ),
    },
    {
      key: 'type',
      header: t('wallet.col_type'),
      render: (row) => <Badge variant={txTypeBadge[row.type] ?? 'info'}>{row.type}</Badge>,
    },
    {
      key: 'amount',
      header: t('wallet.col_amount'),
      align: 'right',
      render: (row) => (
        <span className={row.amount >= 0 ? 'text-success tabular-nums' : 'text-text-primary tabular-nums'}>
          {row.amount >= 0 ? '+' : ''}
          {formatCost(row.amount)}
        </span>
      ),
    },
    {
      key: 'balance_after',
      header: t('wallet.col_balance_after'),
      align: 'right',
      render: (row) => <span className="tabular-nums text-text-secondary">{formatCost(row.balance_after)}</span>,
    },
    {
      key: 'ref_id',
      header: t('wallet.col_ref'),
      render: (row) =>
        row.type === 'usage' && row.ref_id ? (
          <Link
            to={`/analytics/logs?request_id=${encodeURIComponent(row.ref_id)}`}
            className="text-xs text-accent hover:underline"
          >
            {row.ref_id.slice(0, 14)}…
          </Link>
        ) : (
          <span className="text-xs text-text-tertiary">—</span>
        ),
    },
    {
      key: 'description',
      header: t('wallet.col_description'),
      render: (row) => <span className="text-xs text-text-tertiary truncate max-w-[200px] block">{row.description || '—'}</span>,
    },
  ]

  function exportCsv() {
    const rows = data?.data ?? []
    exportData(
      rows.map((r) => ({
        time: r.created_at,
        customer: customerLabel(r),
        email: r.user_email ?? '',
        type: r.type,
        amount: r.amount,
        balance_after: r.balance_after,
        ref_id: r.ref_id,
        description: r.description,
      })),
      [
        { key: 'time', label: 'Time' },
        { key: 'customer', label: 'Customer' },
        { key: 'email', label: 'Email' },
        { key: 'type', label: 'Type' },
        { key: 'amount', label: 'Amount' },
        { key: 'balance_after', label: 'Balance after' },
        { key: 'ref_id', label: 'Ref' },
        { key: 'description', label: 'Description' },
      ],
      'finance-ledger',
      'csv',
    )
  }

  return (
    <>
      <div className="flex flex-wrap items-end gap-4 mb-4">
        <div className="w-40">
          <Select
            label={t('wallet.col_type')}
            value={txType}
            options={[
              { value: '', label: t('marketplace.status_all') },
              { value: 'topup', label: 'topup' },
              { value: 'usage', label: 'usage' },
              { value: 'adjustment', label: 'adjustment' },
              { value: 'refund', label: 'refund' },
            ]}
            onChange={(v) => {
              setTxType(v)
              setCursor('')
            }}
          />
        </div>
        <PillGroup
          label={t('finance.period')}
          value={rangeDays}
          onChange={(d) => {
            setRangeDays(d)
            setCursor('')
          }}
          options={[
            { value: 7, label: '7d' },
            { value: 30, label: '30d' },
            { value: 90, label: '90d' },
          ]}
        />
        <Input
          label={t('finance.filter_user')}
          value={userId}
          onChange={(e) => {
            setUserId(e.target.value)
            setCursor('')
          }}
          placeholder="user UUID"
          className="max-w-xs"
        />
        <Button variant="secondary" size="sm" className="self-end" onClick={exportCsv}>
          {t('finance.export_csv')}
        </Button>
      </div>

      <Table
        columns={columns}
        data={data?.data ?? []}
        keyExtractor={(r) => r.id}
        loading={isLoading}
        emptyMessage={t('finance.no_transactions')}
        pagination={{
          cursor: data?.cursor ?? null,
          hasMore: data?.has_more ?? false,
          onNext: () => setCursor(data?.cursor ?? ''),
          onPrevious: () => setCursor(''),
          hasPrevious: cursor !== '',
        }}
        compact
      />
    </>
  )
}