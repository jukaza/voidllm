import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import apiClient from '../../api/client'
import { CopyButton } from '../../components/ui/CopyButton'
import { Input } from '../../components/ui/Input'
import { Select } from '../../components/ui/Select'
import type { SelectOption } from '../../components/ui/Select'
import { Badge } from '../../components/ui/Badge'
import { ModelLimitPicker } from '../../components/keys/ModelLimitPicker'
import { useAPIKeys } from '../../hooks/useAPIKeys'
import { useSiteConfig } from '../../hooks/useSiteConfig'
import { useTranslation } from '../../lib/i18n'
import {
  INTEGRATION_TOOLS,
  TELEGRAM_BOTFATHER,
  TELEGRAM_USERINFO_BOT,
  type IntegrationTool,
} from '../../lib/integrations/tools'
import { getManualConfigs } from '../../lib/integrations/manualConfigs'
import { buildSetupCommands } from '../../lib/integrations/setupUrl'


interface AvailableModel {
  name: string
}

function interpolate(text: string, vars: Record<string, string>): string {
  return text.replace(/\{\{(\w+)\}\}/g, (_, k) => vars[k] ?? `{{${k}}}`)
}

function ToolLogo({ tool }: { tool: IntegrationTool }) {
  const [failed, setFailed] = useState(false)

  if (failed) {
    return (
      <span className="flex size-10 shrink-0 items-center justify-center rounded-lg border border-border bg-bg-primary text-sm font-bold text-accent">
        {tool.name.charAt(0)}
      </span>
    )
  }

  return (
    <span className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-lg border border-border bg-bg-primary p-1.5">
      <img
        src={tool.logo}
        alt={tool.name}
        className="size-full object-contain"
        onError={() => setFailed(true)}
      />
    </span>
  )
}

function ManualConfigBox({ filename, content }: { filename: string; content: string }) {
  const { t } = useTranslation()

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between gap-2">
        <code className="font-mono text-[10px] text-text-tertiary">{filename}</code>
        <CopyButton text={content} label={t('common.copy')} copiedLabel={t('common.copied')} />
      </div>
      <pre className="overflow-x-auto whitespace-pre rounded border border-border bg-bg-primary/60 p-2 font-mono text-[10px] leading-relaxed text-text-primary">
        {content}
      </pre>
    </div>
  )
}

function SetupCommandBox({
  origin,
  toolId,
  apiKey,
  serverUrl,
  provider,
  modelParams,
  selectedModels,
  telegramBotToken,
  telegramUserId,
}: {
  origin: string
  toolId: string
  apiKey: string
  serverUrl: string
  provider: string
  modelParams: Record<string, string>
  selectedModels: string[]
  telegramBotToken: string
  telegramUserId: string
}) {
  const { t } = useTranslation()
  const [tab, setTab] = useState<'unix' | 'win'>('unix')

  const mainModel = modelParams.model || selectedModels[0] || 'gpt-4o'
  const params = {
    tool: toolId,
    key: apiKey || 'your-api-key',
    serverUrl,
    provider,
    model: mainModel,
    models: selectedModels.length > 0 ? selectedModels : [mainModel],
    subagentModel: modelParams.subagentModel,
    haiku: modelParams.haiku,
    sonnet: modelParams.sonnet,
    opus: modelParams.opus,
    telegramBotToken: telegramBotToken || undefined,
    telegramUserId: telegramUserId || undefined,
  }

  const cmds = buildSetupCommands(origin, params)
  const display = tab === 'unix' ? cmds.displayUnix : cmds.displayWin
  const real = tab === 'unix' ? cmds.realUnix : cmds.realWin

  return (
    <div className="mt-3 rounded-lg border border-border bg-bg-primary/50 p-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <span className="text-xs font-semibold text-text-primary">{t('integrations.auto_cmd')}</span>
        <div className="flex gap-1">
          {(['unix', 'win'] as const).map((k) => (
            <button
              key={k}
              type="button"
              onClick={() => setTab(k)}
              className={`cursor-pointer rounded px-2 py-0.5 text-[10px] font-medium transition-colors ${
                tab === k ? 'bg-accent/15 text-accent' : 'text-text-tertiary hover:text-text-primary'
              }`}
            >
              {k === 'unix' ? 'Linux/macOS' : 'Windows'}
            </button>
          ))}
        </div>
      </div>
      <div className="flex items-start gap-2 overflow-x-auto rounded border border-border bg-bg-secondary px-3 py-2 font-mono text-[11px]">
        <span className="shrink-0 select-none text-accent">$</span>
        <code className="whitespace-nowrap text-text-primary">{display}</code>
      </div>
      <div className="mt-2 flex items-center justify-between gap-2">
        <p className="text-[10px] text-text-tertiary">{t('integrations.auto_cmd_hint')}</p>
        <CopyButton text={real} label={t('common.copy')} copiedLabel={t('common.copied')} />
      </div>
    </div>
  )
}

function ToolCard({
  tool,
  apiKey,
  baseUrl,
  baseUrlV1,
  provider,
  models,
  defaultModel,
}: {
  tool: IntegrationTool
  apiKey: string
  baseUrl: string
  baseUrlV1: string
  provider: string
  models: string[]
  defaultModel: string
}) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const [modelParams, setModelParams] = useState<Record<string, string>>({})
  const [selectedModels, setSelectedModels] = useState<string[]>([])
  const [telegramBotToken, setTelegramBotToken] = useState('')
  const [telegramUserId, setTelegramUserId] = useState('')
  const origin = typeof window !== 'undefined' ? window.location.origin : ''

  useEffect(() => {
    if (!tool.modelSlots) return
    const init: Record<string, string> = {}
    for (const slot of tool.modelSlots) {
      let val = slot.key === 'haiku' || slot.key === 'sonnet' || slot.key === 'opus' ? slot.default : defaultModel || slot.default
      if (models.length > 0 && !models.includes(val)) val = defaultModel || models[0]
      init[slot.key] = val
    }
    setModelParams(init)
  }, [defaultModel, models, tool.modelSlots])

  useEffect(() => {
    if (!tool.multiModel) return
    const preferred = models.slice(0, Math.min(5, models.length))
    if (preferred.length > 0) {
      setSelectedModels(preferred.includes(defaultModel) ? preferred : [defaultModel, ...preferred].slice(0, 5))
    } else if (defaultModel) {
      setSelectedModels([defaultModel])
    }
  }, [defaultModel, models, tool.multiModel])

  const templateVars = {
    baseUrl: baseUrlV1,
    apiKey: apiKey || '<YOUR_API_KEY>',
    model: modelParams.model || defaultModel || 'your-model',
  }

  const modelOptions: SelectOption[] = models.map((m) => ({ value: m, label: m }))

  const manualConfigs =
    tool.configType === 'auto'
      ? getManualConfigs({
          toolId: tool.id,
          apiKey,
          baseUrl,
          provider,
          modelParams,
          selectedModels: tool.multiModel ? selectedModels : [modelParams.model || defaultModel],
          telegramBotToken,
          telegramUserId,
        })
      : []

  return (
    <div className="overflow-hidden rounded-xl border border-border bg-bg-secondary transition-colors hover:border-accent/25">
      <button
        type="button"
        onClick={() => setExpanded((e) => !e)}
        className="flex w-full cursor-pointer items-center justify-between gap-3 p-4 text-left transition-colors hover:bg-bg-tertiary/40"
      >
        <div className="flex min-w-0 items-center gap-3">
          <ToolLogo tool={tool} />
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm font-semibold text-text-primary">{tool.name}</span>
              <Badge variant={tool.configType === 'auto' ? 'success' : 'default'}>
                {tool.configType === 'auto' ? t('integrations.badge_auto') : t('integrations.badge_guide')}
              </Badge>
              {tool.telegram && <Badge variant="default">{t('integrations.badge_telegram')}</Badge>}
            </div>
            <p className="mt-0.5 line-clamp-1 text-[11px] text-text-tertiary">{t(tool.descKey)}</p>
          </div>
        </div>
        <svg
          className={`size-4 shrink-0 text-text-tertiary transition-transform ${expanded ? 'rotate-180' : ''}`}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        >
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </button>

      {expanded && (
        <div className="space-y-4 border-t border-border p-4">
          {tool.installCmd && (
            <p className="text-[11px] text-text-tertiary">
              {t('integrations.install')}:{' '}
              <code className="rounded border border-border bg-bg-primary px-1 font-mono">{tool.installCmd}</code>
            </p>
          )}

          {tool.configType === 'auto' && tool.modelSlots && (
            <div className="space-y-2">
              <p className="text-xs font-medium text-text-primary">{t('integrations.model_config')}</p>
              {tool.modelSlots.map((slot) => (
                <div key={slot.key} className="grid grid-cols-[6rem_1fr] items-center gap-2">
                  <span className="text-right text-xs text-text-tertiary">{slot.label}</span>
                  <Select
                      value={modelParams[slot.key] ?? ''}
                      onChange={(val) => setModelParams((p) => ({ ...p, [slot.key]: val }))}
                    options={
                      slot.key === 'subagentModel'
                        ? [{ value: '', label: t('integrations.same_as_main') }, ...modelOptions]
                        : modelOptions
                    }
                    className="text-xs"
                  />
                </div>
              ))}
            </div>
          )}

          {tool.multiModel && models.length > 0 && (
            <div className="space-y-2">
              <p className="text-xs font-medium text-text-primary">{t('integrations.multi_model')}</p>
              <p className="text-[10px] text-text-tertiary">{t('integrations.multi_model_hint')}</p>
              <ModelLimitPicker
                models={models.map((name) => ({ name }))}
                selected={selectedModels}
                onChange={setSelectedModels}
              />
            </div>
          )}

          {tool.telegram && (
            <div className="space-y-3 rounded-lg border border-border/80 bg-bg-primary/40 p-3">
              <p className="text-xs font-semibold text-text-primary">{t('integrations.telegram_title')}</p>
              <ol className="list-decimal space-y-1 pl-4 text-[11px] text-text-secondary">
                <li>
                  {t('integrations.telegram_step_bot')}{' '}
                  <a href={TELEGRAM_BOTFATHER} target="_blank" rel="noopener noreferrer" className="text-accent hover:underline">
                    @BotFather
                  </a>
                </li>
                <li>
                  {t('integrations.telegram_step_userid')}{' '}
                  <a href={TELEGRAM_USERINFO_BOT} target="_blank" rel="noopener noreferrer" className="text-accent hover:underline">
                    @userinfobot
                  </a>
                </li>
                <li>{t('integrations.telegram_step_paste')}</li>
              </ol>
              <Input
                value={telegramBotToken}
                onChange={(e) => setTelegramBotToken(e.target.value)}
                placeholder={t('integrations.telegram_token_placeholder')}
                className="font-mono text-xs"
              />
              <Input
                value={telegramUserId}
                onChange={(e) => setTelegramUserId(e.target.value.replace(/\D/g, ''))}
                placeholder={t('integrations.telegram_userid_placeholder')}
                className="font-mono text-xs"
              />
            </div>
          )}

          {tool.configType === 'auto' && (
            <>
              <SetupCommandBox
                origin={origin}
                toolId={tool.id}
                apiKey={apiKey}
                serverUrl={baseUrl}
                provider={provider}
                modelParams={modelParams}
                selectedModels={tool.multiModel ? selectedModels : [modelParams.model || defaultModel]}
                telegramBotToken={telegramBotToken}
                telegramUserId={telegramUserId}
              />
              {manualConfigs.length > 0 && (
                <div className="space-y-3">
                  <div>
                    <p className="text-xs font-semibold text-text-primary">{t('integrations.manual_config')}</p>
                    <p className="text-[10px] text-text-tertiary">{t('integrations.manual_config_hint')}</p>
                  </div>
                  {manualConfigs.map((cfg) => (
                    <ManualConfigBox key={cfg.filename} filename={cfg.filename} content={cfg.content} />
                  ))}
                </div>
              )}
            </>
          )}

          {tool.configType === 'guide' && tool.guideSteps && (
            <div className="space-y-3">
              {tool.guideSteps.map((step, idx) => (
                <div key={idx} className="space-y-1">
                  <p className="text-xs font-semibold text-text-primary">{step.title}</p>
                  {step.desc && <p className="text-[11px] text-text-tertiary">{step.desc}</p>}
                  {step.code && (
                    <div className="relative">
                      <pre className="overflow-x-auto rounded-lg border border-border bg-bg-primary px-3 py-2 font-mono text-[10px] leading-relaxed">
                        {interpolate(step.code, templateVars)}
                      </pre>
                      {step.copyable !== false && (
                        <div className="absolute right-1.5 top-1.5">
                          <CopyButton
                            text={interpolate(step.code, templateVars)}
                            label={t('common.copy')}
                            copiedLabel={t('common.copied')}
                            className="bg-bg-secondary/90"
                          />
                        </div>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {tool.docsUrl && (
            <a
              href={tool.docsUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex text-[11px] text-accent hover:underline"
            >
              {t('integrations.official_docs')}
            </a>
          )}
        </div>
      )}
    </div>
  )
}

export function IntegrationsTab() {
  const { t } = useTranslation()
  const { data: site } = useSiteConfig()
  const { data: keysData, isLoading: keysLoading } = useAPIKeys()
  const [selectedKeyId, setSelectedKeyId] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [baseUrl, setBaseUrl] = useState('')
  const [defaultModel, setDefaultModel] = useState('')
  const [search, setSearch] = useState('')
  const [showKey, setShowKey] = useState(false)

  const activeKeys = useMemo(
    () => (keysData?.data ?? []).filter((k) => (k.status ?? 'active') === 'active'),
    [keysData],
  )

  const { data: availableModels } = useQuery({
    queryKey: ['available-models'],
    queryFn: () => apiClient<{ models: AvailableModel[] }>('/me/available-models'),
  })

  const models = useMemo(() => (availableModels?.models ?? []).map((m) => m.name), [availableModels])

  const provider = useMemo(() => {
    const name = (site?.system_name ?? 'tavo').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
    return name || 'tavo'
  }, [site?.system_name])

  useEffect(() => {
    const fromSite = site?.server_address?.trim()
    const origin = typeof window !== 'undefined' ? window.location.origin : ''
    const raw = fromSite || origin
    setBaseUrl(raw.endsWith('/v1') ? raw : `${raw.replace(/\/$/, '')}/v1`)
  }, [site?.server_address])

  useEffect(() => {
    if (activeKeys.length === 0) return
    if (!selectedKeyId) {
      setSelectedKeyId(activeKeys[0].id)
    }
  }, [activeKeys, selectedKeyId])

  useEffect(() => {
    if (!selectedKeyId) {
      setApiKey('')
      return
    }
    let cancelled = false
    setApiKey('')
    apiClient<{ key: string }>(`/keys/${selectedKeyId}/reveal`)
      .then((res) => {
        if (!cancelled) setApiKey(res.key)
      })
      .catch(() => {
        if (!cancelled) setApiKey('')
      })
    return () => {
      cancelled = true
    }
  }, [selectedKeyId])

  useEffect(() => {
    if (models.length === 0) return
    if (!defaultModel) {
      const preferred = models.find(
        (m) =>
          m.toLowerCase().includes('gpt-4o') ||
          m.toLowerCase().includes('sonnet') ||
          m.toLowerCase().includes('deepseek'),
      )
      setDefaultModel(preferred ?? models[0])
    }
  }, [models, defaultModel])

  const onKeyChange = useCallback((id: string) => {
    setSelectedKeyId(id)
    setShowKey(false)
  }, [])

  const baseUrlNoV1 = baseUrl.endsWith('/v1') ? baseUrl.slice(0, -3) : baseUrl
  const baseUrlV1 = baseUrl.endsWith('/v1') ? baseUrl : `${baseUrl}/v1`

  const filtered = INTEGRATION_TOOLS.filter(
    (tool) =>
      !search ||
      tool.name.toLowerCase().includes(search.toLowerCase()) ||
      t(tool.descKey).toLowerCase().includes(search.toLowerCase()),
  )

  const keyOptions: SelectOption[] = activeKeys.map((k) => ({
    value: k.id,
    label: `${k.name} (${k.key_hint})`,
  }))

  const modelOptions: SelectOption[] = models.map((m) => ({ value: m, label: m }))

  return (
    <div className="space-y-6">
      <div className="rounded-xl border border-border bg-bg-secondary p-4">
        <h2 className="text-sm font-semibold text-text-primary">{t('integrations.connection_title')}</h2>
        {keysLoading ? (
          <p className="mt-3 animate-pulse text-sm text-text-tertiary">{t('integrations.loading')}</p>
        ) : (
          <div className="mt-4 grid gap-4 md:grid-cols-3">
            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-text-primary">{t('integrations.field_key')}</label>
              {activeKeys.length === 0 ? (
                <p className="text-xs text-error">
                  {t('integrations.no_keys')}{' '}
                  <Link to="/keys" className="font-semibold underline">
                    {t('integrations.create_key')}
                  </Link>
                </p>
              ) : (
                <>
                  <Select value={selectedKeyId} onChange={onKeyChange} options={keyOptions} />
                  <div className="flex items-center gap-2">
                    <Input
                      value={
                        showKey
                          ? apiKey
                          : apiKey
                            ? `${apiKey.slice(0, 8)}${'•'.repeat(Math.min(16, Math.max(0, apiKey.length - 8)))}`
                            : ''
                      }
                      readOnly={!showKey}
                      onChange={(e) => setApiKey(e.target.value)}
                      placeholder={t('integrations.key_paste_hint')}
                      className="font-mono text-xs"
                    />
                    <button
                      type="button"
                      onClick={() => setShowKey((v) => !v)}
                      className="shrink-0 cursor-pointer rounded border border-border px-2 py-1 text-[10px] text-text-tertiary hover:text-text-primary"
                    >
                      {showKey ? t('integrations.hide_key') : t('integrations.show_key')}
                    </button>
                  </div>
                  {!apiKey && (
                    <p className="text-[10px] text-amber-500">{t('integrations.key_session_hint')}</p>
                  )}
                </>
              )}
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-text-primary">{t('integrations.field_model')}</label>
              {models.length === 0 ? (
                <p className="text-xs text-amber-500">{t('integrations.no_models')}</p>
              ) : (
                <Select value={defaultModel} onChange={setDefaultModel} options={modelOptions} />
              )}
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-text-primary">{t('integrations.field_base_url')}</label>
              <Input value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} className="font-mono text-xs" />
              <p className="text-[10px] text-text-tertiary">{t('integrations.base_url_hint')}</p>
            </div>
          </div>
        )}
      </div>

      <div className="space-y-3">
        <div className="flex flex-wrap items-center gap-3">
          <h2 className="text-base font-bold text-text-primary">{t('integrations.tools_title')}</h2>
          <Badge variant="muted">{INTEGRATION_TOOLS.length}</Badge>
          <div className="flex-1" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t('integrations.search')}
            className="h-8 w-48 text-xs"
          />
        </div>

        <div className="grid gap-3 lg:grid-cols-2">
          {filtered.map((tool) => (
            <ToolCard
              key={tool.id}
              tool={tool}
              apiKey={apiKey}
              baseUrl={baseUrlNoV1}
              baseUrlV1={baseUrlV1}
              provider={provider}
              models={models}
              defaultModel={defaultModel}
            />
          ))}
        </div>

        {filtered.length === 0 && (
          <p className="py-8 text-center text-sm text-text-tertiary">{t('integrations.no_tools', { q: search })}</p>
        )}
      </div>

    </div>
  )
}