import type { FinanceRangeDays } from '../../hooks/useFinance'

const VALID_STATUS = new Set(['pending', 'completed', 'expired', 'failed'])
const VALID_TYPE = new Set(['topup', 'usage', 'adjustment', 'refund'])
const VALID_RANGE = new Set<FinanceRangeDays>([7, 30, 90])

export function parseStatusParam(raw: string | null): string {
  if (!raw) return ''
  return VALID_STATUS.has(raw) ? raw : ''
}

export function parseTypeParam(raw: string | null): string {
  if (!raw) return ''
  return VALID_TYPE.has(raw) ? raw : ''
}

export function parseRangeParam(raw: string | null, defaultDays: FinanceRangeDays): FinanceRangeDays {
  if (!raw) return defaultDays
  const n = parseInt(raw.replace('d', ''), 10)
  return VALID_RANGE.has(n as FinanceRangeDays) ? (n as FinanceRangeDays) : defaultDays
}