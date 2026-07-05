import { Toggle } from '../../components/ui/Toggle'
import { useToast } from '../../hooks/useToast'
import { useTranslation, type Language } from '../../lib/i18n'
import { LiveBadge, PreviewBadge } from '../settings/components/PreviewBadge'
import { SettingsSectionCard } from '../settings/components/SettingsSectionCard'
import { SettingsTabFooter } from '../settings/components/SettingsTabFooter'
import { useAccountDraft } from './useAccountDraft'

export function PreferencesTab() {
  const { t, language, setLanguage } = useTranslation()
  const { toast } = useToast()
  const { draft, setDraft } = useAccountDraft()

  function save() {
    toast({ variant: 'success', message: t('account.saved_preview') })
  }

  function selectLanguage(lang: Language) {
    setLanguage(lang)
    toast({ variant: 'success', message: t('account.language_changed') })
  }

  return (
    <div className="space-y-6">
      <SettingsSectionCard
        title={t('account.pref_language_title')}
        description={t('account.pref_language_desc')}
        badge={<LiveBadge />}
      >
        <div className="flex flex-wrap gap-2">
          {(['vi', 'en'] as Language[]).map((lang) => (
            <button
              key={lang}
              type="button"
              onClick={() => selectLanguage(lang)}
              className={[
                'rounded-md border px-4 py-2 text-sm font-medium transition-colors',
                language === lang
                  ? 'border-accent bg-accent text-white'
                  : 'border-border bg-bg-tertiary text-text-secondary hover:border-accent/50',
              ].join(' ')}
            >
              {lang === 'vi' ? t('account.pref_language_vi') : t('account.pref_language_en')}
            </button>
          ))}
        </div>
        <p className="text-xs text-text-tertiary">{t('account.pref_language_hint')}</p>
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('account.preferences_title')}
        description={t('account.preferences_desc')}
        badge={<PreviewBadge />}
      >
        <div className="flex items-center justify-between gap-4">
          <div>
            <div className="text-sm font-medium text-text-primary">{t('account.pref_record_ip')}</div>
            <p className="mt-0.5 text-xs text-text-tertiary">{t('account.pref_record_ip_hint')}</p>
          </div>
          <Toggle checked={draft.record_ip} onChange={(v) => setDraft({ record_ip: v })} />
        </div>
      </SettingsSectionCard>

      <SettingsTabFooter mode="preview" onSave={save} />
    </div>
  )
}