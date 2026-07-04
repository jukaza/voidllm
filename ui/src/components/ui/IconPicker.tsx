import { getLobeIcon } from '../../lib/lobe-icon'
import { POPULAR_ICONS } from '../../lib/provider-icons'
import { Select } from './Select'
import { Input } from './Input'

interface IconPickerProps {
  value: string
  onChange: (value: string) => void
  label?: string
}

const ICON_OPTIONS = [
  { value: '', label: 'Auto (from slug / model name)' },
  ...POPULAR_ICONS.map((item) => ({ value: item.value, label: item.label })),
]

export function IconPicker({ value, onChange, label = 'Icon' }: IconPickerProps) {
  const inList = value === '' || POPULAR_ICONS.some((i) => i.value === value)
  const selectValue = inList ? value : '__custom__'
  const customValue = inList ? '' : value

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-3">
        <div className="flex size-8 items-center justify-center rounded-md border border-border bg-bg-secondary shrink-0">
          {getLobeIcon(value.trim() || 'OpenAI.Color', 22)}
        </div>
        <div className="flex-1 min-w-0">
          <Select
            label={label}
            options={[
              ...ICON_OPTIONS,
              { value: '__custom__', label: 'Custom @lobehub/icons key…' },
            ]}
            value={selectValue}
            onChange={(v) => {
              if (v === '__custom__') {
                onChange(customValue || 'OpenAI.Color')
                return
              }
              onChange(v)
            }}
            searchable
            placeholder="Select icon"
          />
        </div>
      </div>
      {selectValue === '__custom__' && (
        <Input
          label="Icon key"
          value={customValue}
          onChange={(e) => onChange(e.target.value)}
          placeholder="e.g. OpenAI.Color, Claude.Color"
        />
      )}
      <p className="text-xs text-text-tertiary">
        Powered by{' '}
        <a
          href="https://www.npmjs.com/package/@lobehub/icons"
          target="_blank"
          rel="noreferrer"
          className="text-accent hover:underline"
        >
          @lobehub/icons
        </a>
        . Leave empty to infer from slug or model name.
      </p>
    </div>
  )
}