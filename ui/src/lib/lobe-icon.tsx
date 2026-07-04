import type { ReactNode } from 'react'
import * as LobeIcons from '@lobehub/icons'

function parseValue(raw: string | undefined | null): string | number | boolean {
  if (raw == null) return true

  let v = String(raw).trim()
  if (v.startsWith('{') && v.endsWith('}')) v = v.slice(1, -1).trim()
  if (
    (v.startsWith('"') && v.endsWith('"')) ||
    (v.startsWith("'") && v.endsWith("'"))
  ) {
    return v.slice(1, -1)
  }
  if (v === 'true') return true
  if (v === 'false') return false
  if (/^-?\d+(?:\.\d+)?$/.test(v)) return Number(v)
  return v
}

function FallbackIcon({ label, size }: { label: string; size: number }) {
  const letter = label.charAt(0).toUpperCase() || '?'
  return (
    <div
      className="bg-bg-tertiary text-text-tertiary flex items-center justify-center rounded-md text-xs font-medium shrink-0"
      style={{ width: size, height: size }}
      aria-hidden="true"
    >
      {letter}
    </div>
  )
}

/**
 * Render an icon from @lobehub/icons by key.
 * Examples: "OpenAI.Color", "Claude.Color", "DeepSeek.Color"
 */
export function getLobeIcon(
  iconName: string | undefined | null,
  size = 20,
): ReactNode {
  if (!iconName?.trim()) {
    return <FallbackIcon label="?" size={size} />
  }

  const trimmedName = iconName.trim()

  // Legacy asset paths or external URLs — show as image.
  if (
    trimmedName.startsWith('http://') ||
    trimmedName.startsWith('https://') ||
    trimmedName.startsWith('/')
  ) {
    return (
      <img
        src={trimmedName}
        alt=""
        className="rounded-md object-contain shrink-0"
        style={{ width: size, height: size }}
      />
    )
  }

  const segments = trimmedName.split('.')
  const baseKey = segments[0]
  const BaseIcon = (LobeIcons as Record<string, unknown>)[baseKey] as
    | Record<string, unknown>
    | undefined

  let IconComponent: React.ComponentType<Record<string, unknown>> | undefined
  let propStartIndex: number

  if (BaseIcon && segments.length > 1 && BaseIcon[segments[1]]) {
    IconComponent = BaseIcon[segments[1]] as React.ComponentType<Record<string, unknown>>
    propStartIndex = 2
  } else {
    IconComponent = (LobeIcons as Record<string, unknown>)[baseKey] as
      | React.ComponentType<Record<string, unknown>>
      | undefined
    propStartIndex = segments.length > 1 && /^[A-Z]/.test(segments[1]) ? 2 : 1
  }

  if (
    !IconComponent ||
    (typeof IconComponent !== 'function' && typeof IconComponent !== 'object')
  ) {
    return <FallbackIcon label={trimmedName} size={size} />
  }

  const props: Record<string, string | number | boolean> = {}
  for (let i = propStartIndex; i < segments.length; i++) {
    const seg = segments[i]
    if (!seg) continue
    const eqIdx = seg.indexOf('=')
    if (eqIdx === -1) {
      props[seg.trim()] = true
      continue
    }
    props[seg.slice(0, eqIdx).trim()] = parseValue(seg.slice(eqIdx + 1).trim())
  }
  if (props.size == null) props.size = size

  return <IconComponent {...props} />
}