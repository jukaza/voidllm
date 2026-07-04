export type LatencyVariant = 'fast' | 'warn' | 'slow' | 'unknown'

/** Classify total request duration for color coding (LLM-oriented thresholds). */
export function latencyVariant(durationMs: number): LatencyVariant {
  if (durationMs <= 0) return 'unknown'
  if (durationMs < 3_000) return 'fast'
  if (durationMs < 10_000) return 'warn'
  return 'slow'
}

export function formatLatencySeconds(durationMs: number): string {
  if (durationMs <= 0) return '—'
  const sec = durationMs / 1000
  return sec < 10 ? `${sec.toFixed(1)}s` : `${Math.round(sec)}s`
}