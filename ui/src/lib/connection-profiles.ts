/** How provider/channel forms expose connection fields per wire protocol. */
export type ConnectionProfile = {
  /** Preset supplies base URL — hide on key-only cloud providers. */
  hideBaseUrl: boolean
  /** Channel inherits URL/key from linked provider — hide on deployment form. */
  channelInheritsConnection: boolean
  baseUrlRequired: boolean
  baseUrlPlaceholder: string
  keyLabel: string
  keyPlaceholder: string
  showAzureFields: boolean
  showVertexFields: boolean
}

const PROFILES: Record<string, ConnectionProfile> = {
  gemini: {
    hideBaseUrl: true,
    channelInheritsConnection: true,
    baseUrlRequired: false,
    baseUrlPlaceholder: 'https://generativelanguage.googleapis.com',
    keyLabel: 'API Key',
    keyPlaceholder: 'AIza...',
    showAzureFields: false,
    showVertexFields: false,
  },
  anthropic: {
    hideBaseUrl: true,
    channelInheritsConnection: true,
    baseUrlRequired: false,
    baseUrlPlaceholder: 'https://api.anthropic.com',
    keyLabel: 'API Key',
    keyPlaceholder: 'sk-ant-...',
    showAzureFields: false,
    showVertexFields: false,
  },
  openai: {
    hideBaseUrl: true,
    channelInheritsConnection: true,
    baseUrlRequired: false,
    baseUrlPlaceholder: 'https://api.openai.com/v1',
    keyLabel: 'API Key',
    keyPlaceholder: 'sk-...',
    showAzureFields: false,
    showVertexFields: false,
  },
  azure: {
    hideBaseUrl: false,
    channelInheritsConnection: true,
    baseUrlRequired: true,
    baseUrlPlaceholder: 'https://{resource}.openai.azure.com',
    keyLabel: 'API Key',
    keyPlaceholder: 'Azure API key',
    showAzureFields: true,
    showVertexFields: false,
  },
  vertex: {
    hideBaseUrl: false,
    channelInheritsConnection: true,
    baseUrlRequired: true,
    baseUrlPlaceholder: 'https://us-central1-aiplatform.googleapis.com',
    keyLabel: 'Access token / API key',
    keyPlaceholder: 'Bearer token or API key',
    showAzureFields: false,
    showVertexFields: true,
  },
  vllm: {
    hideBaseUrl: false,
    channelInheritsConnection: true,
    baseUrlRequired: true,
    baseUrlPlaceholder: 'http://your-host:8000/v1',
    keyLabel: 'API Key (optional)',
    keyPlaceholder: 'Leave blank if none',
    showAzureFields: false,
    showVertexFields: false,
  },
  ollama: {
    hideBaseUrl: false,
    channelInheritsConnection: true,
    baseUrlRequired: true,
    baseUrlPlaceholder: 'https://your-ollama-host/v1',
    keyLabel: 'API Key (optional)',
    keyPlaceholder: 'Usually not required',
    showAzureFields: false,
    showVertexFields: false,
  },
  custom: {
    hideBaseUrl: false,
    channelInheritsConnection: true,
    baseUrlRequired: true,
    baseUrlPlaceholder: 'https://your-endpoint/v1',
    keyLabel: 'API Key',
    keyPlaceholder: 'sk-...',
    showAzureFields: false,
    showVertexFields: false,
  },
}

const DEFAULT_PROFILE = PROFILES.custom

export function getConnectionProfile(protocol?: string | null): ConnectionProfile {
  const p = protocol?.trim().toLowerCase()
  if (p && PROFILES[p]) return PROFILES[p]
  return DEFAULT_PROFILE
}

/** Cloud presets that only need an API key when using wizard defaults. */
export function isKeyOnlyPreset(preset: { protocol: string; base_url: string }): boolean {
  const profile = getConnectionProfile(preset.protocol)
  return profile.hideBaseUrl && preset.base_url.trim() !== ''
}