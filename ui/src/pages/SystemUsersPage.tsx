import React, { useMemo, useState } from 'react'
import { Link, Navigate } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { Table } from '../components/ui/Table'
import type { Column } from '../components/ui/Table'
import { Dialog, ConfirmDialog } from '../components/ui/Dialog'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'

import { TimeAgo } from '../components/ui/TimeAgo'
import { StatCard } from '../components/ui/StatCard'
import { useMe } from '../hooks/useMe'
import {
  useUsers,
  useCreateUser,
  useUpdateUser,
  useDeleteUser,
  useManageUser,
  useAdjustUserWallet,
  useUserWallet,
  useRevokeUserSessions,
  type UserResponse,
  type UserRole,
  type CreateUserParams,
  type UpdateUserParams,
} from '../hooks/useUsers'
import { useToast } from '../hooks/useToast'
import { UserKeysDialog } from '../components/admin/UserKeysDialog'
import { formatCost } from '../lib/utils'

function roleBadge(role: UserRole) {
  if (role === 'root') return <Badge variant="default">Root</Badge>
  if (role === 'admin') return <Badge variant="info">Admin</Badge>
  return <Badge variant="muted">Member</Badge>
}

function statusBadge(status: string) {
  if (status === 'disabled') return <Badge variant="warning">Disabled</Badge>
  return <Badge variant="default">Active</Badge>
}

interface CreateUserDialogProps {
  open: boolean
  onClose: () => void
  canAssignAdmin: boolean
}

function CreateUserDialog({ open, onClose, canAssignAdmin }: CreateUserDialogProps) {
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<UserRole>('member')
  const createUser = useCreateUser()
  const { toast } = useToast()

  function handleClose() {
    setEmail('')
    setDisplayName('')
    setPassword('')
    setRole('member')
    onClose()
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const params: CreateUserParams = {
      email: email.trim(),
      display_name: displayName.trim(),
      password,
      role: canAssignAdmin ? role : 'member',
    }
    createUser.mutate(params, {
      onSuccess: () => {
        toast({ variant: 'success', message: 'User created' })
        handleClose()
      },
      onError: (err) => {
        toast({ variant: 'error', message: err instanceof Error ? err.message : 'Failed to create user' })
      },
    })
  }

  return (
    <Dialog open={open} onClose={handleClose} title="Create User">
      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <Input label="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} disabled={createUser.isPending} />
        <Input label="Display Name" value={displayName} onChange={(e) => setDisplayName(e.target.value)} disabled={createUser.isPending} />
        <Input label="Password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} disabled={createUser.isPending} />
        {canAssignAdmin && (
          <div>
            <label className="text-sm text-text-secondary mb-1 block">Role</label>
            <select
              className="w-full rounded-md border border-border bg-surface px-3 py-2 text-sm"
              value={role}
              onChange={(e) => setRole(e.target.value as UserRole)}
              disabled={createUser.isPending}
            >
              <option value="member">Member</option>
              <option value="admin">Admin</option>
            </select>
          </div>
        )}
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={handleClose} disabled={createUser.isPending}>Cancel</Button>
          <Button type="submit" loading={createUser.isPending}>Create User</Button>
        </div>
      </form>
    </Dialog>
  )
}

interface EditUserDialogProps {
  user: UserResponse | null
  onClose: () => void
  canChangeRole: boolean
}

function EditUserDialog({ user, onClose, canChangeRole }: EditUserDialogProps) {
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<UserRole>('member')
  const updateUser = useUpdateUser()
  const { toast } = useToast()

  React.useEffect(() => {
    if (user) {
      setEmail(user.email)
      setDisplayName(user.display_name)
      setPassword('')
      setRole(user.role)
    }
  }, [user])

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!user) return
    const params: UpdateUserParams = {
      email: email.trim(),
      display_name: displayName.trim(),
    }
    if (password) params.password = password
    if (canChangeRole && user.role !== 'root') params.role = role

    updateUser.mutate(
      { userId: user.id, params },
      {
        onSuccess: () => {
          toast({ variant: 'success', message: 'User updated' })
          onClose()
        },
        onError: (err) => {
          toast({ variant: 'error', message: err instanceof Error ? err.message : 'Failed to update user' })
        },
      },
    )
  }

  return (
    <Dialog open={user !== null} onClose={onClose} title="Edit User">
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input label="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} disabled={updateUser.isPending} />
        <Input label="Display Name" value={displayName} onChange={(e) => setDisplayName(e.target.value)} disabled={updateUser.isPending} />
        <Input label="New Password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Leave blank to keep" disabled={updateUser.isPending} />
        {canChangeRole && user?.role !== 'root' && (
          <div>
            <label className="text-sm text-text-secondary mb-1 block">Role</label>
            <select
              className="w-full rounded-md border border-border bg-surface px-3 py-2 text-sm"
              value={role}
              onChange={(e) => setRole(e.target.value as UserRole)}
              disabled={updateUser.isPending}
            >
              <option value="member">Member</option>
              <option value="admin">Admin</option>
            </select>
          </div>
        )}
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={onClose} disabled={updateUser.isPending}>Cancel</Button>
          <Button type="submit" loading={updateUser.isPending}>Save</Button>
        </div>
      </form>
    </Dialog>
  )
}

interface WalletDialogProps {
  user: UserResponse | null
  onClose: () => void
}

function WalletDialog({ user, onClose }: WalletDialogProps) {
  const { data: wallet } = useUserWallet(user?.id ?? null)
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')
  const [mode, setMode] = useState<'add' | 'subtract' | 'set'>('add')
  const adjust = useAdjustUserWallet()
  const { toast } = useToast()

  const current = wallet?.balance ?? 0
  const parsed = parseFloat(amount) || 0
  const preview = useMemo(() => {
    if (mode === 'add') return current + parsed
    if (mode === 'subtract') return current - parsed
    return parsed
  }, [current, parsed, mode])

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!user || parsed === 0) return
    let delta = parsed
    if (mode === 'subtract') delta = -parsed
    if (mode === 'set') delta = parsed - current
    adjust.mutate(
      { userId: user.id, params: { amount: delta, description } },
      {
        onSuccess: () => {
          toast({ variant: 'success', message: 'Wallet adjusted' })
          setAmount('')
          onClose()
        },
        onError: (err) => {
          toast({ variant: 'error', message: err instanceof Error ? err.message : 'Failed to adjust wallet' })
        },
      },
    )
  }

  return (
    <Dialog open={user !== null} onClose={onClose} title="Adjust Wallet">
      <form onSubmit={handleSubmit} className="space-y-4">
        <p className="text-sm text-text-secondary">
          Current balance: <span className="font-medium text-text-primary">{formatCost(current)}</span>
        </p>
        <div className="flex gap-2">
          {(['add', 'subtract', 'set'] as const).map((m) => (
            <Button key={m} type="button" size="sm" variant={mode === m ? 'primary' : 'secondary'} onClick={() => setMode(m)}>
              {m === 'add' ? 'Add' : m === 'subtract' ? 'Subtract' : 'Set'}
            </Button>
          ))}
        </div>
        <Input label="Amount" type="number" step="0.01" value={amount} onChange={(e) => setAmount(e.target.value)} disabled={adjust.isPending} />
        <p className="text-xs text-text-tertiary">Preview: {formatCost(preview)}</p>
        <Input label="Description" value={description} onChange={(e) => setDescription(e.target.value)} disabled={adjust.isPending} />
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose} disabled={adjust.isPending}>Cancel</Button>
          <Button type="submit" loading={adjust.isPending}>Apply</Button>
        </div>
      </form>
    </Dialog>
  )
}

export default function SystemUsersPage() {
  const { data: me } = useMe()
  const [cursor, setCursor] = useState<string | undefined>()
  const [prevCursors, setPrevCursors] = useState<string[]>([])
  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState<UserRole | ''>('')
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [editUser, setEditUser] = useState<UserResponse | null>(null)
  const [walletUser, setWalletUser] = useState<UserResponse | null>(null)
  const [deleteUserId, setDeleteUserId] = useState<string | null>(null)
  const [keysUser, setKeysUser] = useState<UserResponse | null>(null)

  const listParams = useMemo(
    () => ({ cursor, search: search.trim() || undefined, role: roleFilter || undefined }),
    [cursor, search, roleFilter],
  )
  const { data: users, isLoading } = useUsers(listParams)
  const deleteUser = useDeleteUser()
  const manageUser = useManageUser()
  const revokeSessions = useRevokeUserSessions()
  const { toast } = useToast()

  const isAdmin = me?.role === 'admin' || me?.role === 'root' || me?.is_system_admin
  const isRoot = me?.role === 'root'

  if (me && !isAdmin) {
    return <Navigate to="/" replace />
  }

  const allUsers = users?.data ?? []

  const columns: Column<UserResponse>[] = [
    {
      key: 'email',
      header: 'User',
      render: (row) => (
        <div>
          <div className="font-medium text-text-primary">{row.email}</div>
          <div className="text-xs text-text-tertiary">{row.display_name}</div>
        </div>
      ),
    },
    { key: 'role', header: 'Role', render: (row) => roleBadge(row.role) },
    { key: 'status', header: 'Status', render: (row) => statusBadge(row.status) },
    {
      key: 'auth_provider',
      header: 'Auth',
      render: (row) => <span className="text-text-secondary text-sm">{row.auth_provider}</span>,
    },
    {
      key: 'created_at',
      header: 'Created',
      render: (row) => <TimeAgo date={row.created_at} />,
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (row) => (
        <div className="flex items-center justify-end gap-1 flex-wrap">
          <Button variant="ghost" size="sm" className="!px-2 text-xs" onClick={() => setEditUser(row)}>Edit</Button>
          <Button variant="ghost" size="sm" className="!px-2 text-xs" onClick={() => setWalletUser(row)}>Wallet</Button>
          <Button variant="ghost" size="sm" className="!px-2 text-xs" onClick={() => setKeysUser(row)}>Keys</Button>
          {row.status === 'active' && row.role !== 'root' && (
            <Button
              variant="ghost"
              size="sm"
              className="!px-2 text-xs"
              onClick={() =>
                manageUser.mutate(
                  { userId: row.id, action: 'disable' },
                  { onSuccess: () => toast({ variant: 'success', message: 'User disabled' }) },
                )
              }
            >
              Disable
            </Button>
          )}
          {row.status === 'disabled' && (
            <Button
              variant="ghost"
              size="sm"
              className="!px-2 text-xs"
              onClick={() =>
                manageUser.mutate(
                  { userId: row.id, action: 'enable' },
                  { onSuccess: () => toast({ variant: 'success', message: 'User enabled' }) },
                )
              }
            >
              Enable
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            className="!px-2 text-xs"
            onClick={() =>
              revokeSessions.mutate(row.id, {
                onSuccess: () => toast({ variant: 'success', message: 'Sessions revoked' }),
              })
            }
          >
            Revoke sessions
          </Button>
          <Link to={`/analytics/logs?user_id=${row.id}`} className="text-xs text-accent hover:underline px-2">Logs</Link>
          {row.id !== me?.id && row.role !== 'root' && (
            <Button variant="ghost" size="sm" className="!px-2 text-xs text-error" onClick={() => setDeleteUserId(row.id)}>
              Delete
            </Button>
          )}
        </div>
      ),
    },
  ]

  function handleDelete() {
    if (!deleteUserId) return
    deleteUser.mutate(deleteUserId, {
      onSuccess: () => {
        toast({ variant: 'success', message: 'User deleted' })
        setDeleteUserId(null)
      },
      onError: (err) => {
        toast({ variant: 'error', message: err instanceof Error ? err.message : 'Failed to delete user' })
        setDeleteUserId(null)
      },
    })
  }

  return (
    <>
      <PageHeader
        title="Users"
        description="Manage customer accounts, wallets, and access"
        actions={<Button onClick={() => setShowCreateDialog(true)}>Create User</Button>}
      />

      <div className="flex flex-wrap gap-3 mb-4">
        <Input
          placeholder="Search email, name, or ID..."
          value={search}
          onChange={(e) => {
            setSearch(e.target.value)
            setCursor(undefined)
            setPrevCursors([])
          }}
          className="max-w-xs"
        />
        <select
          className="rounded-md border border-border bg-surface px-3 py-2 text-sm"
          value={roleFilter}
          onChange={(e) => {
            setRoleFilter(e.target.value as UserRole | '')
            setCursor(undefined)
            setPrevCursors([])
          }}
        >
          <option value="">All roles</option>
          <option value="member">Member</option>
          <option value="admin">Admin</option>
          <option value="root">Root</option>
        </select>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-4 gap-4 mb-6">
        <StatCard label="On page" value={isLoading ? '—' : allUsers.length} />
        <StatCard label="Members" value={isLoading ? '—' : allUsers.filter((u) => u.role === 'member').length} />
        <StatCard label="Admins" value={isLoading ? '—' : allUsers.filter((u) => u.role === 'admin').length} />
        <StatCard label="Disabled" value={isLoading ? '—' : allUsers.filter((u) => u.status === 'disabled').length} />
      </div>

      <Table<UserResponse>
        columns={columns}
        data={allUsers}
        keyExtractor={(row) => row.id}
        loading={isLoading}
        emptyMessage="No users found"
        pagination={{
          cursor: cursor ?? null,
          hasMore: users?.has_more ?? false,
          hasPrevious: prevCursors.length > 0,
          onNext: () => {
            if (users?.next_cursor) {
              setPrevCursors((prev) => [...prev, cursor ?? ''])
              setCursor(users.next_cursor)
            }
          },
          onPrevious: () => {
            const prev = prevCursors[prevCursors.length - 1]
            setPrevCursors((p) => p.slice(0, -1))
            setCursor(prev || undefined)
          },
        }}
      />

      <CreateUserDialog open={showCreateDialog} onClose={() => setShowCreateDialog(false)} canAssignAdmin={isRoot} />
      <EditUserDialog user={editUser} onClose={() => setEditUser(null)} canChangeRole={isRoot} />
      <WalletDialog user={walletUser} onClose={() => setWalletUser(null)} />

      <ConfirmDialog
        open={deleteUserId !== null}
        onClose={() => setDeleteUserId(null)}
        onConfirm={handleDelete}
        title="Delete User"
        description="Soft-delete this user? They will lose access immediately."
        confirmLabel="Delete"
        loading={deleteUser.isPending}
      />

      {keysUser !== null && (
        <UserKeysDialog user={keysUser} onClose={() => setKeysUser(null)} />
      )}
    </>
  )
}