import type { RouteStepDraft } from './ComboRouteEditor'
import type { UpstreamModelItem } from '../../hooks/useUpstreamModels'

export const PRODUCT_NAME_REGEX = /^[a-zA-Z0-9_.\-]+$/

export function stepKey(providerId: string, upstreamId: string) {
  return `${providerId}\0${upstreamId}`
}

export function stepToKey(step: RouteStepDraft) {
  return stepKey(step.provider_id, step.upstream_model)
}

export function formatStepLabel(step: RouteStepDraft) {
  const provider = step.provider_name ?? 'provider'
  return `${provider}/${step.upstream_model}`
}

export function upstreamToStep(item: UpstreamModelItem): RouteStepDraft {
  return {
    provider_id: item.provider_id,
    upstream_model: item.upstream_id,
    is_enabled: true,
    provider_name: item.provider_name,
  }
}

export interface ProviderModelGroup {
  providerId: string
  providerName: string
  models: UpstreamModelItem[]
}

export function providerDisplayName(m: UpstreamModelItem): string {
  if (m.provider_name?.trim()) return m.provider_name.trim()
  if (m.provider_slug?.trim()) return m.provider_slug.trim()
  return 'Provider'
}

export function groupUpstreamModels(models: UpstreamModelItem[]): ProviderModelGroup[] {
  const map = new Map<string, ProviderModelGroup>()
  for (const m of models) {
    const existing = map.get(m.provider_id)
    if (existing) {
      existing.models.push(m)
    } else {
      map.set(m.provider_id, {
        providerId: m.provider_id,
        providerName: providerDisplayName(m),
        models: [m],
      })
    }
  }
  return [...map.values()].sort((a, b) => a.providerName.localeCompare(b.providerName))
}