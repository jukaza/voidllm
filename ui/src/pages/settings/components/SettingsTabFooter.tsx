import { Button } from '../../../components/ui/Button'
import { useTranslation } from '../../../lib/i18n'

interface SettingsTabFooterProps {
  mode: 'live' | 'preview'
  loading?: boolean
  onSave: () => void
  disabled?: boolean
}

export function SettingsTabFooter({ mode, loading, onSave, disabled }: SettingsTabFooterProps) {
  const { t } = useTranslation()
  return (
    <div className="sticky bottom-0 z-10 -mx-1 mt-6 border-t border-border bg-bg-primary/95 px-1 py-4 backdrop-blur-sm">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-xs text-text-tertiary">
          {mode === 'preview' ? t('settings.footer_preview_hint') : t('settings.footer_live_hint')}
        </p>
        <Button loading={loading} disabled={disabled} onClick={onSave}>
          {mode === 'preview' ? t('settings.save_preview') : t('common.save')}
        </Button>
      </div>
    </div>
  )
}