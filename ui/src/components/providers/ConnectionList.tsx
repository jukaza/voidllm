import { useMemo, useState } from 'react'
import { Table } from '../ui/Table'
import type { Column } from '../ui/Table'
import { Badge } from '../ui/Badge'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { Textarea } from '../ui/Textarea'
import { Dialog, ConfirmDialog } from '../ui/Dialog'
import { Toggle } from '../ui/Toggle'
import { CooldownTimer } from './CooldownTimer'
import {
  useProviderConnections,
  useCreateProviderConnection,
  useBulkCreateProviderConnections,
  useUpdateProviderConnection,
  useDeleteProviderConnection,
  useUnlockProviderConnection,
  useReorderProviderConnections,
  useTestProviderConnection,
} from '../../hooks/useProviderConnections'
import type { ProviderConnection } from '../../hooks/useProviderConnections'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { TimeAgo } from '../ui/TimeAgo'

interface ConnectionListProps {
  providerId: string
}

function isLocked(conn: ProviderConnection): boolean {
  if (conn.earliest_lock_until) {
    const end = Date.parse(conn.earliest_lock_until)
    if (!Number.isNaN(end) && end > Date.now()) return true
  }
  return false
}

function testStatusVariant(status: string): 'success' | 'error' | 'warning' | 'muted' {
  if (status === 'active') return 'success'
  if (status === 'unavailable') return 'error'
  if (status === 'pending') return 'warning'
  return 'muted'
}

export function ConnectionList({ providerId }: ConnectionListProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data, isLoading } = useProviderConnections(providerId)
  const createConn = useCreateProviderConnection(providerId)
  const bulkCreateConn = useBulkCreateProviderConnections(providerId)
  const updateConn = useUpdateProviderConnection(providerId)
  const deleteConn = useDeleteProviderConnection(providerId)
  const unlockConn = useUnlockProviderConnection(providerId)
  const reorderConn = useReorderProviderConnections(providerId)
  const testConn = useTestProviderConnection(providerId)

  const connections = useMemo(
    () => [...(data?.data ?? [])].sort((a, b) => a.priority - b.priority),
    [data?.data],
  )

  const [addOpen, setAddOpen] = useState(false)
  const [bulkOpen, setBulkOpen] = useState(false)
  const [editing, setEditing] = useState<ProviderConnection | null>(null)
  const [deleting, setDeleting] = useState<ProviderConnection | null>(null)
  const [testingId, setTestingId] = useState<string | null>(null)

  const [name, setName] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [bulkText, setBulkText] = useState('')

  function openAdd() {
    setName('')
    setApiKey('')
    setAddOpen(true)
  }

  function openEdit(conn: ProviderConnection) {
    setEditing(conn)
    setName(conn.name)
    setApiKey('')
    setAddOpen(true)
  }

  function openBulk() {
    setBulkText('')
    setBulkOpen(true)
  }

  function submitAdd() {
    if (!name.trim()) {
      toast({ variant: 'error', message: t('provider_detail.conn_name_required') })
      return
    }
    const opts = {
      onSuccess: () => {
        toast({ variant: 'success', message: t('common.saved') })
        setAddOpen(false)
        setEditing(null)
      },
      onError: (e: Error) => toast({ variant: 'error', message: e.message }),
    }
    if (editing) {
      updateConn.mutate(
        {
          connectionId: editing.id,
          name: name.trim(),
          api_key: apiKey.trim() || undefined,
        },
        opts,
      )
    } else {
      createConn.mutate(
        { name: name.trim(), api_key: apiKey.trim() || undefined },
        opts,
      )
    }
  }

  function submitBulk() {
    if (!bulkText.trim()) {
      toast({ variant: 'error', message: t('provider_detail.bulk_empty') })
      return
    }
    bulkCreateConn.mutate(bulkText, {
      onSuccess: (res) => {
        const failed = res.results.filter((r) => r.error)
        const ok = res.results.length - failed.length
        if (failed.length > 0) {
          toast({
            variant: 'info',
            message: t('provider_detail.bulk_partial', { ok, failed: failed.length }),
          })
        } else {
          toast({ variant: 'success', message: t('provider_detail.bulk_success', { ok }) })
        }
        setBulkOpen(false)
      },
      onError: (e: Error) => toast({ variant: 'error', message: e.message }),
    })
  }

  function moveConnection(conn: ProviderConnection, direction: -1 | 1) {
    const idx = connections.findIndex((c) => c.id === conn.id)
    const target = idx + direction
    if (idx < 0 || target < 0 || target >= connections.length) return
    const ordered = connections.map((c) => c.id)
    ;[ordered[idx], ordered[target]] = [ordered[target], ordered[idx]]
    reorderConn.mutate(ordered, {
      onError: (e) => toast({ variant: 'error', message: e.message }),
    })
  }

  function runTest(conn: ProviderConnection) {
    setTestingId(conn.id)
    testConn.mutate(conn.id, {
      onSuccess: (res) => {
        if (res.ok) {
          toast({ variant: 'success', message: t('provider_detail.test_ok') })
        } else {
          toast({
            variant: 'error',
            message: res.error ?? t('provider_detail.test_failed'),
          })
        }
      },
      onError: (e) => toast({ variant: 'error', message: e.message }),
      onSettled: () => setTestingId(null),
    })
  }

  function runUnlock(conn: ProviderConnection) {
    unlockConn.mutate(conn.id, {
      onSuccess: () => toast({ variant: 'success', message: t('provider_detail.unlocked') }),
      onError: (e) => toast({ variant: 'error', message: e.message }),
    })
  }

  function toggleActive(conn: ProviderConnection) {
    updateConn.mutate(
      { connectionId: conn.id, is_active: !conn.is_active },
      { onError: (e) => toast({ variant: 'error', message: e.message }) },
    )
  }

  const columns: Column<ProviderConnection>[] = [
    {
      key: 'priority',
      header: '#',
      width: '48px',
      render: (row) => (
        <span className="text-text-tertiary tabular-nums text-xs">{row.priority}</span>
      ),
    },
    {
      key: 'name',
      header: t('provider_detail.col_name'),
      render: (row) => (
        <div>
          <span className="font-medium">{row.name}</span>
          {row.has_api_key && (
            <span className="ml-2 text-xs text-text-tertiary">{t('connection.api_key_on_file')}</span>
          )}
        </div>
      ),
    },
    {
      key: 'status',
      header: t('common.status'),
      render: (row) => (
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={row.is_active ? 'success' : 'muted'}>
            {row.is_active ? t('providers.status_active') : t('providers.status_paused')}
          </Badge>
          {row.test_status && (
            <Badge variant={testStatusVariant(row.test_status)}>{row.test_status}</Badge>
          )}
          {isLocked(row) && (
            <Badge variant="warning">{t('provider_detail.locked')}</Badge>
          )}
        </div>
      ),
    },
    {
      key: 'locks',
      header: t('provider_detail.col_locks'),
      render: (row) => (
        <div className="space-y-1">
          {row.locked_until && <CooldownTimer until={row.locked_until} />}
          {row.earliest_lock_until &&
            row.earliest_lock_until !== row.locked_until && (
              <CooldownTimer until={row.earliest_lock_until} />
            )}
          {Object.entries(row.model_locks ?? {}).map(([model, until]) => {
            const end = Date.parse(until)
            if (Number.isNaN(end) || end <= Date.now()) return null
            return (
              <div key={model} className="flex flex-wrap items-center gap-1 text-xs">
                <span className="font-mono text-text-tertiary">{model}</span>
                <CooldownTimer until={until} />
              </div>
            )
          })}
          {!isLocked(row) && !row.locked_until && (
            <span className="text-xs text-text-tertiary">—</span>
          )}
        </div>
      ),
    },
    {
      key: 'last_error',
      header: t('provider_detail.col_last_error'),
      render: (row) =>
        row.last_error ? (
          <span className="text-xs text-error line-clamp-2" title={row.last_error}>
            {row.last_error}
          </span>
        ) : (
          <span className="text-xs text-text-tertiary">—</span>
        ),
    },
    {
      key: 'last_used',
      header: t('provider_detail.col_last_used'),
      render: (row) => <TimeAgo date={row.last_used_at ?? ''} fallback="—" />,
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (row) => (
        <div className="flex flex-wrap gap-1 justify-end">
          <Button
            size="sm"
            variant="secondary"
            onClick={() => moveConnection(row, -1)}
            disabled={connections[0]?.id === row.id || reorderConn.isPending}
            title={t('provider_detail.move_up')}
          >
            ↑
          </Button>
          <Button
            size="sm"
            variant="secondary"
            onClick={() => moveConnection(row, 1)}
            disabled={connections[connections.length - 1]?.id === row.id || reorderConn.isPending}
            title={t('provider_detail.move_down')}
          >
            ↓
          </Button>
          <Button
            size="sm"
            variant="secondary"
            onClick={() => runTest(row)}
            loading={testingId === row.id}
          >
            {t('provider_detail.test')}
          </Button>
          {isLocked(row) && (
            <Button
              size="sm"
              variant="secondary"
              onClick={() => runUnlock(row)}
              loading={unlockConn.isPending}
            >
              {t('provider_detail.unlock')}
            </Button>
          )}
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
      <div className="mb-4 flex flex-wrap justify-between items-center gap-2">
        <p className="text-sm text-text-secondary">{t('provider_detail.connections_desc')}</p>
        <div className="flex gap-2">
          <Button variant="secondary" size="sm" onClick={openBulk}>
            {t('provider_detail.bulk_add')}
          </Button>
          <Button size="sm" onClick={openAdd}>
            {t('provider_detail.add_key')}
          </Button>
        </div>
      </div>

      <Table
        columns={columns}
        data={connections}
        keyExtractor={(r) => r.id}
        loading={isLoading}
        emptyMessage={t('provider_detail.connections_empty')}
        compact
      />

      <Dialog
        open={addOpen}
        onClose={() => {
          setAddOpen(false)
          setEditing(null)
        }}
        title={editing ? t('provider_detail.edit_key') : t('provider_detail.add_key')}
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => {
                setAddOpen(false)
                setEditing(null)
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button
              onClick={submitAdd}
              loading={createConn.isPending || updateConn.isPending}
            >
              {t('common.save')}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Input
            label={t('provider_detail.col_name')}
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <Input
            label={t('common.api_key')}
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder={editing?.has_api_key ? t('providers.key_keep') : undefined}
            description={t('connection.encrypted_at_rest')}
          />
          {editing && (
            <Toggle
              checked={editing.is_active}
              onChange={() => toggleActive(editing)}
              label={t('providers.status_active')}
            />
          )}
        </div>
      </Dialog>

      <Dialog
        open={bulkOpen}
        onClose={() => setBulkOpen(false)}
        title={t('provider_detail.bulk_add')}
        footer={
          <>
            <Button variant="secondary" onClick={() => setBulkOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button onClick={submitBulk} loading={bulkCreateConn.isPending}>
              {t('common.save')}
            </Button>
          </>
        }
      >
        <Textarea
          label={t('provider_detail.bulk_label')}
          value={bulkText}
          onChange={(e) => setBulkText(e.target.value)}
          rows={8}
          placeholder={'primary|sk-abc123\nbackup|sk-def456'}
          description={t('provider_detail.bulk_hint')}
        />
      </Dialog>

      <ConfirmDialog
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        onConfirm={() => {
          if (!deleting) return
          deleteConn.mutate(deleting.id, {
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
        title={t('provider_detail.confirm_delete_conn')}
        description={deleting?.name ?? ''}
        confirmLabel={t('common.delete')}
        loading={deleteConn.isPending}
      />
    </>
  )
}