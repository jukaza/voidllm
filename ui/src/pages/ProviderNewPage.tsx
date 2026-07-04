import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { PageHeader } from '../components/ui/PageHeader'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { BrandIcon } from '../components/ui/BrandIcon'
import { useProviderPresets, useCreateProvider } from '../hooks/useProviders'
import type { ProviderPreset } from '../hooks/useProviders'
import { getConnectionProfile } from '../lib/connection-profiles'
import { endpointHintKey } from '../lib/endpoint-hints'
import { useToast } from '../hooks/useToast'
import { useTranslation } from '../lib/i18n'

export default function ProviderNewPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const manual = searchParams.get('manual') === '1'
  const presetId = searchParams.get('preset')
  const { toast } = useToast()
  const { data: presetsData } = useProviderPresets()
  const createProvider = useCreateProvider()
  const presets = presetsData?.data ?? []
  const presetHandled = useRef(false)

  const [name, setName] = useState('')
  const [protocol, setProtocol] = useState('custom')
  const [baseUrl, setBaseUrl] = useState('')

  const profile = getConnectionProfile(protocol)

  function createFromPreset(p: ProviderPreset) {
    createProvider.mutate(
      {
        name: p.name,
        slug: p.id,
        protocol: p.protocol,
        base_url: p.base_url || undefined,
        logo: p.logo,
      },
      {
        onSuccess: (created) => {
          navigate(`/providers/${created.id}?setup=1`, { replace: true })
        },
        onError: (e) => toast({ variant: 'error', message: e.message }),
      },
    )
  }

  useEffect(() => {
    if (manual || !presetId || presets.length === 0 || presetHandled.current) return
    const p = presets.find((x) => x.id === presetId)
    if (p) {
      presetHandled.current = true
      createFromPreset(p)
    }
  }, [manual, presetId, presets])

  function createManual() {
    if (!name.trim()) {
      toast({ variant: 'error', message: t('marketplace.provider_name_required') })
      return
    }
    if (!baseUrl.trim()) {
      toast({ variant: 'error', message: t('provider_detail.base_url_required') })
      return
    }
    createProvider.mutate(
      {
        name: name.trim(),
        protocol,
        base_url: baseUrl.trim(),
      },
      {
        onSuccess: (created) => {
          navigate(`/providers/${created.id}?setup=1`, { replace: true })
        },
        onError: (e) => toast({ variant: 'error', message: e.message }),
      },
    )
  }

  const PROTOCOL_OPTIONS = [
    { value: 'openai', label: t('providers.protocol_openai') },
    { value: 'anthropic', label: t('providers.protocol_anthropic') },
    { value: 'gemini', label: t('providers.protocol_gemini') },
    { value: 'azure', label: t('providers.protocol_azure') },
    { value: 'vertex', label: t('providers.protocol_vertex') },
    { value: 'vllm', label: t('providers.protocol_vllm') },
    { value: 'custom', label: t('providers.protocol_custom') },
  ]

  return (
    <>
      <div className="mb-2">
        <Link
          to="/providers"
          className="text-sm text-text-tertiary hover:text-text-secondary no-underline"
        >
          {t('wizard.back_providers')}
        </Link>
      </div>

      <PageHeader
        title={manual ? t('providers.manual_entry') : t('providers.add_provider')}
        description={t('provider_new.desc')}
      />

      {manual ? (
        <div className="max-w-lg space-y-4 rounded-lg border border-border bg-bg-secondary p-6">
          <Input
            label={t('marketplace.provider_name')}
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <Select
            label={t('common.protocol')}
            options={PROTOCOL_OPTIONS}
            value={protocol}
            onChange={setProtocol}
          />
          <Input
            label={t('common.endpoint')}
            required
            value={baseUrl}
            onChange={(e) => setBaseUrl(e.target.value)}
            placeholder={profile.baseUrlPlaceholder}
            description={t(endpointHintKey(protocol))}
          />
          <div className="flex gap-2 pt-2">
            <Button variant="secondary" onClick={() => navigate('/providers')}>
              {t('common.cancel')}
            </Button>
            <Button onClick={createManual} loading={createProvider.isPending}>
              {t('provider_new.continue')}
            </Button>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 gap-3">
          {presets.map((p) => (
            <button
              key={p.id}
              type="button"
              disabled={createProvider.isPending}
              onClick={() => createFromPreset(p)}
              className="flex flex-col items-center gap-2 rounded-lg border border-border bg-bg-secondary p-4 hover:border-accent/50 hover:bg-bg-tertiary transition-colors disabled:opacity-50"
            >
              <BrandIcon logo={p.logo} slug={p.id} protocol={p.protocol} size={32} />
              <span className="text-xs font-medium text-center leading-tight">{p.name}</span>
            </button>
          ))}
        </div>
      )}
    </>
  )
}