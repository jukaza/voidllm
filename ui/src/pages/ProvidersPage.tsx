import { useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { Table } from '../components/ui/Table'
import type { Column } from '../components/ui/Table'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { ConfirmDialog } from '../components/ui/Dialog'
import {
  useProviders,
  useUpdateProvider,
  useDeleteProvider,
} from '../hooks/useProviders'
import type { ProviderItem } from '../hooks/useProviders'
import { useToast } from '../hooks/useToast'
import { useTranslation } from '../lib/i18n'
import { BrandIcon } from '../components/ui/BrandIcon'

export default function ProvidersPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { toast } = useToast()
  const { data, isLoading } = useProviders()
  const updateProvider = useUpdateProvider()
  const deleteProvider = useDeleteProvider()

  const [deleting, setDeleting] = useState<ProviderItem | null>(null)

  function toggleStatus(p: ProviderItem) {
    updateProvider.mutate(
      { id: p.id, status: p.status === 'active' ? 'paused' : 'active' },
      { onError: (e) => toast({ variant: 'error', message: e.message }) },
    )
  }

  const columns: Column<ProviderItem>[] = useMemo(
    () => [
      {
        key: 'logo',
        header: '',
        width: '40px',
        render: (row) => (
          <BrandIcon logo={row.logo} slug={row.slug} protocol={row.protocol} size={22} />
        ),
      },
      {
        key: 'name',
        header: t('marketplace.col_provider'),
        render: (row) => (
          <div>
            <Link
              to={`/providers/${row.id}`}
              className="font-medium text-accent no-underline hover:underline"
            >
              {row.name}
            </Link>
            {row.slug && (
              <span className="ml-2 text-xs text-text-tertiary font-mono">{row.slug}</span>
            )}
          </div>
        ),
      },
      {
        key: 'protocol',
        header: t('common.protocol'),
        render: (row) => <span className="text-text-secondary text-xs">{row.protocol}</span>,
      },
      {
        key: 'base_url',
        header: t('common.endpoint'),
        render: (row) => (
          <span className="text-text-tertiary text-xs truncate max-w-[200px] inline-block">
            {row.base_url || '—'}
          </span>
        ),
      },
      {
        key: 'status',
        header: t('wallet.col_status'),
        render: (row) => (
          <Badge variant={row.status === 'active' ? 'success' : 'muted'}>
            {row.status === 'active' ? t('providers.status_active') : t('providers.status_paused')}
          </Badge>
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
            <Button size="sm" variant="secondary" onClick={() => navigate(`/providers/${row.id}`)}>
              {t('common.open')}
            </Button>
            <Button size="sm" variant="destructive" onClick={() => setDeleting(row)}>
              {t('common.delete')}
            </Button>
          </div>
        ),
      },
    ],
    [t, navigate],
  )

  return (
    <>
      <PageHeader title={t('providers.title')} description={t('providers.desc_new')} />
      <div className="mb-4 flex justify-end gap-2">
        <Button variant="secondary" onClick={() => navigate('/providers/new?manual=1')}>
          {t('providers.manual_entry')}
        </Button>
        <Button onClick={() => navigate('/providers/new')}>{t('providers.add_provider')}</Button>
      </div>
      <Table
        columns={columns}
        data={data?.data ?? []}
        keyExtractor={(r) => r.id}
        loading={isLoading}
        emptyMessage={`${t('marketplace.no_providers')} ${t('providers.empty_hint_new')}`}
        compact
      />

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