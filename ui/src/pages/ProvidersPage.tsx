import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { Table } from '../components/ui/Table'
import type { Column } from '../components/ui/Table'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { Dialog, ConfirmDialog } from '../components/ui/Dialog'
import { ProviderSetupDrawer } from '../components/providers/ProviderSetupDrawer'
import {
  useProviders,
  useCreateProvider,
  useUpdateProvider,
  useDeleteProvider,
} from '../hooks/useProviders'
import type { ProviderItem } from '../hooks/useProviders'
import { useToast } from '../hooks/useToast'
import { useTranslation } from '../lib/i18n'
import { IconPicker } from '../components/ui/IconPicker'
import { BrandIcon } from '../components/ui/BrandIcon'
import { defaultIconForSlug } from '../lib/provider-icons'
import { getConnectionProfile } from '../lib/connection-profiles'
import { endpointHintKey } from '../lib/endpoint-hints'

export default function ProvidersPage() {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()

  const PROTOCOL_OPTIONS = useMemo(
    () => [
      { value: 'openai', label: t('providers.protocol_openai') },
      { value: 'anthropic', label: t('providers.protocol_anthropic') },
      { value: 'gemini', label: t('providers.protocol_gemini') },
      { value: 'azure', label: t('providers.protocol_azure') },
      { value: 'vertex', label: t('providers.protocol_vertex') },
      { value: 'vllm', label: t('providers.protocol_vllm') },
      { value: 'custom', label: t('providers.protocol_custom') },
    ],
    [t],
  )
  const { toast } = useToast()
  const { data, isLoading } = useProviders()
  const createProvider = useCreateProvider()
  const updateProvider = useUpdateProvider()
  const deleteProvider = useDeleteProvider()

  const setupOpen = searchParams.get('add') === '1'
  const initialPresetId = searchParams.get('preset') ?? undefined

  const [manualOpen, setManualOpen] = useState(false)
  const [editing, setEditing] = useState<ProviderItem | null>(null)
  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [protocol, setProtocol] = useState('openai')
  const [baseUrl, setBaseUrl] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [logo, setLogo] = useState('')
  const [contact, setContact] = useState('')
  const [notes, setNotes] = useState('')
  const [deleting, setDeleting] = useState<ProviderItem | null>(null)

  const profile = getConnectionProfile(protocol)

  function openSetupDrawer() {
    setSearchParams({ add: '1' })
  }

  function closeSetupDrawer() {
    setSearchParams({})
  }

  function openCreateManual() {
    setSearchParams({ add: '1', preset: 'custom' })
  }

  function openEdit(p: ProviderItem) {
    setEditing(p)
    setName(p.name)
    setSlug(p.slug ?? '')
    setProtocol(p.protocol || 'openai')
    setBaseUrl(p.base_url ?? '')
    setApiKey('')
    setLogo(p.logo ?? '')
    setContact(p.contact_info)
    setNotes(p.notes)
    setManualOpen(true)
  }

  function submitManual() {
    if (!name.trim()) {
      toast({ variant: 'error', message: t('marketplace.provider_name_required') })
      return
    }
    const params = {
      name: name.trim(),
      slug: slug.trim() || undefined,
      protocol,
      base_url: baseUrl.trim() || undefined,
      api_key: apiKey.trim() || undefined,
      logo: logo.trim() || undefined,
      contact_info: contact,
      notes,
    }
    const opts = {
      onSuccess: () => {
        toast({ variant: 'success', message: t('common.saved') })
        setManualOpen(false)
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
          <span className="font-medium">{row.name}</span>
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
      <PageHeader title={t('providers.title')} description={t('providers.desc')} />
      <div className="mb-4 flex justify-end gap-2">
        <Button variant="secondary" onClick={openCreateManual}>
          {t('providers.manual_entry')}
        </Button>
        <Button onClick={openSetupDrawer}>{t('providers.add_provider')}</Button>
      </div>
      <Table
        columns={columns}
        data={data?.data ?? []}
        keyExtractor={(r) => r.id}
        loading={isLoading}
        emptyMessage={`${t('marketplace.no_providers')} ${t('providers.empty_hint')}`}
        compact
      />

      <ProviderSetupDrawer
        open={setupOpen}
        onClose={closeSetupDrawer}
        initialPresetId={initialPresetId}
      />

      <Dialog
        open={manualOpen}
        onClose={() => setManualOpen(false)}
        title={editing ? t('marketplace.edit_provider') : t('marketplace.add_provider')}
        footer={
          <>
            <Button variant="secondary" onClick={() => setManualOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button onClick={submitManual} loading={createProvider.isPending || updateProvider.isPending}>
              {t('common.save')}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Input label={t('marketplace.provider_name')} required value={name} onChange={(e) => setName(e.target.value)} />
          <Input
            label={t('common.slug')}
            value={slug}
            onChange={(e) => {
              const next = e.target.value
              setSlug(next)
              if (!logo.trim() && next.trim()) {
                setLogo(defaultIconForSlug(next, protocol))
              }
            }}
            placeholder={t('providers.slug_placeholder')}
          />
          <Select label={t('common.protocol')} options={PROTOCOL_OPTIONS} value={protocol} onChange={setProtocol} />
          <p className="text-xs text-text-tertiary -mt-2">{t('providers.protocol_hint')}</p>
          <Input
            label={t('common.endpoint')}
            value={baseUrl}
            onChange={(e) => setBaseUrl(e.target.value)}
            placeholder={profile.baseUrlPlaceholder}
            description={t(endpointHintKey(protocol))}
          />
          <Input
            label={profile.keyLabel === 'API Key' ? t('common.api_key') : profile.keyLabel}
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder={editing?.has_api_key ? t('providers.key_keep') : profile.keyPlaceholder}
          />
          <IconPicker value={logo} onChange={setLogo} />
          <Input label={t('marketplace.provider_contact')} value={contact} onChange={(e) => setContact(e.target.value)} />
          <Input label={t('marketplace.provider_notes')} value={notes} onChange={(e) => setNotes(e.target.value)} />
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