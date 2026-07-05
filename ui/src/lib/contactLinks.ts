/** Normalize support channel input to a clickable external URL. */
export function contactHref(value: string): string | null {
  const trimmed = value.trim()
  if (!trimmed) return null
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  if (trimmed.startsWith('@')) return `https://t.me/${trimmed.slice(1)}`
  return null
}