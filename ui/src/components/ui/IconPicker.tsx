import { useMemo } from 'react'
import { getLobeIcon } from '../../lib/lobe-icon'
import { POPULAR_ICONS } from '../../lib/provider-icons'
import { useTranslation } from '../../lib/i18n'
import { Select, type SelectOption } from './Select'
import { Input } from './Input'

interface IconPickerProps {
  value: string
  onChange: (value: string) => void
  label?: string
  /** Icon key used for the Auto option and when value is empty. */
  previewKey?: string
}

const ICON_SIZE = 20

function iconNode(key: string) {
  return (
    <span className="inline-flex items-center justify-center" aria-hidden="true">
      {getLobeIcon(key, ICON_SIZE)}
    </span>
  )
}

export function IconPicker({ value, onChange, label, previewKey }: IconPickerProps) {
  const { t } = useTranslation()
  const inList = value === '' || POPULAR_ICONS.some((i) => i.value === value)
  const selectValue = inList ? value : '__custom__'
  const customValue = inList ? '' : value

  const autoIconKey = previewKey || 'LobeHub'

  const options: SelectOption[] = useMemo(
    () => [
      {
        value: '',
        label: t('models.logo_option_auto'),
        icon: iconNode(autoIconKey),
      },
      ...POPULAR_ICONS.map((item) => ({
        value: item.value,
        label: item.label,
        icon: iconNode(item.value),
      })),
      {
        value: '__custom__',
        label: t('models.logo_option_custom'),
        icon: iconNode(customValue.trim() || 'LobeHub'),
      },
    ],
    [t, autoIconKey, customValue],
  )

  return (
    <div className="space-y-2">
      <Select
        label={label ?? t('models.logo')}
        options={options}
        value={selectValue}
        onChange={(v) => {
          if (v === '__custom__') {
            onChange(customValue || 'OpenAI.Color')
            return
          }
          onChange(v)
        }}
        searchable
        placeholder={t('models.logo_select_placeholder')}
      />
      {selectValue === '__custom__' && (
        <Input
          label={t('models.logo_custom_key')}
          value={customValue}
          onChange={(e) => onChange(e.target.value)}
          placeholder="OpenAI.Color"
        />
      )}
      <p className="text-xs text-text-tertiary">{t('models.logo_desc')}</p>
    </div>
  )
}