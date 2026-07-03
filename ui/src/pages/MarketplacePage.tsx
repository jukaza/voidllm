import { useState } from 'react'
import { PageHeader } from '../components/ui/PageHeader'
import { Table } from '../components/ui/Table'
import type { Column } from '../components/ui/Table'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { Dialog, ConfirmDialog } from '../components/ui/Dialog'
import { TimeAgo } from '../components/ui/TimeAgo'
import { shortenId } from '../lib/utils'
import {
  useAdminTopups,
  useReviewTopup,
} from '../hooks/useWallet'
import type { TopupRequestItem } from '../hooks/useWallet'
import {
  useProviders,
  useCreateProvider,
  useUpdateProvider,
  useDeleteProvider,
} from '../hooks/useProviders'
import type { ProviderItem } from '../hooks/useProviders'
import { useToast } from '../hooks/useToast'
import TabSwitcher from '../components/ui/TabSwitcher'
import { useTranslation } from '../lib/i18n'

const topupStatusBadge: Record<string, 'success' | 'error' | 'warning'> = {
  pending: 'warning',
  approved: 'success',
  rejected: 'error',
}

// ---------------------------------------------------------------------------
// Top-up review queue
// ---------------------------------------------------------------------------

function TopupsTab() {
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
      render: (row) => <code className="text-xs">{shortenId(row.user_id)}</code>,
    },
    {
      key: 'amount',
      header: t('wallet.col_amount'),
      align: 'right',
      render: (row) => <span className="font-medium">${row.amount.toFixed(2)}</span>,
    },
    {
      key: 'payment_ref',
      header: t('wallet.col_payment_ref'),
      render: (row) => <code className="text-xs text-text-secondary">{row.payment_ref || '—'}</code>,
    },
    {
      key: 'status',
      header: t('wallet.col_status'),
      render: (row) => (
        <Badge variant={topupStatusBadge[row.status] ?? 'warning'}>{row.status}</Badge>
      ),
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (row) =>
        row.status === 'pending' ? (
          <div className="flex gap-2 justify-end">
            <Button
              size="sm"
              onClick={() => setConfirm({ topup: row, action: 'approved' })}
            >
              {t('marketplace.approve')}
            </Button>
            <Button
              size="sm"
              variant="destructive"
              onClick={() => setConfirm({ topup: row, action: 'rejected' })}
            >
              {t('marketplace.reject')}
            </Button>
          </div>
        ) : null,
    },
  ]

  return (
    <>
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

// ---------------------------------------------------------------------------
// Providers management
// ---------------------------------------------------------------------------

function ProvidersTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data, isLoading } = useProviders()
  const createProvider = useCreateProvider()
  const updateProvider = useUpdateProvider()
  const deleteProvider = useDeleteProvider()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<ProviderItem | null>(null)
  const [name, setName] = useState('')
  const [contact, setContact] = useState('')
  const [notes, setNotes] = useState('')
  const [deleting, setDeleting] = useState<ProviderItem | null>(null)

  function openCreate() {
    setEditing(null)
    setName('')
    setContact('')
    setNotes('')
    setDialogOpen(true)
  }

  function openEdit(p: ProviderItem) {
    setEditing(p)
    setName(p.name)
    setContact(p.contact_info)
    setNotes(p.notes)
    setDialogOpen(true)
  }

  function submit() {
    if (!name.trim()) {
      toast({ variant: 'error', message: t('marketplace.provider_name_required') })
      return
    }
    const params = { name: name.trim(), contact_info: contact, notes }
    const opts = {
      onSuccess: () => {
        toast({ variant: 'success' as const, message: t('common.saved') })
        setDialogOpen(false)
      },
      onError: (e: Error) => toast({ variant: 'error', message: e.message }),
    }
    if (editing) {
      updateProvider.mutate({ id: editing.id, ...params }, opts)
    } else {
      createProvider.mutate(params, opts)
    }
  }

  function toggleStatus(p: ProviderItem) {
    updateProvider.mutate(
      { id: p.id, status: p.status === 'active' ? 'paused' : 'active' },
      { onError: (e) => toast({ variant: 'error', message: e.message }) },
    )
  }

  const columns: Column<ProviderItem>[] = [
    {
      key: 'name',
      header: t('marketplace.col_provider'),
      render: (row) => <span className="font-medium">{row.name}</span>,
    },
    {
      key: 'contact_info',
      header: t('marketplace.col_contact'),
      render: (row) => <span className="text-text-tertiary text-xs">{row.contact_info || '—'}</span>,
    },
    {
      key: 'status',
      header: t('wallet.col_status'),
      render: (row) => (
        <Badge variant={row.status === 'active' ? 'success' : 'muted'}>{row.status}</Badge>
      ),
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (row) => (
        <div className="flex gap-2 justify-end">
          <Button size="sm" variant="secondary" onClick={() => toggleStatus(row)}>
            {row.status === 'active' ? t('marketplace.pause') : t('marketplace.activate')}
          </Button>
          <Button size="sm" variant="secondary" onClick={() => openEdit(row)}>
            {t('common.edit')}
          </Button>
          <Button size="sm" variant="destructive" onClick={() => setDeleting(row)}>
            {t('common.delete')}
          </Button>
        </div>
      ),
    },
  ]

  return (
    <>
      <div className="mb-4 flex justify-end">
        <Button onClick={openCreate}>{t('marketplace.add_provider')}</Button>
      </div>
      <Table
        columns={columns}
        data={data?.data ?? []}
        keyExtractor={(r) => r.id}
        loading={isLoading}
        emptyMessage={t('marketplace.no_providers')}
        compact
      />
      <Dialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        title={editing ? t('marketplace.edit_provider') : t('marketplace.add_provider')}
        footer={
          <>
            <Button variant="secondary" onClick={() => setDialogOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              onClick={submit}
              loading={createProvider.isPending || updateProvider.isPending}
            >
              {t('common.save')}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Input
            label={t('marketplace.provider_name')}
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <Input
            label={t('marketplace.provider_contact')}
            value={contact}
            onChange={(e) => setContact(e.target.value)}
            placeholder="email / telegram"
          />
          <Input
            label={t('marketplace.provider_notes')}
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
        </div>
      </Dialog>
      <ConfirmDialog
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        onConfirm={() => {
          if (!deleting) return
          deleteProvider.mutate(deleting.id, {
            onSuccess: () => {
              toast({ variant: 'success', message: t('common.deleted') })
              setDeleting(null)
            },
            onError: (e) => {
              toast({ variant: 'error', message: e.message })
              setDeleting(null)
            },
          })
        }}
        title={t('marketplace.confirm_delete_provider')}
        description={deleting?.name ?? ''}
        confirmLabel={t('common.delete')}
        loading={deleteProvider.isPending}
      />
    </>
  )
}

// ---------------------------------------------------------------------------
// MarketplacePage — admin console for the resell business
// ---------------------------------------------------------------------------

export default function MarketplacePage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState('topups')

  return (
    <>
      <PageHeader
        title={t('marketplace.title')}
        description={t('marketplace.subtitle')}
      />
      <TabSwitcher
        tabs={[
          { key: 'topups', label: t('marketplace.tab_topups') },
          { key: 'providers', label: t('marketplace.tab_providers') },
        ]}
        activeKey={tab}
        onChange={setTab}
      />
      <div className="mt-6">
        {tab === 'topups' ? <TopupsTab /> : <ProvidersTab />}
      </div>
    </>
  )
}
