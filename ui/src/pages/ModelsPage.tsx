import React, { useState, useMemo } from 'react'
import { PageHeader } from '../components/ui/PageHeader'
import { Table } from '../components/ui/Table'
import type { Column } from '../components/ui/Table'
import { Dialog, ConfirmDialog } from '../components/ui/Dialog'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import type { SelectOption } from '../components/ui/Select'
import { Toggle } from '../components/ui/Toggle'
import { StatCard } from '../components/ui/StatCard'

import {
  useModels,
  useCreateModel,
  useUpdateModel,
  useDeleteModel,
  useToggleModel,
  useCreateDeployment,
  useUpdateDeployment,
  useDeleteDeployment,
} from '../hooks/useModels'
import type { ModelResponse, DeploymentResponse, CreateModelParams, UpdateModelParams } from '../hooks/useModels'
import { useModelHealth } from '../hooks/useModelHealth'
import type { ModelHealthInfo } from '../hooks/useModelHealth'
import { useToast } from '../hooks/useToast'
import { providerBadgeVariant, isKnownProvider } from '../lib/providers'
import { cn } from '../lib/utils'
import { useProviders } from '../hooks/useProviders'
import type { ProviderItem } from '../hooks/useProviders'
import { getConnectionProfile } from '../lib/connection-profiles'
import { endpointHintKey, wireMismatchKey } from '../lib/endpoint-hints'
import { BrandIcon } from '../components/ui/BrandIcon'
import { useTranslation } from '../lib/i18n'

// ---------------------------------------------------------------------------
// Module-level constants
// ---------------------------------------------------------------------------

const providerLabels: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  gemini: 'Gemini',
  azure: 'Azure',
  vertex: 'Vertex AI',
  vllm: 'vLLM',
  ollama: 'Ollama',
  custom: 'Custom',
}

const PROVIDER_OPTIONS = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'azure', label: 'Azure' },
  { value: 'vertex', label: 'Vertex AI' },
  { value: 'vllm', label: 'vLLM' },
  { value: 'ollama', label: 'Ollama' },
  { value: 'custom', label: 'Custom' },
]

const MODEL_TYPE_OPTIONS = [
  { value: 'chat', label: 'Chat' },
  { value: 'embedding', label: 'Embedding' },
  { value: 'reranking', label: 'Reranking' },
  { value: 'completion', label: 'Completion' },
  { value: 'image', label: 'Image Generation' },
  { value: 'audio_transcription', label: 'Audio Transcription' },
  { value: 'tts', label: 'Text to Speech' },
]

// Tri-state: 'default' maps to undefined (omit from request), 'true' / 'false' map to boolean.
const PII_FILTER_OPTIONS = [
  {
    value: 'default',
    label: 'Default (network-based)',
    description: 'Private endpoints pass through; public endpoints are anonymized.',
  },
  {
    value: 'true',
    label: 'Always anonymize',
    description: 'PII is stripped from every request to this model.',
  },
  {
    value: 'false',
    label: 'Never (trusted endpoint)',
    description: 'No anonymization — use only for private, trusted endpoints.',
  },
]

/** Converts the PII filter Select string value back to the API boolean. */
function piiFilterToParam(value: string): boolean | undefined {
  if (value === 'true') return true
  if (value === 'false') return false
  return undefined
}

/** Converts an API boolean (or undefined/null) to the Select string value. */
function piiFilterFromResponse(value: boolean | undefined | null): string {
  if (value === true) return 'true'
  if (value === false) return 'false'
  return 'default'
}

const typeLabels: Record<string, string> = {
  chat: 'Chat',
  embedding: 'Embedding',
  reranking: 'Reranking',
  completion: 'Completion',
  image: 'Image',
  audio_transcription: 'Audio',
  tts: 'TTS',
}

const typeBadgeVariant: Record<string, 'default' | 'info' | 'muted' | 'success' | 'warning'> = {
  chat: 'default',
  embedding: 'info',
  reranking: 'info',
  completion: 'muted',
  image: 'success',
  audio_transcription: 'warning',
  tts: 'warning',
}

const BASE_URL_PLACEHOLDERS: Record<string, string> = {
  openai: 'https://api.openai.com/v1',
  anthropic: 'https://api.anthropic.com',
  gemini: 'https://generativelanguage.googleapis.com',
  azure: 'https://<resource>.openai.azure.com',
  vertex: 'https://us-central1-aiplatform.googleapis.com',
  vllm: 'http://localhost:8000/v1',
  ollama: 'http://localhost:11434/v1',
  custom: 'https://your-endpoint/v1',
}

// ---------------------------------------------------------------------------
// Deployment form types + connection fields (provider inherit vs manual)
// ---------------------------------------------------------------------------

interface DeploymentFormEntry {
  name: string
  providerId: string
  provider: string
  baseUrl: string
  apiKey: string
  upstreamModel: string
  azureDeployment: string
  azureApiVersion: string
  weight: number
  priority: number
}

interface DeploymentConnectionFieldsProps {
  entry: DeploymentFormEntry
  onChange: (patch: Partial<DeploymentFormEntry>) => void
  errors: { provider?: string; base_url?: string }
  providerOptions: SelectOption[]
  providers: ProviderItem[]
  isPending: boolean
  isEdit?: boolean
}

function DeploymentConnectionFields({
  entry,
  onChange,
  errors,
  providerOptions,
  providers,
  isPending,
  isEdit = false,
}: DeploymentConnectionFieldsProps) {
  const { t } = useTranslation()
  const inherits = Boolean(entry.providerId)
  const selectedProvider = providers.find((p) => p.id === entry.providerId)
  const effectiveProtocol = entry.provider || selectedProvider?.protocol || ''
  const isAzure = effectiveProtocol === 'azure'
  const wireWarn = wireMismatchKey(effectiveProtocol, entry.upstreamModel)

  const protocolOptions = useMemo<SelectOption[]>(
    () => [
      { value: 'openai', label: t('providers.protocol_openai') },
      { value: 'anthropic', label: t('providers.protocol_anthropic') },
      { value: 'gemini', label: t('providers.protocol_gemini') },
      { value: 'azure', label: t('providers.protocol_azure') },
      { value: 'vertex', label: t('providers.protocol_vertex') },
      { value: 'vllm', label: t('providers.protocol_vllm') },
      { value: 'ollama', label: 'Ollama' },
      { value: 'custom', label: t('providers.protocol_custom') },
    ],
    [t],
  )

  function onProviderSourceChange(providerId: string) {
    const patch: Partial<DeploymentFormEntry> = { providerId }
    if (providerId) {
      patch.provider = ''
      patch.baseUrl = ''
      patch.apiKey = ''
    }
    onChange(patch)
  }

  return (
    <>
      <Select
        label={t('connection.provider_source')}
        options={providerOptions}
        value={entry.providerId}
        onChange={onProviderSourceChange}
        error={errors.provider}
        disabled={isPending}
      />
      {inherits ? (
        <p className="text-xs text-text-tertiary -mt-2">
          {t('connection.inherited_from')}{' '}
          <span className="text-text-secondary font-medium">{selectedProvider?.name ?? 'provider'}</span>
          {selectedProvider?.protocol ? (
            <>
              {' '}
              · {t('connection.inherited_protocol')}{' '}
              <code className="text-[11px]">{selectedProvider.protocol}</code>
            </>
          ) : null}
          {selectedProvider?.base_url ? (
            <>
              {' '}
              · <code className="text-[11px]">{selectedProvider.base_url}</code>
            </>
          ) : null}
          {selectedProvider?.has_api_key ? ` ${t('connection.api_key_on_file')}` : null}
        </p>
      ) : null}
      {inherits ? (
        <>
          <Select
            label={t('connection.protocol_override')}
            options={[{ value: '', label: t('connection.inherit_from_provider') }, ...protocolOptions]}
            value={entry.provider}
            onChange={(v) => onChange({ provider: v })}
            disabled={isPending}
          />
          <p className="text-xs text-text-tertiary -mt-2">{t('connection.protocol_override_hint')}</p>
        </>
      ) : (
        <>
          <Select
            label={t('common.protocol')}
            options={protocolOptions}
            value={entry.provider}
            onChange={(v) => onChange({ provider: v })}
            error={errors.provider}
            disabled={isPending}
          />
          <p className="text-xs text-text-tertiary -mt-2">{t('providers.protocol_hint')}</p>
        </>
      )}
      <Input
        label={t('connection.upstream_model')}
        value={entry.upstreamModel}
        onChange={(e) => onChange({ upstreamModel: e.target.value })}
        placeholder={t('connection.upstream_placeholder')}
        disabled={isPending}
      />
      {entry.upstreamModel.trim().toLowerCase().startsWith('claude-') ? (
        <p className="text-xs text-text-tertiary -mt-2">{t('connection.claude_upstream_hint')}</p>
      ) : null}
      {wireWarn ? (
        <p className="text-xs text-amber-600 dark:text-amber-400 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2">
          {t(wireWarn)}
        </p>
      ) : null}
      {!inherits && (
        <>
          <Input
            label={t('common.endpoint')}
            value={entry.baseUrl}
            onChange={(e) => onChange({ baseUrl: e.target.value })}
            placeholder={BASE_URL_PLACEHOLDERS[entry.provider] ?? 'https://'}
            error={errors.base_url}
            description={
              isEdit ? t('connection.leave_empty_keep') : t(endpointHintKey(entry.provider))
            }
            disabled={isPending}
          />
          <Input
            label={t('common.api_key')}
            type="password"
            value={entry.apiKey}
            onChange={(e) => onChange({ apiKey: e.target.value })}
            placeholder={isEdit ? t('connection.leave_empty_keep') : getConnectionProfile(entry.provider).keyPlaceholder}
            description={isEdit ? t('connection.leave_empty_keep_key') : t('connection.encrypted_at_rest')}
            disabled={isPending}
          />
        </>
      )}
      {isAzure && (
        <>
          <Input
            label={t('connection.azure_deployment')}
            value={entry.azureDeployment}
            onChange={(e) => onChange({ azureDeployment: e.target.value })}
            placeholder="e.g. gpt-4o-deployment"
            disabled={isPending}
          />
          <Input
            label={t('connection.azure_api_version')}
            value={entry.azureApiVersion}
            onChange={(e) => onChange({ azureApiVersion: e.target.value })}
            placeholder="e.g. 2024-02-01"
            disabled={isPending}
          />
        </>
      )}
    </>
  )
}

// ---------------------------------------------------------------------------
// Icons
// ---------------------------------------------------------------------------

function IconLayers() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M12 2 2 7l10 5 10-5-10-5z" />
      <path d="M2 17l10 5 10-5" />
      <path d="M2 12l10 5 10-5" />
    </svg>
  )
}

function IconActivity() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
    </svg>
  )
}

function IconPauseCircle() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="10" />
      <line x1="10" y1="15" x2="10" y2="9" />
      <line x1="14" y1="15" x2="14" y2="9" />
    </svg>
  )
}

function IconPencil() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
    </svg>
  )
}

function IconTrash() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polyline points="3 6 5 6 21 6" />
      <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
      <path d="M10 11v6" />
      <path d="M14 11v6" />
      <path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// HealthBadge
// ---------------------------------------------------------------------------

const healthConfig: Record<
  ModelHealthInfo['status'],
  { dotClass: string; label: string }
> = {
  healthy:   { dotClass: 'bg-success',         label: 'Healthy' },
  degraded:  { dotClass: 'bg-warning',          label: 'Degraded' },
  unhealthy: { dotClass: 'bg-error',            label: 'Unhealthy' },
  unknown:   { dotClass: 'bg-text-tertiary',    label: 'Unknown' },
}

interface HealthBadgeProps {
  info: ModelHealthInfo | undefined
}

function HealthBadge({ info }: HealthBadgeProps) {
  if (info === undefined) {
    return (
      <div className="flex items-center gap-1.5">
        <span className="w-2 h-2 rounded-full bg-text-tertiary opacity-40 shrink-0" aria-hidden="true" />
        <span className="text-text-tertiary text-sm">Unknown</span>
      </div>
    )
  }

  const { dotClass, label } = healthConfig[info.status]

  return (
    <div className="flex items-center gap-2">
      <div className="flex items-center gap-1.5">
        <span className={cn('w-2 h-2 rounded-full shrink-0', dotClass)} aria-hidden="true" />
        <span className="text-text-secondary text-sm">{label}</span>
      </div>
      {info.latency_ms > 0 && (
        <span className="text-text-tertiary text-xs tabular-nums">{info.latency_ms}ms</span>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// DeploymentDialog
// ---------------------------------------------------------------------------

interface DeploymentDialogProps {
  modelId: string
  deployment: DeploymentResponse | null
  onClose: () => void
}

interface DeploymentFormErrors {
  name?: string
  provider?: string
  base_url?: string
}

function DeploymentDialog({ modelId, deployment, onClose }: DeploymentDialogProps) {
  const { t } = useTranslation()
  const isEdit = deployment !== null

  const [name, setName] = useState(deployment?.name ?? '')
  const [entry, setEntry] = useState<DeploymentFormEntry>(() => ({
    name: deployment?.name ?? '',
    providerId: deployment?.provider_id ?? '',
    provider: deployment?.provider_id ? '' : (deployment?.provider ?? 'openai'),
    baseUrl: deployment?.provider_id ? '' : (deployment?.base_url ?? ''),
    apiKey: '',
    upstreamModel: deployment?.upstream_model ?? '',
    azureDeployment: deployment?.azure_deployment ?? '',
    azureApiVersion: deployment?.azure_api_version ?? '',
    weight: deployment?.weight ?? 1,
    priority: deployment?.priority ?? 0,
  }))
  const [errors, setErrors] = useState<DeploymentFormErrors>({})

  const createDeployment = useCreateDeployment()
  const updateDeployment = useUpdateDeployment()
  const { toast } = useToast()
  const { data: providersData } = useProviders()

  const providers = providersData?.data ?? []
  const providerOptions: SelectOption[] = useMemo(() => {
    return [
      { value: '', label: t('connection.manual_connection') },
      ...providers.map((p) => ({
        value: p.id,
        label: p.slug ? `${p.name} (${p.slug})` : p.name,
      })),
    ]
  }, [providers, t])

  const isPending = createDeployment.isPending || updateDeployment.isPending

  function patchEntry(patch: Partial<DeploymentFormEntry>) {
    setEntry((prev) => ({ ...prev, ...patch }))
  }

  function handleClose() {
    setName('')
    setEntry(emptyDeploymentEntry())
    setErrors({})
    onClose()
  }

  function validate(): boolean {
    const next: DeploymentFormErrors = {}
    if (!name.trim()) next.name = 'Name is required'
    if (!entry.providerId && !entry.provider) {
      next.provider = t('connection.select_or_protocol')
    }
    if (!entry.providerId && !isEdit && !entry.baseUrl.trim()) {
      next.base_url = t('connection.base_url_required')
    }
    setErrors(next)
    return Object.keys(next).length === 0
  }

  function handleSubmit(e: React.MouseEvent) {
    e.preventDefault()
    if (!validate()) return

    const parsedWeight = entry.weight
    const parsedPriority = entry.priority
    const effectiveProtocol = entry.provider || undefined

    if (isEdit) {
      const params: Record<string, unknown> = {}
      if (name.trim() !== deployment.name) params.name = name.trim()
      if (effectiveProtocol && effectiveProtocol !== deployment.provider) {
        params.provider = effectiveProtocol
      }
      if (!entry.providerId && entry.baseUrl.trim() && entry.baseUrl.trim() !== deployment.base_url) {
        params.base_url = entry.baseUrl.trim()
      }
      if (entry.apiKey.trim()) params.api_key = entry.apiKey.trim()
      const providerIdChanged = (entry.providerId || null) !== (deployment.provider_id ?? null)
      if (providerIdChanged) {
        params.provider_id = entry.providerId || null
      }
      if (entry.upstreamModel.trim() !== (deployment.upstream_model ?? '')) {
        params.upstream_model = entry.upstreamModel.trim() || undefined
      }
      const isAzure = (effectiveProtocol || deployment.provider) === 'azure'
      if (isAzure && entry.azureDeployment.trim() !== (deployment.azure_deployment ?? '')) {
        params.azure_deployment = entry.azureDeployment.trim() || undefined
      }
      if (isAzure && entry.azureApiVersion.trim() !== (deployment.azure_api_version ?? '')) {
        params.azure_api_version = entry.azureApiVersion.trim() || undefined
      }
      if (parsedWeight !== deployment.weight) params.weight = parsedWeight
      if (parsedPriority !== deployment.priority) params.priority = parsedPriority

      updateDeployment.mutate(
        { modelId, deploymentId: deployment.id, params },
        {
          onSuccess: () => {
            toast({ variant: 'success', message: 'Deployment updated' })
            handleClose()
          },
          onError: (err) => {
            toast({ variant: 'error', message: err instanceof Error ? err.message : 'Failed to update deployment' })
          },
        },
      )
    } else {
      createDeployment.mutate(
        {
          modelId,
          params: {
            name: name.trim(),
            provider: effectiveProtocol,
            base_url: entry.providerId ? undefined : entry.baseUrl.trim() || undefined,
            api_key: entry.providerId ? undefined : entry.apiKey.trim() || undefined,
            provider_id: entry.providerId || undefined,
            upstream_model: entry.upstreamModel.trim() || undefined,
            azure_deployment:
              effectiveProtocol === 'azure' && entry.azureDeployment.trim()
                ? entry.azureDeployment.trim()
                : undefined,
            azure_api_version:
              effectiveProtocol === 'azure' && entry.azureApiVersion.trim()
                ? entry.azureApiVersion.trim()
                : undefined,
            weight: parsedWeight,
            priority: parsedPriority,
          },
        },
        {
          onSuccess: () => {
            toast({ variant: 'success', message: 'Deployment added' })
            handleClose()
          },
          onError: (err) => {
            toast({ variant: 'error', message: err instanceof Error ? err.message : 'Failed to add deployment' })
          },
        },
      )
    }
  }

  return (
    <Dialog open onClose={handleClose} title={isEdit ? t('models.edit_deployment') : t('models.add_deployment')}>
      <div className="space-y-4">
        <Input
          label={t('common.name')}
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. primary"
          error={errors.name}
          disabled={isPending}
        />
        <DeploymentConnectionFields
          entry={entry}
          onChange={patchEntry}
          errors={errors}
          providerOptions={providerOptions}
          providers={providers}
          isPending={isPending}
          isEdit={isEdit}
        />
        <div className="grid grid-cols-2 gap-4">
          <Input
            label={t('common.weight')}
            type="number"
            value={String(entry.weight)}
            onChange={(e) => patchEntry({ weight: parseInt(e.target.value, 10) || 1 })}
            placeholder="1"
            disabled={isPending}
          />
          <Input
            label={t('common.priority')}
            type="number"
            value={String(entry.priority)}
            onChange={(e) => patchEntry({ priority: parseInt(e.target.value, 10) || 0 })}
            placeholder="0"
            disabled={isPending}
          />
        </div>
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={handleClose} disabled={isPending}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSubmit} loading={isPending}>
            {isEdit ? t('models.save_changes') : t('models.add_deployment')}
          </Button>
        </div>
      </div>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// CreateModelDialog
// ---------------------------------------------------------------------------

interface CreateModelDialogProps {
  open: boolean
  onClose: () => void
}

interface FormErrors {
  name?: string
  provider?: string
  base_url?: string
  deployments?: string
}

const emptyDeploymentEntry = (): DeploymentFormEntry => ({
  name: '',
  providerId: '',
  provider: '',
  baseUrl: '',
  apiKey: '',
  upstreamModel: '',
  azureDeployment: '',
  azureApiVersion: '',
  weight: 1,
  priority: 0,
})

interface InlineDeploymentFormErrors {
  name?: string
  provider?: string
  base_url?: string
}

function CreateModelDialog({ open, onClose }: CreateModelDialogProps) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [type, setType] = useState('chat')
  const [aliases, setAliases] = useState('')
  const [maxContextTokens, setMaxContextTokens] = useState('')
  const [inputPricePer1m, setInputPricePer1m] = useState('')
  const [outputPricePer1m, setOutputPricePer1m] = useState('')
  const [timeout, setTimeout] = useState('')
  const [piiFilter, setPiiFilter] = useState('default')
  const [billPerToken, setBillPerToken] = useState(true)
  const [billPerRequest, setBillPerRequest] = useState(false)
  const [sellInputPer1m, setSellInputPer1m] = useState('')
  const [sellOutputPer1m, setSellOutputPer1m] = useState('')
  const [sellCachedInputPer1m, setSellCachedInputPer1m] = useState('')
  const [sellPerRequest, setSellPerRequest] = useState('')

  // Channel routing uses priority + round-robin within same priority.
  const strategy = 'priority'
  const [maxRetries, setMaxRetries] = useState('')
  const [fallbackModelName, setFallbackModelName] = useState('')
  const [deployments, setDeployments] = useState<DeploymentFormEntry[]>([])
  const [showDeploymentForm, setShowDeploymentForm] = useState(false)
  const [editingDeployment, setEditingDeployment] = useState<number | null>(null)
  const [depFormEntry, setDepFormEntry] = useState<DeploymentFormEntry>(emptyDeploymentEntry())
  const [depFormErrors, setDepFormErrors] = useState<InlineDeploymentFormErrors>({})

  const [errors, setErrors] = useState<FormErrors>({})

  const createModel = useCreateModel()
  const createDeployment = useCreateDeployment()
  const { toast } = useToast()
  const { data: modelsData } = useModels()
  const { data: providersData } = useProviders()

  const providerOptions: SelectOption[] = useMemo(() => {
    const items = providersData?.data ?? []
    return [
      { value: '', label: t('connection.select_provider') },
      ...items.map((p) => ({
        value: p.id,
        label: p.slug ? `${p.name} (${p.slug})` : p.name,
      })),
    ]
  }, [providersData, t])

  function handleClose() {
    setName('')
    setType('chat')
    setAliases('')
    setMaxContextTokens('')
    setInputPricePer1m('')
    setOutputPricePer1m('')
    setTimeout('')
    setMaxRetries('')
    setFallbackModelName('')
    setPiiFilter('default')
    setDeployments([])
    setShowDeploymentForm(false)
    setEditingDeployment(null)
    setDepFormEntry(emptyDeploymentEntry())
    setDepFormErrors({})
    setErrors({})
    onClose()
  }

  function validateDepForm(): boolean {
    const next: InlineDeploymentFormErrors = {}
    if (!depFormEntry.name.trim()) next.name = 'Name is required'
    if (!depFormEntry.providerId && !depFormEntry.provider) {
      next.provider = t('connection.select_or_protocol')
    }
    if (!depFormEntry.providerId && !depFormEntry.baseUrl.trim()) {
      next.base_url = t('connection.base_url_required')
    }
    setDepFormErrors(next)
    return Object.keys(next).length === 0
  }

  function handleDepFormSave(e: React.MouseEvent) {
    e.preventDefault()
    if (!validateDepForm()) return

    if (editingDeployment !== null) {
      setDeployments((prev) => {
        const next = [...prev]
        next[editingDeployment] = { ...depFormEntry }
        return next
      })
      setEditingDeployment(null)
    } else {
      setDeployments((prev) => [...prev, { ...depFormEntry }])
      setShowDeploymentForm(false)
    }
    setDepFormEntry(emptyDeploymentEntry())
    setDepFormErrors({})
  }

  function handleDepFormCancel(e: React.MouseEvent) {
    e.preventDefault()
    setShowDeploymentForm(false)
    setEditingDeployment(null)
    setDepFormEntry(emptyDeploymentEntry())
    setDepFormErrors({})
  }

  function handleEditDeploymentEntry(index: number, e: React.MouseEvent) {
    e.preventDefault()
    setEditingDeployment(index)
    setShowDeploymentForm(false)
    setDepFormEntry({ ...deployments[index] })
    setDepFormErrors({})
  }

  function handleRemoveDeploymentEntry(index: number, e: React.MouseEvent) {
    e.preventDefault()
    setDeployments((prev) => prev.filter((_, i) => i !== index))
    if (editingDeployment === index) {
      setEditingDeployment(null)
      setDepFormEntry(emptyDeploymentEntry())
      setDepFormErrors({})
    }
  }

  function validate(): boolean {
    const next: FormErrors = {}

    if (!name.trim()) {
      next.name = 'Name is required'
    }

    if (deployments.length === 0) next.deployments = 'At least one channel is required'

    setErrors(next)
    return Object.keys(next).length === 0
  }

  async function handleSubmit(e: React.MouseEvent) {
    e.preventDefault()
    if (!validate()) return

    const parsedAliases = aliases.split(',').map((a) => a.trim()).filter(Boolean)

    const params: CreateModelParams = {
      name: name.trim(),
      type,
      strategy,
      bill_per_token: billPerToken,
      bill_per_request: billPerRequest,
    }

    if (maxRetries.trim()) {
      const parsed = parseInt(maxRetries, 10)
      if (!isNaN(parsed)) params.max_retries = parsed
    }
    if (fallbackModelName) params.fallback_model_name = fallbackModelName
    if (maxContextTokens.trim()) {
      const parsed = parseInt(maxContextTokens, 10)
      if (!isNaN(parsed)) params.max_context_tokens = parsed
    }
    if (inputPricePer1m.trim()) {
      const parsed = parseFloat(inputPricePer1m)
      if (!isNaN(parsed)) params.input_price_per_1m = parsed
    }
    if (outputPricePer1m.trim()) {
      const parsed = parseFloat(outputPricePer1m)
      if (!isNaN(parsed)) params.output_price_per_1m = parsed
    }
    if (timeout.trim()) params.timeout = timeout.trim()
    if (parsedAliases.length > 0) params.aliases = parsedAliases
    const piiFilterValueLB = piiFilterToParam(piiFilter)
    if (piiFilterValueLB !== undefined) params.pii_filter = piiFilterValueLB
    if (sellInputPer1m.trim()) {
      const parsedSell = parseFloat(sellInputPer1m)
      if (!isNaN(parsedSell)) params.sell_input_per_1m = parsedSell
    }
    if (sellOutputPer1m.trim()) {
      const parsedSell = parseFloat(sellOutputPer1m)
      if (!isNaN(parsedSell)) params.sell_output_per_1m = parsedSell
    }
    if (sellCachedInputPer1m.trim()) {
      const parsedSell = parseFloat(sellCachedInputPer1m)
      if (!isNaN(parsedSell)) params.sell_cached_input_per_1m = parsedSell
    }
    if (sellPerRequest.trim()) {
      const parsedSell = parseFloat(sellPerRequest)
      if (!isNaN(parsedSell)) params.sell_per_request = parsedSell
    }

    try {
      const model = await createModel.mutateAsync(params)
      for (const dep of deployments) {
        await createDeployment.mutateAsync({
          modelId: model.id,
          params: {
            name: dep.name,
            provider: dep.provider || undefined,
            base_url: dep.baseUrl || undefined,
            provider_id: dep.providerId || undefined,
            upstream_model: dep.upstreamModel || undefined,
            api_key: dep.apiKey || undefined,
            azure_deployment: dep.azureDeployment || undefined,
            azure_api_version: dep.azureApiVersion || undefined,
            weight: dep.weight,
            priority: dep.priority,
          },
        })
      }
      toast({ variant: 'success', message: 'Model added' })
      handleClose()
    } catch (err) {
      toast({
        variant: 'error',
        message: err instanceof Error ? err.message : 'Failed to add model',
      })
    }
  }

  const providers = providersData?.data ?? []
  const isPending = createModel.isPending || createDeployment.isPending
  const depFormIsOpen = showDeploymentForm || editingDeployment !== null

  function patchDepForm(patch: Partial<DeploymentFormEntry>) {
    setDepFormEntry((prev) => ({ ...prev, ...patch }))
  }

  const fallbackOptions: SelectOption[] = useMemo(() => {
    const allModels = modelsData?.data ?? []
    return [
      { value: '', label: 'None' },
      ...allModels
        .filter((m) => m.name !== name && m.type === type)
        .map((m) => ({ value: m.name, label: m.name })),
    ]
  }, [modelsData, name, type])

  return (
    <Dialog open={open} onClose={handleClose} title={t('models.add')}>
      <div className="space-y-4">
        <Input
          label="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. gpt-4o"
          error={errors.name}
          disabled={isPending}
        />
        <Select
          label="Type"
          options={MODEL_TYPE_OPTIONS}
          value={type}
          onChange={setType}
          disabled={isPending}
        />

        <>
            <p className="text-xs text-text-tertiary">{t('models.routing_hint')}</p>
            <Input
              label="Max Retries"
              type="number"
              value={maxRetries}
              onChange={(e) => setMaxRetries(e.target.value)}
              placeholder="0"
              disabled={isPending}
            />
            <div>
              <Select
                label="Fallback Model"
                options={fallbackOptions}
                value={fallbackModelName}
                onChange={setFallbackModelName}
                disabled={isPending}
              />
              <p className="text-xs text-text-tertiary mt-1">
                When this model fails, requests automatically retry on the fallback model.
              </p>
            </div>

            {/* Deployments list */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium text-text-secondary">{t('models.deployments')}</span>
              </div>

              {deployments.length === 0 && !depFormIsOpen && (
                <p className="text-sm text-text-tertiary mb-2">{t('models.no_deployments')}</p>
              )}

              {deployments.length > 0 && (
                <div className="rounded-md border border-border mb-2 divide-y divide-border/40">
                  {deployments.map((dep, index) => (
                    <div key={index}>
                      {editingDeployment === index ? (
                        <div className="p-3 space-y-3 bg-bg-tertiary/50">
                          <p className="text-xs font-medium text-text-tertiary uppercase tracking-wider">Edit Deployment</p>
                          <Input
                            label="Name"
                            value={depFormEntry.name}
                            onChange={(e) => setDepFormEntry((prev) => ({ ...prev, name: e.target.value }))}
                            placeholder="e.g. primary"
                            error={depFormErrors.name}
                            disabled={isPending}
                          />
                          <DeploymentConnectionFields
                            entry={depFormEntry}
                            onChange={patchDepForm}
                            errors={depFormErrors}
                            providerOptions={providerOptions}
                            providers={providers}
                            isPending={isPending}
                          />
                          <div className="grid grid-cols-2 gap-3">
                            <Input
                              label="Weight"
                              type="number"
                              value={String(depFormEntry.weight)}
                              onChange={(e) => setDepFormEntry((prev) => ({ ...prev, weight: parseInt(e.target.value, 10) || 1 }))}
                              placeholder="1"
                              disabled={isPending}
                            />
                            <Input
                              label="Priority"
                              type="number"
                              value={String(depFormEntry.priority)}
                              onChange={(e) => setDepFormEntry((prev) => ({ ...prev, priority: parseInt(e.target.value, 10) || 0 }))}
                              placeholder="0"
                              disabled={isPending}
                            />
                          </div>
                          <div className="flex gap-2">
                            <Button size="sm" onClick={handleDepFormSave} disabled={isPending}>
                              Save
                            </Button>
                            <Button size="sm" variant="secondary" onClick={handleDepFormCancel} disabled={isPending}>
                              Cancel
                            </Button>
                          </div>
                        </div>
                      ) : (
                        <div className="flex items-center justify-between px-3 py-2">
                          <div className="min-w-0">
                            <span className="font-mono text-sm text-text-primary">{dep.name}</span>
                            <span className="text-text-tertiary text-xs ml-2">
                              {dep.providerId
                                ? providers.find((p) => p.id === dep.providerId)?.name ?? 'provider'
                                : providerLabels[dep.provider] ?? dep.provider}
                            </span>
                            {!dep.providerId && (
                              <span className="text-text-tertiary text-xs ml-2 truncate hidden sm:inline">
                                {dep.baseUrl.length > 40 ? dep.baseUrl.slice(0, 40) + '…' : dep.baseUrl}
                              </span>
                            )}
                            {dep.upstreamModel && (
                              <span className="text-text-tertiary text-xs ml-2 font-mono hidden md:inline">
                                → {dep.upstreamModel}
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-1 shrink-0 ml-2">
                            <button
                              type="button"
                              onClick={(e) => handleEditDeploymentEntry(index, e)}
                              className="p-1 rounded text-text-tertiary hover:text-text-primary hover:bg-bg-tertiary transition-colors"
                              title="Edit deployment"
                            >
                              <IconPencil />
                            </button>
                            <button
                              type="button"
                              onClick={(e) => handleRemoveDeploymentEntry(index, e)}
                              className="p-1 rounded text-text-tertiary hover:text-error hover:bg-error/10 transition-colors"
                              title="Remove deployment"
                            >
                              <IconTrash />
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}

              {errors.deployments && (
                <p className="text-xs text-error mb-2">{errors.deployments}</p>
              )}

              {showDeploymentForm && (
                <div className="rounded-md border border-border p-3 space-y-3 mb-2 bg-bg-tertiary/50">
                  <p className="text-xs font-medium text-text-tertiary uppercase tracking-wider">New Deployment</p>
                  <Input
                    label="Name"
                    value={depFormEntry.name}
                    onChange={(e) => setDepFormEntry((prev) => ({ ...prev, name: e.target.value }))}
                    placeholder="e.g. primary"
                    error={depFormErrors.name}
                    disabled={isPending}
                  />
                  <DeploymentConnectionFields
                    entry={depFormEntry}
                    onChange={patchDepForm}
                    errors={depFormErrors}
                    providerOptions={providerOptions}
                    providers={providers}
                    isPending={isPending}
                  />
                  <div className="grid grid-cols-2 gap-3">
                    <Input
                      label="Weight"
                      type="number"
                      value={String(depFormEntry.weight)}
                      onChange={(e) => setDepFormEntry((prev) => ({ ...prev, weight: parseInt(e.target.value, 10) || 1 }))}
                      placeholder="1"
                      disabled={isPending}
                    />
                    <Input
                      label="Priority"
                      type="number"
                      value={String(depFormEntry.priority)}
                      onChange={(e) => setDepFormEntry((prev) => ({ ...prev, priority: parseInt(e.target.value, 10) || 0 }))}
                      placeholder="0"
                      disabled={isPending}
                    />
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" onClick={handleDepFormSave} disabled={isPending}>
                      Add
                    </Button>
                    <Button size="sm" variant="secondary" onClick={handleDepFormCancel} disabled={isPending}>
                      Cancel
                    </Button>
                  </div>
                </div>
              )}

              {!depFormIsOpen && (
                <button
                  type="button"
                  onClick={(e) => { e.preventDefault(); setShowDeploymentForm(true) }}
                  className="text-xs text-accent hover:text-accent/80 transition-colors"
                >
                  {t('models.add_deployment_link')}
                </button>
              )}
            </div>
        </>

        <Input
          label="Aliases"
          value={aliases}
          onChange={(e) => setAliases(e.target.value)}
          placeholder="default, gpt4, latest"
          description="Comma-separated. Must be globally unique."
          disabled={isPending}
        />

        <details className="group">
          <summary className="flex items-center justify-between cursor-pointer list-none select-none py-2">
            <span className="text-xs font-medium tracking-wider uppercase text-text-tertiary">
              Advanced Settings
            </span>
            <svg
              className="h-3.5 w-3.5 text-text-tertiary transition-transform duration-200 group-open:rotate-180"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
              aria-hidden="true"
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </summary>
          <div className="space-y-4 pt-2">
            <Input
              label="Max Context Tokens"
              type="number"
              value={maxContextTokens}
              onChange={(e) => setMaxContextTokens(e.target.value)}
              placeholder="e.g. 128000"
              disabled={isPending}
            />
            <div className="grid grid-cols-2 gap-4">
              <Input
                label="Input Price per 1M tokens"
                type="number"
                value={inputPricePer1m}
                onChange={(e) => setInputPricePer1m(e.target.value)}
                placeholder="e.g. 2.50"
                disabled={isPending}
              />
              <Input
                label="Output Price per 1M tokens"
                type="number"
                value={outputPricePer1m}
                onChange={(e) => setOutputPricePer1m(e.target.value)}
                placeholder="e.g. 10.00"
                disabled={isPending}
              />
            </div>
            <div className="rounded-lg border border-white/5 p-4 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-text-secondary">Storefront (sell pricing)</p>
                  <p className="text-xs text-text-tertiary">Customer-facing prices; wallet is debited at these rates.</p>
                </div>
                <div className="flex gap-4">
                  <Toggle checked={billPerToken} onChange={setBillPerToken} label="Per token" disabled={isPending} />
                  <Toggle checked={billPerRequest} onChange={setBillPerRequest} label="Per request" disabled={isPending} />
                </div>
              </div>
              <div className="grid grid-cols-3 gap-4">
                <Input
                  label="Sell Input /1M"
                  type="number"
                  value={sellInputPer1m}
                  onChange={(e) => setSellInputPer1m(e.target.value)}
                  placeholder="3.00"
                  disabled={isPending}
                />
                <Input
                  label="Sell Output /1M"
                  type="number"
                  value={sellOutputPer1m}
                  onChange={(e) => setSellOutputPer1m(e.target.value)}
                  placeholder="12.00"
                  disabled={isPending}
                />
                <Input
                  label="Sell Cached /1M"
                  type="number"
                  value={sellCachedInputPer1m}
                  onChange={(e) => setSellCachedInputPer1m(e.target.value)}
                  placeholder="1.50"
                  disabled={isPending}
                />
              </div>
              {billPerRequest && (
                <Input
                  label="Sell per request (USD)"
                  type="number"
                  value={sellPerRequest}
                  onChange={(e) => setSellPerRequest(e.target.value)}
                  placeholder="0.01"
                  disabled={isPending}
                />
              )}
            </div>
            <Input
              label="Timeout"
              value={timeout}
              onChange={(e) => setTimeout(e.target.value)}
              placeholder="e.g. 30s, 2m, 5m"
              description="Per-model upstream timeout. Empty = use global default."
              disabled={isPending}
            />
            <div>
              <Select
                label="PII Filter"
                options={PII_FILTER_OPTIONS}
                value={piiFilter}
                onChange={setPiiFilter}
                disabled={isPending}
              />
              <p className="text-xs text-text-tertiary mt-1">
                Controls PII anonymization for requests to this model. Default applies network-based rules.
              </p>
            </div>
          </div>
        </details>

        <div className="flex justify-end gap-2 pt-2">
          <Button
            variant="secondary"
            onClick={handleClose}
            disabled={isPending}
          >
            Cancel
          </Button>
          <Button onClick={handleSubmit} loading={isPending}>
            Add Model
          </Button>
        </div>
      </div>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// EditModelDialog
// ---------------------------------------------------------------------------

interface EditModelDialogProps {
  model: ModelResponse
  onClose: () => void
}

function EditModelDialog({ model, onClose }: EditModelDialogProps) {
  const [name, setName] = useState(model.name)
  const [provider, setProvider] = useState(model.provider)
  const [type, setType] = useState(model.type || 'chat')
  const [baseUrl, setBaseUrl] = useState(model.base_url)
  const [apiKey, setApiKey] = useState('')
  const [aliases, setAliases] = useState((model.aliases ?? []).join(', '))
  const [maxContextTokens, setMaxContextTokens] = useState(
    model.max_context_tokens > 0 ? String(model.max_context_tokens) : '',
  )
  const [inputPrice, setInputPrice] = useState(
    model.input_price_per_1m > 0 ? String(model.input_price_per_1m) : '',
  )
  const [outputPrice, setOutputPrice] = useState(
    model.output_price_per_1m > 0 ? String(model.output_price_per_1m) : '',
  )
  const [azureDeployment, setAzureDeployment] = useState(model.azure_deployment ?? '')
  const [azureApiVersion, setAzureApiVersion] = useState(model.azure_api_version ?? '')
  const [timeout, setTimeout] = useState(model.timeout ?? '')
  const [fallbackModelName, setFallbackModelName] = useState(model.fallback_model_name ?? '')
  const [piiFilter, setPiiFilter] = useState(() => piiFilterFromResponse(model.pii_filter))
  const [billPerToken, setBillPerToken] = useState(model.bill_per_token !== false)
  const [billPerRequest, setBillPerRequest] = useState(model.bill_per_request === true)
  const [sellInputPer1m, setSellInputPer1m] = useState(
    model.sell_input_per_1m != null ? String(model.sell_input_per_1m) : '',
  )
  const [sellOutputPer1m, setSellOutputPer1m] = useState(
    model.sell_output_per_1m != null ? String(model.sell_output_per_1m) : '',
  )
  const [sellCachedInputPer1m, setSellCachedInputPer1m] = useState(
    model.sell_cached_input_per_1m != null ? String(model.sell_cached_input_per_1m) : '',
  )
  const [sellPerRequest, setSellPerRequest] = useState(
    model.sell_per_request != null ? String(model.sell_per_request) : '',
  )

  const updateModel = useUpdateModel()
  const { toast } = useToast()
  const { data: modelsData } = useModels()

  const isAzure = provider === 'azure'

  const fallbackOptions: SelectOption[] = useMemo(() => {
    const allModels = modelsData?.data ?? []
    return [
      { value: '', label: 'None' },
      ...allModels
        .filter((m) => m.name !== name && m.type === type)
        .map((m) => ({ value: m.name, label: m.name })),
    ]
  }, [modelsData, name, type])

  function handleSubmit(e: React.FormEvent | React.MouseEvent) {
    e.preventDefault()

    const params: UpdateModelParams = {}

    if (name.trim() !== model.name) params.name = name.trim()
    if (provider !== model.provider) params.provider = provider
    if (type !== (model.type || 'chat')) params.type = type
    if (baseUrl.trim() !== model.base_url) params.base_url = baseUrl.trim()
    if (apiKey.trim()) params.api_key = apiKey.trim()

    if (maxContextTokens.trim()) {
      const parsed = parseInt(maxContextTokens, 10)
      if (!isNaN(parsed) && parsed !== model.max_context_tokens) {
        params.max_context_tokens = parsed
      }
    } else if (model.max_context_tokens > 0) {
      params.max_context_tokens = 0
    }

    if (inputPrice.trim()) {
      const parsed = parseFloat(inputPrice)
      if (!isNaN(parsed) && parsed !== model.input_price_per_1m) {
        params.input_price_per_1m = parsed
      }
    } else if (model.input_price_per_1m > 0) {
      params.input_price_per_1m = 0
    }

    if (outputPrice.trim()) {
      const parsed = parseFloat(outputPrice)
      if (!isNaN(parsed) && parsed !== model.output_price_per_1m) {
        params.output_price_per_1m = parsed
      }
    } else if (model.output_price_per_1m > 0) {
      params.output_price_per_1m = 0
    }

    if (isAzure) {
      if (azureDeployment.trim() !== (model.azure_deployment ?? '')) {
        params.azure_deployment = azureDeployment.trim()
      }
      if (azureApiVersion.trim() !== (model.azure_api_version ?? '')) {
        params.azure_api_version = azureApiVersion.trim()
      }
    }

    const trimmedTimeout = timeout.trim()
    if (trimmedTimeout !== (model.timeout ?? '')) {
      params.timeout = trimmedTimeout || undefined
    }

    const newAliases = aliases.split(',').map((a) => a.trim()).filter(Boolean)
    const sortedNew = [...newAliases].sort()
    const sortedOld = [...(model.aliases ?? [])].sort()
    if (JSON.stringify(sortedNew) !== JSON.stringify(sortedOld)) {
      params.aliases = newAliases
    }

    if (fallbackModelName !== (model.fallback_model_name ?? '')) {
      // Empty string sent as empty string — backend treats it as "clear"
      // Non-empty string sent as the chosen name
      params.fallback_model_name = fallbackModelName || ''
    }

    const newPiiFilter = piiFilterToParam(piiFilter)
    const currentPiiFilter = model.pii_filter
    // Only include in the patch when the value differs from what the server has.
    // null and undefined both mean "default", so compare normalized values.
    const piiFilterChanged =
      newPiiFilter !== currentPiiFilter &&
      !(newPiiFilter === undefined && (currentPiiFilter === undefined || currentPiiFilter === null))
    if (piiFilterChanged) {
      // Send null to clear (reset to default), boolean to set explicitly.
      params.pii_filter = newPiiFilter ?? null
    }

    if (billPerToken !== (model.bill_per_token !== false)) params.bill_per_token = billPerToken
    if (billPerRequest !== (model.bill_per_request === true)) params.bill_per_request = billPerRequest
    if (sellInputPer1m.trim()) {
      const parsed = parseFloat(sellInputPer1m)
      if (!isNaN(parsed) && parsed !== model.sell_input_per_1m) params.sell_input_per_1m = parsed
    }
    if (sellOutputPer1m.trim()) {
      const parsed = parseFloat(sellOutputPer1m)
      if (!isNaN(parsed) && parsed !== model.sell_output_per_1m) params.sell_output_per_1m = parsed
    }
    if (sellCachedInputPer1m.trim()) {
      const parsed = parseFloat(sellCachedInputPer1m)
      if (!isNaN(parsed) && parsed !== model.sell_cached_input_per_1m) params.sell_cached_input_per_1m = parsed
    }
    if (sellPerRequest.trim()) {
      const parsed = parseFloat(sellPerRequest)
      if (!isNaN(parsed) && parsed !== model.sell_per_request) params.sell_per_request = parsed
    }

    if (Object.keys(params).length === 0) {
      onClose()
      return
    }

    updateModel.mutate(
      { modelId: model.id, params },
      {
        onSuccess: () => {
          toast({ variant: 'success', message: 'Model updated' })
          onClose()
        },
        onError: (err) => {
          toast({
            variant: 'error',
            message: err instanceof Error ? err.message : 'Update failed',
          })
        },
      },
    )
  }

  return (
    <Dialog open onClose={onClose} title="Edit Model">
      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <Input
          label="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. gpt-4o"
          disabled={updateModel.isPending}
        />
        <Select
          label="Provider"
          options={PROVIDER_OPTIONS}
          value={provider}
          onChange={setProvider}
          disabled={updateModel.isPending}
        />
        <Select
          label="Type"
          options={MODEL_TYPE_OPTIONS}
          value={type}
          onChange={setType}
          disabled={updateModel.isPending}
        />
        <Input
          label="Base URL"
          value={baseUrl}
          onChange={(e) => setBaseUrl(e.target.value)}
          placeholder={BASE_URL_PLACEHOLDERS[provider] ?? 'https://'}
          disabled={updateModel.isPending}
        />
        <Input
          label="API Key"
          type="password"
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
          placeholder="Leave empty to keep current key"
          description="Leave empty to keep current key. Enter a new value to replace."
          disabled={updateModel.isPending}
        />
        <Input
          label="Max Context Tokens"
          type="number"
          value={maxContextTokens}
          onChange={(e) => setMaxContextTokens(e.target.value)}
          placeholder="e.g. 128000"
          disabled={updateModel.isPending}
        />
        <div className="grid grid-cols-2 gap-4">
          <Input
            label="Input Price per 1M tokens"
            type="number"
            value={inputPrice}
            onChange={(e) => setInputPrice(e.target.value)}
            placeholder="e.g. 2.50"
            disabled={updateModel.isPending}
          />
          <Input
            label="Output Price per 1M tokens"
            type="number"
            value={outputPrice}
            onChange={(e) => setOutputPrice(e.target.value)}
            placeholder="e.g. 10.00"
            disabled={updateModel.isPending}
          />
        </div>
        <div className="rounded-lg border border-white/5 p-4 space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-text-secondary">Storefront (sell pricing)</p>
              <p className="text-xs text-text-tertiary">Customer-facing prices; wallet is debited at these rates.</p>
            </div>
            <div className="flex gap-4">
              <Toggle checked={billPerToken} onChange={setBillPerToken} label="Per token" disabled={updateModel.isPending} />
              <Toggle checked={billPerRequest} onChange={setBillPerRequest} label="Per request" disabled={updateModel.isPending} />
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <Input
              label="Sell Input /1M"
              type="number"
              value={sellInputPer1m}
              onChange={(e) => setSellInputPer1m(e.target.value)}
              placeholder="3.00"
              disabled={updateModel.isPending}
            />
            <Input
              label="Sell Output /1M"
              type="number"
              value={sellOutputPer1m}
              onChange={(e) => setSellOutputPer1m(e.target.value)}
              placeholder="12.00"
              disabled={updateModel.isPending}
            />
            <Input
              label="Sell Cached /1M"
              type="number"
              value={sellCachedInputPer1m}
              onChange={(e) => setSellCachedInputPer1m(e.target.value)}
              placeholder="1.50"
              disabled={updateModel.isPending}
            />
          </div>
          {billPerRequest && (
            <Input
              label="Sell per request (USD)"
              type="number"
              value={sellPerRequest}
              onChange={(e) => setSellPerRequest(e.target.value)}
              placeholder="0.01"
              disabled={updateModel.isPending}
            />
          )}
        </div>
        {isAzure && (
          <>
            <Input
              label="Azure Deployment"
              value={azureDeployment}
              onChange={(e) => setAzureDeployment(e.target.value)}
              placeholder="e.g. gpt-4o-deployment"
              disabled={updateModel.isPending}
            />
            <Input
              label="Azure API Version"
              value={azureApiVersion}
              onChange={(e) => setAzureApiVersion(e.target.value)}
              placeholder="e.g. 2024-02-01"
              disabled={updateModel.isPending}
            />
          </>
        )}
        <Input
          label="Timeout"
          value={timeout}
          onChange={(e) => setTimeout(e.target.value)}
          placeholder="e.g. 30s, 2m, 5m"
          description="Per-model upstream timeout. Empty = use global default."
          disabled={updateModel.isPending}
        />
        <div>
          <Select
            label="Fallback Model"
            options={fallbackOptions}
            value={fallbackModelName}
            onChange={setFallbackModelName}
            disabled={updateModel.isPending}
          />
          <p className="text-xs text-text-tertiary mt-1">
            When this model fails, requests automatically retry on the fallback model.
          </p>
        </div>
        <Input
          label="Aliases"
          value={aliases}
          onChange={(e) => setAliases(e.target.value)}
          placeholder="default, gpt4, latest"
          description="Comma-separated. Must be globally unique."
          disabled={updateModel.isPending}
        />
        <div>
          <Select
            label="PII Filter"
            options={PII_FILTER_OPTIONS}
            value={piiFilter}
            onChange={setPiiFilter}
            disabled={updateModel.isPending}
          />
          <p className="text-xs text-text-tertiary mt-1">
            Controls PII anonymization for requests to this model. Default applies network-based rules.
          </p>
        </div>
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={onClose} disabled={updateModel.isPending}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} loading={updateModel.isPending}>
            Save Changes
          </Button>
        </div>
      </form>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// ModelsPage
// ---------------------------------------------------------------------------

export default function ModelsPage() {
  const { t } = useTranslation()
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [editModel, setEditModel] = useState<ModelResponse | null>(null)
  const [deleteModelId, setDeleteModelId] = useState<string | null>(null)
  const [expandedModels, setExpandedModels] = useState<Set<string>>(new Set())
  const [editDeployment, setEditDeployment] = useState<{ modelId: string; deployment: DeploymentResponse | null } | null>(null)
  const [deleteDeployment, setDeleteDeployment] = useState<{ modelId: string; deploymentId: string } | null>(null)

  const { data: models, isLoading } = useModels()
  const { data: healthData } = useModelHealth()
  const deleteModel = useDeleteModel()
  const toggleModel = useToggleModel()
  const deleteDeploymentMutation = useDeleteDeployment()
  const { toast } = useToast()

  const allModels = models?.data ?? []
  const activeCount = allModels.filter((m) => m.is_active).length
  const inactiveCount = allModels.length - activeCount

  // Build O(1) lookup: model name → health info
  const healthByName = React.useMemo(() => {
    const map = new Map<string, ModelHealthInfo>()
    for (const h of healthData?.models ?? []) {
      map.set(h.name, h)
    }
    return map
  }, [healthData])

  const columns: Column<ModelResponse>[] = [
    {
      key: 'logo',
      header: '',
      width: '36px',
      render: (row) => <BrandIcon logo={row.logo} modelName={row.name} size={22} />,
    },
    {
      key: 'name',
      header: t('models.col_name'),
      render: (row) => (
        <span className="font-mono text-text-primary text-sm">{row.name}</span>
      ),
    },
    {
      key: 'provider',
      header: 'Provider',
      render: (row) => {
        const depCount = row.deployments?.length ?? 0
        if (depCount > 0) {
          return (
            <div className="flex items-center gap-2">
              <Badge variant="info">{depCount} deployments</Badge>
              {row.strategy && <Badge variant="muted">{row.strategy}</Badge>}
            </div>
          )
        }
        const key = isKnownProvider(row.provider) ? row.provider : 'custom'
        return (
          <Badge variant={providerBadgeVariant[key]}>
            {providerLabels[key]}
          </Badge>
        )
      },
    },
    {
      key: 'type',
      header: 'Type',
      render: (row) => (
        <Badge variant={typeBadgeVariant[row.type] ?? 'muted'}>
          {typeLabels[row.type] ?? row.type ?? 'Chat'}
        </Badge>
      ),
    },
    {
      key: 'health',
      header: 'Health',
      render: (row) => {
        if (row.deployments?.length) {
          const depHealths = row.deployments
            .map((d) => healthByName.get(`${row.name}/${d.name}`))
            .filter((h): h is ModelHealthInfo => h != null)
          if (depHealths.length === 0) return <HealthBadge info={undefined} />
          const allUnhealthy = depHealths.every((h) => h.status === 'unhealthy')
          const allHealthy = depHealths.every((h) => h.status === 'healthy')
          const allUnknown = depHealths.every((h) => h.status === 'unknown')
          const status = allUnknown ? 'unknown' : allUnhealthy ? 'unhealthy' : allHealthy ? 'healthy' : 'degraded'
          const avgLatency = Math.round(
            depHealths.reduce((sum, h) => sum + (h.latency_ms ?? 0), 0) / depHealths.length,
          )
          const syntheticInfo: ModelHealthInfo = {
            name: row.name,
            status,
            latency_ms: avgLatency,
            last_check: '',
            health_ok: null,
            models_ok: null,
            functional_ok: null,
          }
          return <HealthBadge info={syntheticInfo} />
        }
        return <HealthBadge info={healthByName.get(row.name)} />
      },
    },
    {
      key: 'aliases',
      header: 'Aliases',
      render: (row) => {
        const list = row.aliases ?? []
        if (list.length === 0) return <span className="text-text-tertiary">—</span>
        return (
          <div className="flex flex-wrap gap-1">
            {list.map((a) => (
              <Badge key={a} variant="muted">{a}</Badge>
            ))}
          </div>
        )
      },
    },
    {
      key: 'max_context_tokens',
      header: 'Context',
      render: (row) =>
        row.max_context_tokens > 0 ? (
          <span className="text-text-secondary">
            {row.max_context_tokens.toLocaleString()}
          </span>
        ) : (
          <span className="text-text-tertiary">—</span>
        ),
    },
    {
      key: 'source',
      header: 'Source',
      render: (row) => (
        <Badge variant={row.source === 'yaml' ? 'muted' : 'default'}>
          {row.source}
        </Badge>
      ),
    },
    {
      key: 'is_active',
      header: 'Status',
      render: (row) => (
        <Toggle
          checked={row.is_active}
          onChange={(activate) =>
            toggleModel.mutate(
              { modelId: row.id, activate },
              {
                onError: (err) => {
                  toast({
                    variant: 'error',
                    message:
                      err instanceof Error
                        ? err.message
                        : 'Failed to update model status',
                  })
                },
              },
            )
          }
          disabled={toggleModel.isPending && toggleModel.variables?.modelId === row.id}
          size="sm"
        />
        ),
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (row) => {
        if (row.source !== 'api') return null
        return (
          <div className="flex items-center justify-end gap-1">
            {!row.deployments?.length && row.strategy && (
              <button
                type="button"
                onClick={() => setEditDeployment({ modelId: row.id, deployment: null })}
                title="Add deployment"
                className="p-1.5 rounded-md text-text-tertiary hover:text-accent hover:bg-accent/10 transition-colors"
              >
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                </svg>
              </button>
            )}
            <button
              type="button"
              onClick={() => setEditModel(row)}
              disabled={deleteModel.isPending && deleteModelId === row.id}
              title="Edit model"
              className="p-1.5 rounded-md text-text-tertiary hover:text-text-primary hover:bg-bg-tertiary transition-colors disabled:opacity-40"
            >
              <IconPencil />
            </button>
            <button
              type="button"
              onClick={() => setDeleteModelId(row.id)}
              disabled={deleteModel.isPending && deleteModelId === row.id}
              title="Delete model"
              className="p-1.5 rounded-md text-text-tertiary hover:text-error hover:bg-error/10 transition-colors disabled:opacity-40"
            >
              <IconTrash />
            </button>
          </div>
        )
      },
    },
  ]

  function handleDelete() {
    if (!deleteModelId) return
    deleteModel.mutate(deleteModelId, {
      onSuccess: () => {
        toast({ variant: 'success', message: 'Model deleted' })
        setDeleteModelId(null)
      },
      onError: (err) => {
        toast({
          variant: 'error',
          message: err instanceof Error ? err.message : 'Failed to delete model',
        })
        setDeleteModelId(null)
      },
    })
  }

  function handleDeleteDeployment() {
    if (!deleteDeployment) return
    deleteDeploymentMutation.mutate(
      { modelId: deleteDeployment.modelId, deploymentId: deleteDeployment.deploymentId },
      {
        onSuccess: () => {
          toast({ variant: 'success', message: 'Deployment deleted' })
          setDeleteDeployment(null)
        },
        onError: (err) => {
          toast({
            variant: 'error',
            message: err instanceof Error ? err.message : 'Failed to delete deployment',
          })
          setDeleteDeployment(null)
        },
      },
    )
  }

  return (
    <>
      <PageHeader
        title={t('models.title')}
        description={t('models.desc')}
        actions={
          <Button onClick={() => setShowCreateDialog(true)}>{t('models.add')}</Button>
        }
      />

      {/* Stat cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
        <StatCard
          label={t('models.total')}
          value={isLoading ? '—' : allModels.length}
          icon={<IconLayers />}
          iconColor="purple"
        />
        <StatCard
          label={t('models.active')}
          value={isLoading ? '—' : activeCount}
          icon={<IconActivity />}
          iconColor="green"
        />
        <StatCard
          label={t('models.inactive')}
          value={isLoading ? '—' : inactiveCount}
          icon={<IconPauseCircle />}
          iconColor="yellow"
        />
      </div>

      <Table<ModelResponse>
        columns={columns}
        data={allModels}
        keyExtractor={(row) => row.id}
        loading={isLoading}
        emptyMessage={t('models.empty')}
        expandedKeys={expandedModels}
        onToggleExpand={(key) => {
          setExpandedModels((prev) => {
            const next = new Set(prev)
            if (next.has(key)) next.delete(key)
            else next.add(key)
            return next
          })
        }}
        renderExpandedRow={(row) => {
          if (!row.deployments?.length) return null
          const isApi = row.source === 'api'
          return (
            <div className="py-3" style={{ paddingLeft: 'calc(2rem + 1rem + 1rem)' }}>
              {row.deployments?.length ? (
                <table className="min-w-full">
                  <thead>
                    <tr className="border-b border-border/40">
                      <th className="px-3 py-2 text-[10px] font-medium text-text-tertiary uppercase tracking-wider text-left">{t('models.col_deployment')}</th>
                      <th className="px-3 py-2 text-[10px] font-medium text-text-tertiary uppercase tracking-wider text-left">{t('models.col_provider')}</th>
                      <th className="px-3 py-2 text-[10px] font-medium text-text-tertiary uppercase tracking-wider text-left">{t('models.col_health')}</th>
                      <th className="px-3 py-2 text-[10px] font-medium text-text-tertiary uppercase tracking-wider text-left">{t('common.endpoint')}</th>
                      <th className="px-3 py-2 text-[10px] font-medium text-text-tertiary uppercase tracking-wider text-left">{t('common.weight')}</th>
                      <th className="px-3 py-2 text-[10px] font-medium text-text-tertiary uppercase tracking-wider text-left">{t('common.priority')}</th>
                      {isApi && (
                        <th className="px-3 py-2 text-[10px] font-medium text-text-tertiary uppercase tracking-wider text-right">{t('common.actions')}</th>
                      )}
                    </tr>
                  </thead>
                  <tbody>
                    {row.deployments.map((dep: DeploymentResponse) => (
                      <tr key={dep.id} className="border-b border-border/20 last:border-b-0">
                        <td className="px-3 py-2 text-sm">
                          <span className="font-mono text-text-secondary">{dep.name}</span>
                        </td>
                        <td className="px-3 py-2 text-sm">
                          <Badge
                            variant={providerBadgeVariant[dep.provider as keyof typeof providerBadgeVariant] ?? 'muted'}
                          >
                            {providerLabels[dep.provider as keyof typeof providerLabels] ?? dep.provider}
                          </Badge>
                        </td>
                        <td className="px-3 py-2 text-sm">
                          <HealthBadge info={healthByName.get(`${row.name}/${dep.name}`)} />
                        </td>
                        <td className="px-3 py-2 text-sm">
                          <span className="text-xs text-text-tertiary font-mono">{dep.base_url}</span>
                        </td>
                        <td className="px-3 py-2 text-sm text-text-secondary">{dep.weight}</td>
                        <td className="px-3 py-2 text-sm text-text-secondary">{dep.priority}</td>
                        {isApi && (
                          <td className="px-3 py-2 text-sm text-right">
                            <div className="flex items-center justify-end gap-1">
                              <button
                                type="button"
                                onClick={() => setEditDeployment({ modelId: row.id, deployment: dep })}
                                title="Edit deployment"
                                className="p-1 rounded text-text-tertiary hover:text-text-primary hover:bg-bg-tertiary transition-colors"
                              >
                                <IconPencil />
                              </button>
                              <button
                                type="button"
                                onClick={() => setDeleteDeployment({ modelId: row.id, deploymentId: dep.id })}
                                title="Delete deployment"
                                className="p-1 rounded text-text-tertiary hover:text-error hover:bg-error/10 transition-colors"
                              >
                                <IconTrash />
                              </button>
                            </div>
                          </td>
                        )}
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : null}
              {isApi && (
                <div className="mt-2">
                  <button
                    type="button"
                    onClick={() => setEditDeployment({ modelId: row.id, deployment: null })}
                    className="text-xs text-accent hover:text-accent/80 transition-colors"
                  >
                    {t('models.add_deployment_link')}
                  </button>
                </div>
              )}
            </div>
          )
        }}
      />

      <CreateModelDialog
        open={showCreateDialog}
        onClose={() => setShowCreateDialog(false)}
      />

      {editModel !== null && (
        <EditModelDialog
          model={editModel}
          onClose={() => setEditModel(null)}
        />
      )}

      <ConfirmDialog
        open={deleteModelId !== null}
        onClose={() => setDeleteModelId(null)}
        onConfirm={handleDelete}
        title={t('models.delete')}
        description={t('models.delete_confirm')}
        confirmLabel={t('common.delete')}
        loading={deleteModel.isPending}
      />

      {editDeployment !== null && (
        <DeploymentDialog
          modelId={editDeployment.modelId}
          deployment={editDeployment.deployment}
          onClose={() => setEditDeployment(null)}
        />
      )}

      <ConfirmDialog
        open={deleteDeployment !== null}
        onClose={() => setDeleteDeployment(null)}
        onConfirm={handleDeleteDeployment}
        title={t('models.delete_deployment')}
        description={t('models.delete_dep_confirm')}
        confirmLabel={t('common.delete')}
        loading={deleteDeploymentMutation.isPending}
      />
    </>
  )
}
