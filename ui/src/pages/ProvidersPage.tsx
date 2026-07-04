import { useState } from 'react'
import { PageHeader } from '../components/ui/PageHeader'
import { Table } from '../components/ui/Table'
import type { Column } from '../components/ui/Table'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { Dialog, ConfirmDialog } from '../components/ui/Dialog'
import {
  useProviders,
  useProviderPresets,
  useCreateProvider,
  useUpdateProvider,
  useDeleteProvider,
} from '../hooks/useProviders'
import type { ProviderItem, ProviderPreset } from '../hooks/useProviders'
import { useToast } from '../hooks/useToast'
import { useTranslation } from '../lib/i18n'
import { IconPicker } from '../components/ui/IconPicker'
import { BrandIcon } from '../components/ui/BrandIcon'
import { defaultIconForSlug } from '../lib/provider-icons'

const PROTOCOL_OPTIONS = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'azure', label: 'Azure' },
  { value: 'vertex', label: 'Vertex' },
  { value: 'vllm', label: 'vLLM' },
  { value: 'ollama', label: 'Ollama' },
  { value: 'custom', label: 'Custom' },
]

export default function ProvidersPage() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data, isLoading } = useProviders()
  const { data: presetsData } = useProviderPresets()
  const createProvider = useCreateProvider()
  const updateProvider = useUpdateProvider()
  const deleteProvider = useDeleteProvider()

  const [dialogOpen, setDialogOpen] = useState(false)
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

  function openCreate(preset?: ProviderPreset) {
    setEditing(null)
    setName(preset?.name ?? '')
    setSlug(preset?.id ?? '')
    setProtocol(preset?.protocol ?? 'openai')
    setBaseUrl(preset?.base_url ?? '')
    setApiKey('')
    setLogo(preset?.logo ?? '')
    setContact('')
    setNotes('')
    setDialogOpen(true)
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
    setDialogOpen(true)
  }

  function submit() {
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
      header: 'Protocol',
      render: (row) => <span className="text-text-secondary text-xs">{row.protocol}</span>,
    },
    {
      key: 'base_url',
      header: 'Base URL',
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
      <PageHeader
        title="Providers"
        description="Upstream sources — connection defaults inherited by model channels."
      />
      {(presetsData?.data?.length ?? 0) > 0 && (
        <section className="mb-8">
          <h2 className="text-sm font-medium text-text-secondary mb-3">Quick add from preset</h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
            {presetsData!.data.map((preset) => (
              <button
                key={preset.id}
                type="button"
                onClick={() => openCreate(preset)}
                className="flex flex-col items-center gap-2 rounded-xl border border-border bg-bg-secondary p-4 hover:border-accent/40 hover:bg-bg-tertiary transition-colors text-left"
              >
                <BrandIcon logo={preset.logo} slug={preset.id} protocol={preset.protocol} size={32} />
                <span className="text-xs font-medium text-text-primary text-center leading-tight">
                  {preset.name}
                </span>
              </button>
            ))}
          </div>
        </section>
      )}
      <div className="mb-4 flex justify-end">
        <Button onClick={() => openCreate()}>{t('marketplace.add_provider')}</Button>
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
            <Button onClick={submit} loading={createProvider.isPending || updateProvider.isPending}>
              {t('common.save')}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Input label={t('marketplace.provider_name')} required value={name} onChange={(e) => setName(e.target.value)} />
          <Input
            label="Slug"
            value={slug}
            onChange={(e) => {
              const next = e.target.value
              setSlug(next)
              if (!logo.trim() && next.trim()) {
                setLogo(defaultIconForSlug(next, protocol))
              }
            }}
            placeholder="openai, ds, or"
          />
          <Select label="Protocol" options={PROTOCOL_OPTIONS} value={protocol} onChange={setProtocol} />
          <Input label="Base URL" value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} placeholder="https://api.openai.com/v1" />
          <Input
            label="API Key"
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder={editing?.has_api_key ? '••••••••  (leave blank to keep)' : 'sk-...'}
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