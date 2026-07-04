import { twMerge } from 'tailwind-merge'

/** Merge class names using tailwind-merge to resolve conflicting Tailwind classes. */
export function cn(...classes: (string | false | null | undefined)[]): string {
  return twMerge(classes.filter(Boolean).join(' '))
}

/** Format a number with locale-aware separators. */
export function formatNumber(n: number): string {
  return new Intl.NumberFormat().format(n)
}

/** Format a token count with locale-aware separators. */
export function formatTokens(n: number): string {
  return new Intl.NumberFormat().format(n)
}

/** Format a number as VND currency. */
export function formatCost(n: number): string {
  return new Intl.NumberFormat('vi-VN', {
    style: 'currency',
    currency: 'VND',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(n)
}

/** Format an ISO UTC timestamp in the user's local timezone. */
export function formatDate(iso: string): string {
  return new Date(iso).toLocaleString()
}

/** Truncate a UUID or long ID for use as a secondary display hint. */
export function shortenId(id: string): string {
  if (!id) return ''
  return id.length <= 12 ? id : `${id.slice(0, 8)}…`
}

/** Truncate a request ID for table display; full value stays in title/tooltip. */
export function truncateRequestId(id: string, head = 10, tail = 6): string {
  if (!id) return ''
  if (id.length <= head + tail + 1) return id
  return `${id.slice(0, head)}…${id.slice(-tail)}`
}
