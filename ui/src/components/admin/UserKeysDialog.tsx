import React, { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Dialog, ConfirmDialog } from '../ui/Dialog'
import { Table } from '../ui/Table'
import type { Column } from '../ui/Table'
import { Badge } from '../ui/Badge'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { Toggle } from '../ui/Toggle'
import { ExpiryField, dateInputToISO, defaultExpiryDateInput } from '../keys/ExpiryField'
import { KeyHint } from '../ui/KeyHint'
import { CopyButton } from '../ui/CopyButton'
import { forgetApiKey, getRememberedApiKey, rememberApiKey } from '../../lib/apiKeySecrets'
import { TimeAgo } from '../ui/TimeAgo'
import { useTranslation } from '../../lib/i18n'
import {
  useAPIKeys,
  useCreateAPIKey,
  useDeleteAPIKey,
  usePatchAPIKeyStatus,
  normalizeKeyStatus,
} from '../../hooks/useAPIKeys'
import type { APIKeyResponse, APIKeyStatus } from '../../hooks/useAPIKeys'
import type { UserResponse } from '../../hooks/useUsers'
import { useToast } from '../../hooks/useToast'

interface UserKeysDialogProps {
  user: UserResponse | null
  onClose: () => void
}

const statusBadgeVariant: Record<APIKeyStatus, 'default' | 'info' | 'warning' | 'muted'> = {
  active: 'default',
  disabled: 'muted',
  expired: 'warning',
  quota_exhausted: 'warning',
}

interface AdminCreateKeyOverlayProps {
  userId: string
  onClose: () => void
  onCreated: (data: { id: string; key: string }) => void
}

function AdminCreateKeyOverlay({ userId, onClose, onCreated }: AdminCreateKeyOverlayProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const createKey = useCreateAPIKey()
  const [name, setName] = useState('')
  const [expiresAt, setExpiresAt] = useState<string | null>(dateInputToISO(defaultExpiryDateInput(90)))

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) return

    createKey.mutate(
      {
        name: trimmed,
        user_id: userId,
        expires_at: expiresAt ?? undefined,
      },
      {
        onSuccess: (data) => {
          onClose()
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
      },
    )
  }

  return (
    <Dialog open onClose={onClose} title={t('keys.create.title')}>
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label={t('keys.create.name')}
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('keys.placeholder.name')}
          disabled={createKey.isPending}
        />
        <ExpiryField
          label={t('keys.create.expires')}
          value={expiresAt}
          onChange={setExpiresAt}
          disabled={createKey.isPending}
          description={t('keys.expiry.hint')}
        />
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose} disabled={createKey.isPending}>
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

export function UserKeysDialog({ user, onClose }: UserKeysDialogProps) {
  const { t } = useTranslation()
  const { toast } = useToast()

  const [cursor, setCursor] = useState<string | undefined>()
  const [prevCursors, setPrevCursors] = useState<string[]>([])
  const [showCreate, setShowCreate] = useState(false)
  const [createdKey, setCreatedKey] = useState<string | null>(null)
  const [revokeKeyId, setRevokeKeyId] = useState<string | null>(null)

  const { data: keys, isLoading } = useAPIKeys({ cursor, userId: user?.id })
  const deleteKey = useDeleteAPIKey()
  const patchStatus = usePatchAPIKeyStatus()

  const allKeys = useMemo(() => keys?.data ?? [], [keys?.data])

  const statusLabel = (status: APIKeyStatus) => {
    const map: Record<APIKeyStatus, string> = {
      active: t('keys.status.active'),
      disabled: t('keys.status.disabled'),
      expired: t('keys.status.expired'),
      quota_exhausted: t('keys.status.quota_exhausted'),
    }
    return map[status] ?? status
  }

  function toggleStatus(row: APIKeyResponse) {
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

  const columns: Column<APIKeyResponse>[] = [
    {
      key: 'name',
      header: t('keys.table.name'),
      render: (row) => <span className="font-medium text-text-primary">{row.name}</span>,
    },
    {
      key: 'key_hint',
      header: t('keys.table.key'),
      render: (row) => (
        <KeyHint
          hint={row.key_hint}
          copyValue={getRememberedApiKey(row.id) ?? row.key_hint}
          copyLabel={t('keys.copy')}
          copiedLabel={t('common.copied')}
        />
      ),
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
        const status = normalizeKeyStatus(row.status)
        return (
        <div className="flex items-center justify-end gap-1">
          <Toggle
            checked={status === 'active'}
            onChange={() => toggleStatus(row)}
            disabled={patchStatus.isPending || status === 'expired' || status === 'quota_exhausted'}
            size="sm"
            title={status === 'active' ? t('keys.action.disable') : t('keys.action.enable')}
          />
          <Link
            to={`/analytics?key_id=${row.id}`}
            className="text-xs text-accent hover:underline no-underline px-2"
          >
            {t('keys.action.analytics')}
          </Link>
          <Button
            variant="ghost"
            size="sm"
            className="!px-2 text-xs text-error"
            onClick={() => setRevokeKeyId(row.id)}
            disabled={deleteKey.isPending}
          >
            {t('keys.revoke.confirm')}
          </Button>
        </div>
        )
      },
    },
  ]

  if (!user) return null

  return (
    <>
      <Dialog
        open
        onClose={onClose}
        title={t('keys.admin.user_keys', { name: user.display_name || user.email })}
      >
        <div className="space-y-4">
          <p className="text-sm text-text-secondary">{t('keys.admin.user_keys_desc')}</p>
          <div className="flex justify-end">
            <Button size="sm" onClick={() => setShowCreate(true)}>
              {t('keys.add')}
            </Button>
          </div>

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
        </div>
      </Dialog>

      {showCreate && (
        <AdminCreateKeyOverlay
          userId={user.id}
          onClose={() => setShowCreate(false)}
          onCreated={({ key }) => {
            setShowCreate(false)
            setCreatedKey(key)
          }}
        />
      )}

      {createdKey !== null && (
        <Dialog
          open
          onClose={() => setCreatedKey(null)}
          title={t('keys.created.title')}
          closeOnBackdrop={false}
        >
          <div className="space-y-4">
            <p className="text-xs text-warning">{t('keys.created.warning')}</p>
            <p className="break-all font-mono text-sm">{createdKey}</p>
            <div className="flex items-center justify-end gap-2">
              <CopyButton
                text={createdKey}
                label={t('keys.created.copy')}
                copiedLabel={t('keys.created.copied')}
              />
              <Button onClick={() => setCreatedKey(null)}>{t('keys.created.done')}</Button>
            </div>
          </div>
        </Dialog>
      )}

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