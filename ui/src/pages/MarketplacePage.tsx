import { useState } from 'react'
import { PageHeader } from '../components/ui/PageHeader'
import { Table } from '../components/ui/Table'
import type { Column } from '../components/ui/Table'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { Select } from '../components/ui/Select'
import { ConfirmDialog } from '../components/ui/Dialog'
import { TimeAgo } from '../components/ui/TimeAgo'
import { shortenId } from '../lib/utils'
import { useAdminTopups, useReviewTopup } from '../hooks/useWallet'
import type { TopupRequestItem } from '../hooks/useWallet'
import { useToast } from '../hooks/useToast'
import { useTranslation } from '../lib/i18n'

const topupStatusBadge: Record<string, 'success' | 'error' | 'warning'> = {
  pending: 'warning',
  approved: 'success',
  rejected: 'error',
}

export default function MarketplacePage() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const [status, setStatus] = useState('pending')
  const [cursor, setCursor] = useState('')
  const { data, isLoading } = useAdminTopups(status, cursor)
  const review = useReviewTopup()

  const [confirm, setConfirm] = useState<{
    topup: TopupRequestItem
    action: 'approved' | 'rejected'
  } | null>(null)

  function doReview() {
    if (!confirm) return
    review.mutate(
      { topupId: confirm.topup.id, status: confirm.action },
      {
        onSuccess: (res) => {
          toast({
            variant: 'success',
            message:
              confirm.action === 'approved'
                ? t('marketplace.topup_approved', { balance: res.balance.toFixed(2) })
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

  const columns: Column<TopupRequestItem>[] = [
    {
      key: 'created_at',
      header: t('wallet.col_time'),
      render: (row) => <TimeAgo date={row.created_at} />,
    },
    {
      key: 'user_id',
      header: t('marketplace.col_customer'),
      render: (row) => (
        <span className="font-mono text-xs text-text-secondary">{shortenId(row.user_id)}</span>
      ),
    },
    {
      key: 'amount',
      header: t('wallet.col_amount'),
      render: (row) => <span className="tabular-nums">${row.amount.toFixed(2)}</span>,
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
            <Button size="sm" onClick={() => setConfirm({ topup: row, action: 'approved' })}>
              {t('marketplace.approve')}
            </Button>
            <Button size="sm" variant="destructive" onClick={() => setConfirm({ topup: row, action: 'rejected' })}>
              {t('marketplace.reject')}
            </Button>
          </div>
        ) : null,
    },
  ]

  return (
    <>
      <PageHeader title={t('sidebar.topups')} description={t('marketplace.subtitle')} />
      <div className="mb-4 w-48">
        <Select
          options={[
            { value: 'pending', label: t('marketplace.status_pending') },
            { value: 'approved', label: t('marketplace.status_approved') },
            { value: 'rejected', label: t('marketplace.status_rejected') },
            { value: '', label: t('marketplace.status_all') },
          ]}
          value={status}
          onChange={(v) => {
            setStatus(v)
            setCursor('')
          }}
        />
      </div>
      <Table
        columns={columns}
        data={data?.data ?? []}
        keyExtractor={(r) => r.id}
        loading={isLoading}
        emptyMessage={t('marketplace.no_topups')}
        pagination={{
          cursor: data?.cursor ?? null,
          hasMore: data?.has_more ?? false,
          onNext: () => setCursor(data?.cursor ?? ''),
          onPrevious: () => setCursor(''),
          hasPrevious: cursor !== '',
        }}
        compact
      />
      <ConfirmDialog
        open={confirm !== null}
        onClose={() => setConfirm(null)}
        onConfirm={doReview}
        title={
          confirm?.action === 'approved'
            ? t('marketplace.confirm_approve_title')
            : t('marketplace.confirm_reject_title')
        }
        description={t('marketplace.confirm_review_message', {
          amount: confirm ? `$${confirm.topup.amount.toFixed(2)}` : '',
        })}
        confirmLabel={
          confirm?.action === 'approved' ? t('marketplace.approve') : t('marketplace.reject')
        }
        loading={review.isPending}
      />
    </>
  )
}