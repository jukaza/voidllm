/** Legacy /logos/*.svg paths from older presets → @lobehub/icons keys */
const LEGACY_LOGO_PATH: Record<string, string> = {
  '/logos/openai.svg': 'OpenAI.Color',
  '/logos/anthropic.svg': 'Claude.Color',
  '/logos/gemini.svg': 'Gemini.Color',
  '/logos/deepseek.svg': 'DeepSeek.Color',
  '/logos/openrouter.svg': 'OpenRouter',
  '/logos/groq.svg': 'Groq',
  '/logos/xai.svg': 'XAI',
  '/logos/mistral.svg': 'Mistral.Color',
  '/logos/siliconflow.svg': 'SiliconCloud.Color',
  '/logos/moonshot.svg': 'Kimi.Color',
  '/logos/zhipu.svg': 'Zhipu.Color',
  '/logos/ollama.svg': 'Ollama',
  '/logos/azure.svg': 'AzureAI',
  '/logos/vertex.svg': 'Gemini.Color',
  '/logos/custom.svg': 'LobeHub',
}

function normalizeLogoKey(logo?: string | null): string | undefined {
  const v = logo?.trim()
  if (!v) return undefined
  return LEGACY_LOGO_PATH[v] ?? v
}

/** Preset slug / provider id → default @lobehub/icons key */
export const PRESET_ICON_BY_SLUG: Record<string, string> = {
  openai: 'OpenAI.Color',
  anthropic: 'Claude.Color',
  gemini: 'Gemini.Color',
  deepseek: 'DeepSeek.Color',
  openrouter: 'OpenRouter',
  groq: 'Groq',
  xai: 'XAI',
  mistral: 'Mistral.Color',
  siliconflow: 'SiliconCloud.Color',
  moonshot: 'Kimi.Color',
  zhipu: 'Zhipu.Color',
  ollama: 'Ollama',
  'ollama-cloud': 'Ollama',
  azure: 'AzureAI',
  vertex: 'Gemini.Color',
  custom: 'LobeHub',
}

export interface PopularIconOption {
  value: string
  label: string
}

export const POPULAR_ICONS: PopularIconOption[] = [
  { value: 'OpenAI.Color', label: 'OpenAI' },
  { value: 'Claude.Color', label: 'Claude / Anthropic' },
  { value: 'Gemini.Color', label: 'Gemini / Google' },
  { value: 'DeepSeek.Color', label: 'DeepSeek' },
  { value: 'XAI', label: 'Grok / xAI' },
  { value: 'Mistral.Color', label: 'Mistral' },
  { value: 'Groq', label: 'Groq' },
  { value: 'OpenRouter', label: 'OpenRouter' },
  { value: 'Kimi.Color', label: 'Kimi / Moonshot' },
  { value: 'Zhipu.Color', label: 'GLM / Zhipu' },
  { value: 'Qwen.Color', label: 'Qwen / Alibaba' },
  { value: 'SiliconCloud.Color', label: 'SiliconFlow' },
  { value: 'Ollama', label: 'Ollama / Llama' },
  { value: 'AzureAI', label: 'Azure' },
  { value: 'Cohere.Color', label: 'Cohere' },
  { value: 'Minimax.Color', label: 'MiniMax' },
  { value: 'Doubao.Color', label: 'Doubao' },
  { value: 'LobeHub', label: 'Custom / Other' },
]

const MODEL_NAME_RULES: Array<{ test: RegExp; icon: string }> = [
  { test: /^(gpt-|o[134]-|chatgpt|text-embedding)/i, icon: 'OpenAI.Color' },
  { test: /^claude/i, icon: 'Claude.Color' },
  { test: /^gemini/i, icon: 'Gemini.Color' },
  { test: /^deepseek/i, icon: 'DeepSeek.Color' },
  { test: /^grok/i, icon: 'XAI' },
  { test: /^mistral|mixtral|codestral/i, icon: 'Mistral.Color' },
  { test: /^llama|meta-/i, icon: 'Ollama' },
  { test: /^qwen/i, icon: 'Qwen.Color' },
  { test: /^glm-/i, icon: 'Zhipu.Color' },
  { test: /^kimi|moonshot/i, icon: 'Kimi.Color' },
  { test: /^command/i, icon: 'Cohere.Color' },
]

/** Resolve icon key for a provider from stored logo, slug, or protocol. */
export function resolveProviderIcon(
  logo?: string | null,
  slug?: string | null,
  protocol?: string | null,
): string {
  const normalized = normalizeLogoKey(logo)
  if (normalized) return normalized
  const s = slug?.trim().toLowerCase()
  if (s && PRESET_ICON_BY_SLUG[s]) return PRESET_ICON_BY_SLUG[s]
  const p = protocol?.trim().toLowerCase()
  if (p && PRESET_ICON_BY_SLUG[p]) return PRESET_ICON_BY_SLUG[p]
  return 'LobeHub'
}

/** Resolve icon for a catalog model from stored logo or model name. */
export function resolveModelIcon(logo?: string | null, modelName?: string | null): string {
  const normalized = normalizeLogoKey(logo)
  if (normalized) return normalized
  const name = modelName?.trim() ?? ''
  for (const rule of MODEL_NAME_RULES) {
    if (rule.test.test(name)) return rule.icon
  }
  return 'LobeHub'
}

/** Default icon when creating a provider from slug/protocol. */
export function defaultIconForSlug(slug: string, protocol?: string): string {
  return resolveProviderIcon(null, slug, protocol)
}