import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Table } from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import { Badge } from '../../components/ui/Badge'
import { Select } from '../../components/ui/Select'
import { Input } from '../../components/ui/Input'
import { Button } from '../../components/ui/Button'
import { PillGroup } from '../../components/ui/PillGroup'
import { TimeAgo } from '../../components/ui/TimeAgo'
import { Dialog, ConfirmDialog } from '../../components/ui/Dialog'
import {
  financeRangeISO,
  useAdminFinanceTopups,
  useReviewFinanceTopup,
  type FinanceRangeDays,
  type FinanceTopupItem,
} from '../../hooks/useFinance'
import { parseRangeParam, parseStatusParam } from './financeUrl'
import { exportData } from '../../lib/export'
import { formatCost } from '../../lib/utils'
import { useTranslation } from '../../lib/i18n'
import { useToast } from '../../hooks/useToast'

const topupStatusBadge: Record<string, 'success' | 'error' | 'warning'> = {
  pending: 'warning',
  completed: 'success',
  expired: 'error',
  failed: 'error',
}

function customerLabel(row: FinanceTopupItem): string {
  if (row.user_display_name) return row.user_display_name
  if (row.user_email) return row.user_email
  return row.user_id.slice(0, 8) + '…'
}

function creditAmount(row: FinanceTopupItem): number {
  return row.credit_amount ?? row.amount
}

export default function FinanceTopupsPage() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const review = useReviewFinanceTopup()
  const [searchParams, setSearchParams] = useSearchParams()

  const [status, setStatus] = useState(() => parseStatusParam(searchParams.get('status')))
  const [userId, setUserId] = useState(searchParams.get('user_id') ?? '')
  const [rangeDays, setRangeDays] = useState<FinanceRangeDays>(() =>
    parseRangeParam(searchParams.get('range'), 30),
  )
  const [cursor, setCursor] = useState('')
  const [confirm, setConfirm] = useState<{
    topup: FinanceTopupItem
    action: 'approve' | 'reject'
  } | null>(null)

  const { from, to } = useMemo(
    () => (status === 'pending' ? { from: undefined, to: undefined } : financeRangeISO(rangeDays)),
    [status, rangeDays],
  )

  const { data, isLoading } = useAdminFinanceTopups({
    from,
    to,
    status: status || undefined,
    user_id: userId || undefined,
    cursor,
  })

  useEffect(() => {
    const next = new URLSearchParams()
    if (status) next.set('status', status)
    if (userId) next.set('user_id', userId)
    if (status !== 'pending') next.set('range', `${rangeDays}d`)
    setSearchParams(next, { replace: true })
  }, [status, userId, rangeDays, setSearchParams])

  function doReview() {
    if (!confirm) return
    review.mutate(
      { topupId: confirm.topup.id, action: confirm.action },
      {
        onSuccess: (res) => {
          toast({
            variant: 'success',
            message:
              confirm.action === 'approve'
                ? t('marketplace.topup_approved', { balance: formatCost(res.balance) })
                : t('marketplace.topup_rejected'),
          })
          setConfirm(null)
        },
        onError: (e) => {
          toast({ variant: 'error', message: e.message })
          setConfirm(null)
        },
      },
    )
  }

  const columns: Column<FinanceTopupItem>[] = useMemo(() => [
    {
      key: 'created_at',
      header: t('wallet.col_time'),
      render: (row) => <TimeAgo date={row.completed_at ?? row.created_at} />,
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
      key: 'pay_amount',
      header: t('wallet.col_pay_amount'),
      render: (row) => <span className="tabular-nums">{formatCost(row.pay_amount ?? row.amount)}</span>,
    },
    {
      key: 'credit_amount',
      header: t('wallet.col_credit_amount'),
      render: (row) => (
        <span className="tabular-nums text-success">
          {formatCost(row.credit_amount ?? row.amount)}
          {(row.bonus_amount ?? 0) > 0 && (
            <span className="text-text-tertiary text-xs ml-1">(+{formatCost(row.bonus_amount ?? 0)})</span>
          )}
        </span>
      ),
    },
    {
      key: 'trade_no',
      header: t('wallet.col_payment_ref'),
      render: (row) => <code className="text-xs">{row.trade_no || row.payment_ref || '—'}</code>,
    },
    {
      key: 'sepay_tx_id',
      header: t('finance.col_sepay_tx'),
      render: (row) => <code className="text-xs text-text-secondary">{row.sepay_tx_id || '—'}</code>,
    },
    {
      key: 'status',
      header: t('wallet.col_status'),
      render: (row) => <Badge variant={topupStatusBadge[row.status] ?? 'muted'}>{row.status}</Badge>,
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (row) =>
        row.status === 'pending' ? (
          <div className="flex gap-2 justify-end">
            <Button size="sm" onClick={() => setConfirm({ topup: row, action: 'approve' })}>
              {t('marketplace.approve')}
            </Button>
            <Button
              size="sm"
              variant="destructive"
              onClick={() => setConfirm({ topup: row, action: 'reject' })}
            >
              {t('marketplace.reject')}
            </Button>
          </div>
        ) : null,
    },
  ], [t])

  function exportCsv() {
    const rows = data?.data ?? []
    exportData(
      rows.map((r) => ({
        time: r.completed_at ?? r.created_at,
        customer: customerLabel(r),
        email: r.user_email ?? '',
        pay_amount: r.pay_amount ?? r.amount,
        credit_amount: r.credit_amount ?? r.amount,
        bonus_amount: r.bonus_amount ?? 0,
        trade_no: r.trade_no ?? r.payment_ref,
        sepay_tx_id: r.sepay_tx_id ?? '',
        status: r.status,
      })),
      [
        { key: 'time', label: 'Time' },
        { key: 'customer', label: 'Customer' },
        { key: 'email', label: 'Email' },
        { key: 'pay_amount', label: 'Pay' },
        { key: 'credit_amount', label: 'Credit' },
        { key: 'bonus_amount', label: 'Bonus' },
        { key: 'trade_no', label: 'Trade no' },
        { key: 'sepay_tx_id', label: 'SePay TX' },
        { key: 'status', label: 'Status' },
      ],
      'finance-topups',
      'csv',
    )
  }

  const confirmAmount = confirm ? formatCost(creditAmount(confirm.topup)) : ''

  return (
    <>
      {status === 'pending' && (
        <p className="mb-4 text-xs text-text-tertiary">{t('finance.manual_review_hint')}</p>
      )}

      <div className="flex flex-wrap items-end gap-4 mb-4">
        <div className="w-40">
          <Select
            label={t('wallet.col_status')}
            value={status}
            options={[
              { value: '', label: t('marketplace.status_all') },
              { value: 'pending', label: t('marketplace.status_pending') },
              { value: 'completed', label: t('marketplace.status_completed') },
              { value: 'expired', label: t('marketplace.status_expired') },
              { value: 'failed', label: t('marketplace.status_failed') },
            ]}
            onChange={(v) => {
              setStatus(v)
              setCursor('')
            }}
          />
        </div>
        {status !== 'pending' && (
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
        )}
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
        emptyMessage={t('finance.no_topups')}
        pagination={{
          cursor: data?.cursor ?? null,
          hasMore: data?.has_more ?? false,
          onNext: () => setCursor(data?.cursor ?? ''),
          onPrevious: () => setCursor(''),
          hasPrevious: cursor !== '',
        }}
        compact
      />

      {confirm?.action === 'approve' && (
        <Dialog
          open
          onClose={() => setConfirm(null)}
          title={t('marketplace.confirm_approve_title')}
          footer={
            <div className="flex justify-end gap-3">
              <Button variant="secondary" onClick={() => setConfirm(null)} disabled={review.isPending}>
                {t('common.cancel')}
              </Button>
              <Button onClick={doReview} loading={review.isPending}>
                {t('marketplace.approve')}
              </Button>
            </div>
          }
        >
          <p className="text-sm text-text-secondary">
            {t('marketplace.confirm_review_message', { amount: confirmAmount })}
          </p>
        </Dialog>
      )}

      {confirm?.action === 'reject' && (
        <ConfirmDialog
          open
          onClose={() => setConfirm(null)}
          title={t('marketplace.confirm_reject_title')}
          description={t('marketplace.confirm_review_message', { amount: confirmAmount })}
          confirmLabel={t('marketplace.reject')}
          loading={review.isPending}
          onConfirm={doReview}
        />
      )}
    </>
  )
}