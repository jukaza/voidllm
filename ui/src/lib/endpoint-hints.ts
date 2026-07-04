import type { TranslationKey } from './i18n'

/** i18n key for endpoint placeholder hint under a wire protocol. */
export function endpointHintKey(protocol?: string | null): TranslationKey {
  const p = protocol?.trim().toLowerCase()
  switch (p) {
    case 'anthropic':
      return 'providers.endpoint_hint_anthropic'
    case 'gemini':
      return 'providers.endpoint_hint_gemini'
    case 'azure':
      return 'providers.endpoint_hint_azure'
    case 'vertex':
      return 'providers.endpoint_hint_vertex'
    case 'vllm':
    case 'ollama':
      return 'providers.endpoint_hint_self_hosted'
    case 'openai':
    case 'custom':
    default:
      return 'providers.endpoint_hint_openai'
  }
}

/** i18n key when upstream model id conflicts with the selected wire protocol. */
export function wireMismatchKey(
  protocol?: string | null,
  upstreamModel?: string | null,
): TranslationKey | null {
  const id = upstreamModel?.trim().toLowerCase()
  if (!id) return null
  const p = protocol?.trim().toLowerCase() || 'openai'
  if (id.startsWith('claude-') && p !== 'anthropic') {
    return 'providers.wire_mismatch_claude'
  }
  if (id.startsWith('gemini-') && p !== 'gemini' && p !== 'vertex') {
    return 'providers.wire_mismatch_gemini'
  }
  return null
}

export function modelIdsIncludeClaude(ids: Iterable<string>): boolean {
  for (const id of ids) {
    if (id.trim().toLowerCase().startsWith('claude-')) return true
  }
  return false
}