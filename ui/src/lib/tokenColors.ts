/** Shared token metric colors — high contrast on dark UI. */
export const TOKEN_CHART_COLORS = {
  input: '#22d3ee',
  output: '#fb923c',
  cacheRead: '#34d399',
  cacheWrite: '#fbbf24',
} as const

export type TokenMetricKey = keyof typeof TOKEN_CHART_COLORS

/** Inline styles — avoids Tailwind merge conflicts with text-text-primary. */
export const TOKEN_METRIC_STYLES: Record<TokenMetricKey, { color: string; backgroundColor: string }> = {
  input: { color: TOKEN_CHART_COLORS.input, backgroundColor: 'rgba(34, 211, 238, 0.14)' },
  output: { color: TOKEN_CHART_COLORS.output, backgroundColor: 'rgba(251, 146, 60, 0.14)' },
  cacheRead: { color: TOKEN_CHART_COLORS.cacheRead, backgroundColor: 'rgba(52, 211, 153, 0.14)' },
  cacheWrite: { color: TOKEN_CHART_COLORS.cacheWrite, backgroundColor: 'rgba(251, 191, 36, 0.14)' },
}

/** @deprecated Prefer TOKEN_METRIC_STYLES — kept for table cells that only need text color. */
export const TOKEN_COLOR_CLASSES = {
  input: 'text-sky-400',
  output: 'text-orange-400',
  cacheRead: 'text-emerald-400',
  cacheWrite: 'text-amber-400',
} as const

/** @deprecated Prefer TOKEN_METRIC_STYLES */
export const TOKEN_BG_CLASSES = {
  input: 'bg-sky-400/15',
  output: 'bg-orange-400/15',
  cacheRead: 'bg-emerald-400/15',
  cacheWrite: 'bg-amber-400/15',
} as const

export interface TokenBarSegment {
  value: number
  color: string
}

/** Stacked bar segments: fresh input → cache read → output. */
export function tokenBarSegments(
  promptTokens: number,
  completionTokens: number,
  cachedTokens = 0,
): TokenBarSegment[] {
  const freshInput = Math.max(0, promptTokens - cachedTokens)
  const segments: TokenBarSegment[] = []
  if (freshInput > 0) segments.push({ value: freshInput, color: TOKEN_CHART_COLORS.input })
  if (cachedTokens > 0) segments.push({ value: cachedTokens, color: TOKEN_CHART_COLORS.cacheRead })
  if (completionTokens > 0) segments.push({ value: completionTokens, color: TOKEN_CHART_COLORS.output })
  return segments
}