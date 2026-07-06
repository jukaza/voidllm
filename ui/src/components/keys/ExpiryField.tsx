import { Input } from '../ui/Input'
import { Toggle } from '../ui/Toggle'
import { useTranslation } from '../../lib/i18n'

/** Format ISO timestamp to YYYY-MM-DD for native date input. */
export function toDateInputValue(iso?: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

/** Convert YYYY-MM-DD (local) to RFC3339 end-of-day. */
export function dateInputToISO(dateStr: string): string {
  const [y, m, d] = dateStr.split('-').map(Number)
  const dt = new Date(y, m - 1, d, 23, 59, 59, 999)
  return dt.toISOString()
}

/** Default expiry: N days from today as YYYY-MM-DD. */
export function defaultExpiryDateInput(days: number): string {
  const dt = new Date()
  dt.setDate(dt.getDate() + days)
  return toDateInputValue(dt.toISOString())
}

export function todayDateInput(): string {
  return toDateInputValue(new Date().toISOString())
}

interface ExpiryFieldProps {
  label: string
  /** ISO expiry or null when never. */
  value: string | null
  onChange: (iso: string | null) => void
  disabled?: boolean
  description?: string
}

export function ExpiryField({ label, value, onChange, disabled, description }: ExpiryFieldProps) {
  const { t } = useTranslation()
  const never = value == null
  const dateValue = never ? '' : toDateInputValue(value)

  return (
    <div className="space-y-3">
      <Toggle
        checked={never}
        onChange={(checked) => {
          if (checked) {
            onChange(null)
            return
          }
          onChange(dateInputToISO(defaultExpiryDateInput(90)))
        }}
        label={t('keys.expiry.never')}
        disabled={disabled}
      />
      {!never && (
        <Input
          label={label}
          type="date"
          value={dateValue}
          min={todayDateInput()}
          onChange={(e) => {
            const next = e.target.value
            if (!next) {
              onChange(null)
              return
            }
            onChange(dateInputToISO(next))
          }}
          disabled={disabled}
          description={description}
          className="[color-scheme:dark] cursor-pointer"
        />
      )}
    </div>
  )
}