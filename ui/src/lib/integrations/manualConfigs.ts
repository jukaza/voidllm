import type { IntegrationToolId } from './tools'

export interface ManualConfigFile {
  filename: string
  content: string
  lang?: string
}

export interface ManualConfigParams {
  toolId: IntegrationToolId
  apiKey: string
  baseUrl: string
  provider: string
  modelParams: Record<string, string>
  selectedModels?: string[]
  telegramBotToken?: string
  telegramUserId?: string
}

function kiloModelsJSON(models: string[]): string {
  const entries = models.map((m) => `        "${m}": {"name": "${m}"}`).join(',\n')
  return `{\n${entries}\n      }`
}

export function getManualConfigs(params: ManualConfigParams): ManualConfigFile[] {
  const {
    toolId,
    apiKey,
    baseUrl,
    provider,
    modelParams,
    selectedModels = [],
    telegramBotToken = '',
    telegramUserId = '',
  } = params

  const baseUrlV1 = baseUrl.endsWith('/v1') ? baseUrl : `${baseUrl}/v1`
  const baseUrlNoV1 = baseUrl.endsWith('/v1') ? baseUrl.slice(0, -3) : baseUrl
  const key = apiKey || '<YOUR_API_KEY>'
  const model = modelParams.model || selectedModels[0] || 'gpt-4o'
  const haiku = modelParams.haiku || 'claude-haiku-4-5'
  const sonnet = modelParams.sonnet || 'claude-sonnet-4-5'
  const opus = modelParams.opus || 'claude-opus-4-5'
  const subagentModel = modelParams.subagentModel || model
  const kiloModels = selectedModels.length > 0 ? selectedModels : [model]

  switch (toolId) {
    case 'claude':
      return [
        {
          filename: '~/.claude/settings.json',
          content: JSON.stringify(
            {
              hasCompletedOnboarding: true,
              env: {
                ANTHROPIC_BASE_URL: baseUrlNoV1,
                ANTHROPIC_AUTH_TOKEN: key,
                ANTHROPIC_DEFAULT_HAIKU_MODEL: haiku,
                ANTHROPIC_DEFAULT_SONNET_MODEL: sonnet,
                ANTHROPIC_DEFAULT_OPUS_MODEL: opus,
              },
            },
            null,
            2,
          ),
          lang: 'json',
        },
      ]

    case 'codex':
      return [
        {
          filename: '~/.codex/config.toml',
          content: `model = "${model}"\nmodel_provider = "${provider}"\n\n[model_providers.${provider}]\nname = "${provider}"\nbase_url = "${baseUrlV1}"\nwire_api = "responses"\n\n[agents.subagent]\nmodel = "${subagentModel}"`,
          lang: 'toml',
        },
        {
          filename: '~/.codex/auth.json',
          content: JSON.stringify({ OPENAI_API_KEY: key, auth_mode: 'apikey' }, null, 2),
          lang: 'json',
        },
      ]

    case 'cline':
      return [
        {
          filename: '~/.cline/data/globalState.json',
          content: JSON.stringify(
            {
              actModeApiProvider: 'openai',
              planModeApiProvider: 'openai',
              openAiBaseUrl: baseUrlNoV1,
              openAiModelId: model,
              planModeOpenAiModelId: model,
            },
            null,
            2,
          ),
          lang: 'json',
        },
        {
          filename: '~/.cline/data/secrets.json',
          content: JSON.stringify({ openAiApiKey: key }, null, 2),
          lang: 'json',
        },
      ]

    case 'kilo':
      return [
        {
          filename: '~/.config/kilo/kilo.jsonc',
          content: `{
  "$schema": "https://app.kilo.ai/config.json",
  "enabled_providers": ["${provider}"],
  "provider": {
    "${provider}": {
      "api": "openai",
      "options": {
        "apiKey": "${key}",
        "baseURL": "${baseUrlV1}"
      },
      "models": ${kiloModelsJSON(kiloModels)}
    }
  },
  "model": "${provider}/${model}"
}`,
          lang: 'json',
        },
      ]

    case 'hermes': {
      const envLines = [`OPENAI_API_KEY=${key}`]
      if (telegramBotToken) envLines.push(`TELEGRAM_BOT_TOKEN=${telegramBotToken}`)
      if (telegramUserId) envLines.push(`TELEGRAM_ALLOWED_USERS=${telegramUserId}`)
      return [
        {
          filename: '~/.hermes/config.yaml',
          content: `model:\n  default: "${model}"\n  provider: "custom"\n  base_url: "${baseUrlV1}"`,
          lang: 'yaml',
        },
        {
          filename: '~/.hermes/.env',
          content: envLines.join('\n'),
        },
      ]
    }

    case 'openclaw': {
      const base: Record<string, unknown> = {
        api_base: baseUrlV1,
        api_key: key,
        model,
      }
      if (telegramBotToken) {
        base.channels = {
          telegram: {
            enabled: true,
            botToken: telegramBotToken,
            dmPolicy: 'allowlist',
            allowFrom: telegramUserId ? [telegramUserId] : [],
          },
        }
      }
      return [
        {
          filename: '~/.openclaw/openclaw.json',
          content: JSON.stringify(base, null, 2),
          lang: 'json',
        },
      ]
    }

    case 'opencode':
      return [
        {
          filename: '~/.config/opencode/opencode.json',
          content: JSON.stringify(
            {
              selected_provider: provider,
              provider: {
                [provider]: {
                  npm: '@ai-sdk/openai-compatible',
                  options: { baseURL: baseUrlV1, apiKey: key },
                  models: { [model]: {} },
                },
              },
              default_model: `${provider}/${model}`,
            },
            null,
            2,
          ),
          lang: 'json',
        },
      ]

    case 'jcode':
      return [
        {
          filename: '~/.jcode/config.toml',
          content: `[model_providers.${provider}]\napi_key = "${key}"\nbase_url = "${baseUrlV1}"`,
          lang: 'toml',
        },
        {
          filename: `~/.config/jcode/provider-${provider}.env`,
          content: `OPENAI_API_KEY=${key}\nOPENAI_API_BASE=${baseUrlV1}`,
        },
      ]

    default:
      return []
  }
}