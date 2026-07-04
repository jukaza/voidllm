import { useEffect, useMemo, useRef, useState } from 'react'
import { BrandIcon } from '../ui/BrandIcon'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { Badge } from '../ui/Badge'
import { Drawer, DrawerSection } from '../ui/Drawer'
import {
  useProviderPresets,
  useDiscoverProviderModels,
  useImportProvider,
  useCreateProvider,
} from '../../hooks/useProviders'
import type { DiscoveredModel, ProviderPreset } from '../../hooks/useProviders'
import { getConnectionProfile, isKeyOnlyPreset } from '../../lib/connection-profiles'
import { endpointHintKey, modelIdsIncludeClaude, wireMismatchKey } from '../../lib/endpoint-hints'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'

interface ProviderSetupDrawerProps {
  open: boolean
  onClose: () => void
  /** Pre-select a preset when opening (e.g. from legacy wizard URL). */
  initialPresetId?: string
}

export function ProviderSetupDrawer({ open, onClose, initialPresetId }: ProviderSetupDrawerProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data: presetsData } = useProviderPresets()
  const discover = useDiscoverProviderModels()
  const importProvider = useImportProvider()
  const createProvider = useCreateProvider()

  const [preset, setPreset] = useState<ProviderPreset | null>(null)
  const [name, setName] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [baseUrl, setBaseUrl] = useState('')
  const [discovered, setDiscovered] = useState<DiscoveredModel[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [markup, setMarkup] = useState('1.5')
  const [discoverMsg, setDiscoverMsg] = useState('')
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [customEndpoint, setCustomEndpoint] = useState(false)

  const presets = presetsData?.data ?? []

  const profile = useMemo(
    () => (preset ? getConnectionProfile(preset.protocol) : null),
    [preset],
  )

  function resetForm() {
    setPreset(null)
    setName('')
    setApiKey('')
    setBaseUrl('')
    setDiscovered([])
    setSelected(new Set())
    setMarkup('1.5')
    setDiscoverMsg('')
    setShowAdvanced(false)
    setCustomEndpoint(false)
  }

  function handleClose() {
    resetForm()
    onClose()
  }

  function pickPreset(p: ProviderPreset, clearModels = true) {
    setPreset(p)
    setName(p.name)
    setBaseUrl(p.base_url ?? '')
    setApiKey('')
    setCustomEndpoint(false)
    if (clearModels) {
      setDiscovered([])
      setSelected(new Set())
      setDiscoverMsg('')
    }
  }

  const initKeyRef = useRef('')
  useEffect(() => {
    if (!open) {
      initKeyRef.current = ''
      return
    }
    if (presets.length === 0) return
    const initKey = `${initialPresetId ?? 'default'}`
    if (initKeyRef.current === initKey) return
    const p = initialPresetId
      ? presets.find((x) => x.id === initialPresetId) ?? presets[0]
      : presets[0]
    if (p) pickPreset(p, true)
    initKeyRef.current = initKey
  }, [open, initialPresetId, presets])

  async function runDiscover() {
    if (!preset) return
    if (!apiKey.trim() && preset.protocol !== 'ollama') {
      toast({ variant: 'error', message: t('wizard.api_key_required') })
      return
    }
    const res = await discover.mutateAsync({
      preset_id: preset.id,
      api_key: apiKey.trim(),
      base_url: baseUrl.trim() || undefined,
      protocol: preset.protocol,
    })
    setDiscoverMsg(res.message)
    if (!res.success && res.data.length === 0) {
      toast({ variant: 'error', message: res.message || t('wizard.discover_failed') })
      return
    }
    setDiscovered(res.data)
    setSelected(new Set(res.data.map((m) => m.id)))
    if (res.message) {
      toast({ variant: res.success ? 'success' : 'info', message: res.message })
    }
  }

  function toggleModel(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function selectAllModels() {
    setSelected(new Set(discovered.map((m) => m.id)))
  }

  function selectNoModels() {
    setSelected(new Set())
  }

  async function runImport() {
    if (!preset || selected.size === 0) return
    const markupNum = parseFloat(markup)
    try {
      const res = await importProvider.mutateAsync({
        preset_id: preset.id,
        name: name.trim() || undefined,
        api_key: apiKey.trim() || undefined,
        base_url: baseUrl.trim() || undefined,
        protocol: preset.protocol,
        markup: Number.isFinite(markupNum) && markupNum > 0 ? markupNum : 1.5,
        models: [...selected].map((id) => ({ upstream_id: id })),
      })
      const failed = res.results.filter((r) => r.error)
      const ok = res.results.length - failed.length
      if (failed.length > 0) {
        toast({
          variant: 'info',
          message: t('wizard.import_partial', { ok, failed: failed.length }),
        })
      } else {
        toast({ variant: 'success', message: t('wizard.import_success', { ok }) })
      }
      handleClose()
    } catch (e) {
      toast({
        variant: 'error',
        message: e instanceof Error ? e.message : t('wizard.import_failed'),
      })
    }
  }

  async function saveProviderOnly() {
    if (!preset) return
    if (!name.trim()) {
      toast({ variant: 'error', message: t('marketplace.provider_name_required') })
      return
    }
    try {
      await createProvider.mutateAsync({
        name: name.trim(),
        slug: preset.id,
        protocol: preset.protocol,
        base_url: baseUrl.trim() || preset.base_url || undefined,
        api_key: apiKey.trim() || undefined,
        logo: preset.logo,
      })
      toast({ variant: 'success', message: t('common.saved') })
      handleClose()
    } catch (e) {
      toast({
        variant: 'error',
        message: e instanceof Error ? e.message : t('wizard.import_failed'),
      })
    }
  }

  const isPending = discover.isPending || importProvider.isPending || createProvider.isPending
  const canDiscover = Boolean(preset && (apiKey.trim() || preset.protocol === 'ollama'))
  const canImport = selected.size > 0
  const keyOnly = preset ? isKeyOnlyPreset(preset) : false
  const showEndpointField = preset && profile && (!keyOnly || customEndpoint)
  const effectiveProtocol = preset?.protocol ?? ''
  const claudeWireWarn =
    preset &&
    modelIdsIncludeClaude(discovered.map((m) => m.id))
      ? wireMismatchKey(
          effectiveProtocol,
          discovered.find((m) => m.id.toLowerCase().startsWith('claude-'))?.id,
        )
      : null

  return (
    <Drawer
      open={open}
      onClose={handleClose}
      title={t('drawer.add_provider')}
      footer={
        <div className="flex flex-wrap items-center justify-end gap-2">
          <Button variant="secondary" onClick={handleClose} disabled={isPending}>
            {t('common.cancel')}
          </Button>
          {preset && !canImport && (
            <Button
              variant="secondary"
              onClick={() => void saveProviderOnly()}
              loading={createProvider.isPending}
              disabled={!apiKey.trim() && preset.protocol !== 'ollama'}
            >
              {t('drawer.save_provider_only')}
            </Button>
          )}
          <Button
            onClick={() => void runImport()}
            loading={importProvider.isPending}
            disabled={!canImport || !preset}
          >
            {t('drawer.create_and_import', { count: selected.size })}
          </Button>
        </div>
      }
    >
      <DrawerSection title={t('drawer.section_upstream')} description={t('drawer.section_upstream_desc')}>
        <div className="grid grid-cols-3 sm:grid-cols-4 gap-2">
          {presets.map((p) => {
            const active = preset?.id === p.id
            return (
              <button
                key={p.id}
                type="button"
                onClick={() => pickPreset(p)}
                className={
                  active
                    ? 'flex flex-col items-center gap-1.5 rounded-lg border-2 border-accent bg-accent/10 p-3 transition-colors'
                    : 'flex flex-col items-center gap-1.5 rounded-lg border border-border bg-bg-secondary p-3 hover:border-accent/40 hover:bg-bg-tertiary transition-colors'
                }
              >
                <BrandIcon logo={p.logo} slug={p.id} protocol={p.protocol} size={28} />
                <span className="text-[10px] font-medium text-center leading-tight line-clamp-2">{p.name}</span>
              </button>
            )
          })}
        </div>
      </DrawerSection>

      {preset && profile && (
        <DrawerSection title={t('drawer.section_connection')} description={t('drawer.section_connection_desc')}>
          <div className="space-y-4">
            <div className="flex items-center gap-3 rounded-lg border border-border bg-bg-secondary px-3 py-2">
              <BrandIcon logo={preset.logo} slug={preset.id} protocol={preset.protocol} size={24} />
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium truncate">{preset.name}</p>
                <p className="text-xs text-text-tertiary">{t('wizard.protocol_label', { protocol: preset.protocol })}</p>
              </div>
            </div>

            <Input
              label={t('marketplace.provider_name')}
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={preset.name}
            />

            {keyOnly ? (
              <p className="text-xs text-text-tertiary -mt-2">
                {t('wizard.key_only_hint', { url: preset.base_url })}{' '}
                {t('drawer.key_only_proxy_hint')}
              </p>
            ) : null}

            {keyOnly ? (
              <label className="flex items-center gap-2 text-sm text-text-secondary cursor-pointer">
                <input
                  type="checkbox"
                  checked={customEndpoint}
                  onChange={(e) => {
                    setCustomEndpoint(e.target.checked)
                    if (!e.target.checked) setBaseUrl(preset.base_url ?? '')
                  }}
                  className="accent-accent"
                />
                {t('drawer.custom_endpoint')}
              </label>
            ) : null}

            <Input
              label={profile.keyLabel === 'API Key' ? t('common.api_key') : profile.keyLabel}
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder={preset.key_hint || profile.keyPlaceholder}
            />

            {showEndpointField ? (
              <Input
                label={t('common.endpoint')}
                value={baseUrl}
                onChange={(e) => setBaseUrl(e.target.value)}
                placeholder={profile.baseUrlPlaceholder}
                required={profile.baseUrlRequired || customEndpoint}
                description={
                  customEndpoint
                    ? t('drawer.custom_endpoint_hint')
                    : t(endpointHintKey(effectiveProtocol))
                }
              />
            ) : null}

            {claudeWireWarn ? (
              <p className="text-xs text-amber-600 dark:text-amber-400 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2">
                {t(claudeWireWarn)}
              </p>
            ) : null}
          </div>
        </DrawerSection>
      )}

      {preset && (
        <DrawerSection
          title={t('drawer.section_models')}
          description={t('drawer.section_models_desc')}
          actions={
            <>
              <Button size="sm" variant="secondary" onClick={() => void runDiscover()} loading={discover.isPending} disabled={!canDiscover}>
                {t('drawer.fetch_models')}
              </Button>
              {discovered.length > 0 && (
                <>
                  <Button size="sm" variant="secondary" onClick={selectAllModels}>
                    {t('drawer.select_all')}
                  </Button>
                  <Button size="sm" variant="secondary" onClick={selectNoModels}>
                    {t('drawer.select_none')}
                  </Button>
                </>
              )}
            </>
          }
        >
          {!canDiscover && (
            <p className="text-xs text-text-tertiary mb-3">{t('drawer.fetch_requires_key')}</p>
          )}
          {discoverMsg && <p className="text-xs text-text-tertiary mb-3">{discoverMsg}</p>}

          {discovered.length === 0 ? (
            <p className="text-sm text-text-tertiary rounded-lg border border-dashed border-border px-4 py-8 text-center">
              {t('drawer.models_empty')}
            </p>
          ) : (
            <div className="rounded-lg border border-border overflow-hidden max-h-64 overflow-y-auto">
              <table className="w-full text-sm">
                <thead className="bg-bg-secondary text-text-tertiary text-left sticky top-0">
                  <tr>
                    <th className="px-3 py-2 w-8" />
                    <th className="px-3 py-2">{t('wizard.col_upstream')}</th>
                    <th className="px-3 py-2 hidden sm:table-cell">{t('wizard.col_cost')}</th>
                    <th className="px-3 py-2">{t('common.status')}</th>
                  </tr>
                </thead>
                <tbody>
                  {discovered.map((m) => (
                    <tr key={m.id} className="border-t border-border/60">
                      <td className="px-3 py-2">
                        <input
                          type="checkbox"
                          checked={selected.has(m.id)}
                          onChange={() => toggleModel(m.id)}
                          className="accent-accent"
                        />
                      </td>
                      <td className="px-3 py-2 font-mono text-xs text-text-primary">{m.id}</td>
                      <td className="px-3 py-2 text-text-secondary tabular-nums text-xs hidden sm:table-cell">
                        {m.known_cost ? `$${m.known_cost.in} / $${m.known_cost.out}` : '—'}
                      </td>
                      <td className="px-3 py-2">
                        {m.exists ? (
                          <Badge variant="info">{t('wizard.badge_attach')}</Badge>
                        ) : (
                          <Badge variant="success">{t('wizard.badge_new')}</Badge>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </DrawerSection>
      )}

      {preset && (
        <div>
          <button
            type="button"
            onClick={() => setShowAdvanced((v) => !v)}
            className="text-xs text-text-tertiary hover:text-text-secondary flex items-center gap-1"
          >
            <span className={showAdvanced ? 'rotate-90 inline-block' : 'inline-block'}>▸</span>
            {t('drawer.section_advanced')}
          </button>
          {showAdvanced && (
            <div className="mt-3 space-y-3 pl-4 border-l border-border/60">
              <Input
                label={t('wizard.markup_label')}
                value={markup}
                onChange={(e) => setMarkup(e.target.value)}
                placeholder="1.5"
              />
              <p className="text-xs text-text-tertiary">{t('wizard.markup_hint')}</p>
            </div>
          )}
        </div>
      )}
    </Drawer>
  )
}