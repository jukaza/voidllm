import { BrandIcon } from '../ui/BrandIcon'
import { Toggle } from '../ui/Toggle'
import { useTranslation } from '../../lib/i18n'

export interface ModelLimitOption {
  name: string
  logo?: string
}

interface ModelLimitPickerProps {
  models: ModelLimitOption[]
  selected: string[]
  onChange: (next: string[]) => void
  disabled?: boolean
}

export function ModelLimitPicker({ models, selected, onChange, disabled }: ModelLimitPickerProps) {
  const { t } = useTranslation()

  function toggle(model: string, on: boolean) {
    if (disabled) return
    if (on) {
      if (!selected.includes(model)) onChange([...selected, model])
      return
    }
    onChange(selected.filter((m) => m !== model))
  }

  if (models.length === 0) {
    return <p className="text-xs text-text-tertiary">{t('playground.no_models')}</p>
  }

  return (
    <div className="max-h-52 overflow-y-auto rounded-lg border border-border bg-bg-primary divide-y divide-border">
      {models.map((model) => {
        const checked = selected.includes(model.name)
        return (
          <div
            key={model.name}
            className="flex items-center justify-between gap-3 px-3 py-2.5 hover:bg-bg-secondary/80 transition-colors"
          >
            <div className="flex min-w-0 items-center gap-2.5">
              <BrandIcon logo={model.logo} modelName={model.name} size={20} />
              <span className="truncate text-sm text-text-primary font-medium">{model.name}</span>
            </div>
            <Toggle
              checked={checked}
              onChange={(on) => toggle(model.name, on)}
              disabled={disabled}
              size="sm"
              aria-label={model.name}
            />
          </div>
        )
      })}
    </div>
  )
}