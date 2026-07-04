import { useState } from 'react'
import { Link } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { StatCard } from '../components/ui/StatCard'
import { Table } from '../components/ui/Table'
import type { Column } from '../components/ui/Table'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Dialog } from '../components/ui/Dialog'
import { TimeAgo } from '../components/ui/TimeAgo'
import {
  useMyWallet,
  useMyTransactions,
  useMyTopups,
  useCreateTopup,
} from '../hooks/useWallet'
import type { TransactionItem, TopupRequestItem } from '../hooks/useWallet'
import { useToast } from '../hooks/useToast'
import { useTranslation } from '../lib/i18n'
import { formatCost } from '../lib/utils'

function IconWallet() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M21 12V7H5a2 2 0 0 1 0-4h14v4" />
      <path d="M3 5v14a2 2 0 0 0 2 2h16v-5" />
      <path d="M18 12a2 2 0 0 0 0 4h4v-4Z" />
    </svg>
  )
}

const txTypeBadge: Record<string, 'success' | 'error' | 'info' | 'warning'> = {
  topup: 'success',
  usage: 'info',
  adjustment: 'warning',
  refund: 'success',
}

const topupStatusBadge: Record<string, 'success' | 'error' | 'warning'> = {
  pending: 'warning',
  approved: 'success',
  rejected: 'error',
}

export default function WalletPage() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data: wallet, isLoading: walletLoading } = useMyWallet()

  const [txCursor, setTxCursor] = useState('')
  const { data: txData, isLoading: txLoading } = useMyTransactions(txCursor)

  const [topupCursor] = useState('')
  const { data: topupData } = useMyTopups(topupCursor)

  const createTopup = useCreateTopup()
  const [topupOpen, setTopupOpen] = useState(false)
  const [amount, setAmount] = useState('')
  const [paymentRef, setPaymentRef] = useState('')

  function submitTopup() {
    const value = parseFloat(amount)
    if (isNaN(value) || value <= 0) {
      toast({ variant: 'error', message: t('wallet.invalid_amount') })
      return
    }
    createTopup.mutate(
      { amount: value, payment_ref: paymentRef },
      {
        onSuccess: () => {
          toast({ variant: 'success', message: t('wallet.topup_submitted') })
          setTopupOpen(false)
          setAmount('')
          setPaymentRef('')
        },
        onError: (e) => toast({ variant: 'error', message: e.message }),
      },
    )
  }

  const txColumns: Column<TransactionItem>[] = [
    {
      key: 'created_at',
      header: t('wallet.col_time'),
      render: (row) => <TimeAgo date={row.created_at} />,
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
        <span className={row.amount >= 0 ? 'text-success' : 'text-text-primary'}>
          {row.amount >= 0 ? '+' : ''}
          {formatCost(row.amount)}
        </span>
      ),
    },
    {
      key: 'balance_after',
      header: t('wallet.col_balance_after'),
      align: 'right',
      render: (row) => <span className="text-text-secondary">{formatCost(row.balance_after)}</span>,
    },
    {
      key: 'description',
      header: t('wallet.col_description'),
      render: (row) => <span className="text-text-tertiary text-xs">{row.description || '—'}</span>,
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
            {row.ref_id.slice(0, 12)}…
          </Link>
        ) : (
          <span className="text-text-tertiary text-xs">—</span>
        ),
    },
  ]

  const topupColumns: Column<TopupRequestItem>[] = [
    {
      key: 'created_at',
      header: t('wallet.col_time'),
      render: (row) => <TimeAgo date={row.created_at} />,
    },
    {
      key: 'amount',
      header: t('wallet.col_amount'),
      align: 'right',
      render: (row) => <span>{formatCost(row.amount)}</span>,
    },
    {
      key: 'payment_ref',
      header: t('wallet.col_payment_ref'),
      render: (row) => (
        <code className="text-xs text-text-secondary">{row.payment_ref || '—'}</code>
      ),
    },
    {
      key: 'status',
      header: t('wallet.col_status'),
      render: (row) => (
        <Badge variant={topupStatusBadge[row.status] ?? 'warning'}>{row.status}</Badge>
      ),
    },
    {
      key: 'note',
      header: t('wallet.col_note'),
      render: (row) => <span className="text-text-tertiary text-xs">{row.note || '—'}</span>,
    },
  ]

  return (
    <>
      <PageHeader
        title={t('wallet.title')}
        description={t('wallet.subtitle')}
        actions={
          <Button onClick={() => setTopupOpen(true)}>{t('wallet.topup_button')}</Button>
        }
      />

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <StatCard
          label={t('wallet.balance')}
          value={walletLoading ? '...' : formatCost(wallet?.balance ?? 0)}
          icon={<IconWallet />}
          iconColor="green"
        />
      </div>

      <h3 className="text-sm font-semibold text-text-primary mb-3">
        {t('wallet.topup_history')}
      </h3>
      <Table
        columns={topupColumns}
        data={topupData?.data ?? []}
        keyExtractor={(r) => r.id}
        emptyMessage={t('wallet.no_topups')}
        compact
        className="mb-8"
      />

      <h3 className="text-sm font-semibold text-text-primary mb-3">
        {t('wallet.tx_history')}
      </h3>
      <Table
        columns={txColumns}
        data={txData?.data ?? []}
        keyExtractor={(r) => r.id}
        loading={txLoading}
        emptyMessage={t('wallet.no_transactions')}
        pagination={{
          cursor: txData?.cursor ?? null,
          hasMore: txData?.has_more ?? false,
          onNext: () => setTxCursor(txData?.cursor ?? ''),
          onPrevious: () => setTxCursor(''),
          hasPrevious: txCursor !== '',
        }}
        compact
      />

      <Dialog
        open={topupOpen}
        onClose={() => setTopupOpen(false)}
        title={t('wallet.topup_title')}
        footer={
          <>
            <Button variant="secondary" onClick={() => setTopupOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button onClick={submitTopup} loading={createTopup.isPending}>
              {t('wallet.topup_submit')}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <p className="text-sm text-text-tertiary">{t('wallet.topup_instructions')}</p>
          <Input
            label={t('wallet.topup_amount')}
            type="number"
            min="0"
            step="1"
            suffix="₫"
            inputMode="numeric"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="100000"
          />
          <Input
            label={t('wallet.topup_ref')}
            value={paymentRef}
            onChange={(e) => setPaymentRef(e.target.value)}
            placeholder={t('wallet.topup_ref_placeholder')}
          />
        </div>
      </Dialog>
    </>
  )
}
