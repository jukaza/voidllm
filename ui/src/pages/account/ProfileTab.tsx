import { useEffect, useState } from 'react'
import { Input } from '../../components/ui/Input'
import { useMe } from '../../hooks/useMe'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { LiveBadge, PreviewBadge } from '../settings/components/PreviewBadge'
import { SettingsSectionCard } from '../settings/components/SettingsSectionCard'
import { SettingsTabFooter } from '../settings/components/SettingsTabFooter'
import { useAccountDraft } from './useAccountDraft'

export function ProfileTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data: me } = useMe()
  const { draft, setDraft } = useAccountDraft()

  const [displayName, setDisplayName] = useState('')
  const [error, setError] = useState<string | undefined>()

  useEffect(() => {
    if (!me) return
    const initial = draft.display_name || me.display_name || ''
    setDisplayName(initial)
  }, [me, draft.display_name])

  const baseline = draft.display_name || me?.display_name || ''
  const isDirty = displayName.trim() !== baseline.trim()

  function save() {
    const trimmed = displayName.trim()
    if (!trimmed) {
      setError(t('account.display_name_required'))
      return
    }
    setError(undefined)
    setDraft({ display_name: trimmed })
    toast({ variant: 'success', message: t('account.saved_preview') })
  }

  if (!me) return null

  return (
    <div className="space-y-6">
      <SettingsSectionCard
        title={t('account.profile_identity_title')}
        description={t('account.profile_identity_desc')}
        badge={<PreviewBadge />}
      >
        <Input
          label={t('account.display_name')}
          value={displayName}
          onChange={(e) => {
            setDisplayName(e.target.value)
            if (error) setError(undefined)
          }}
          error={error}
          description={t('account.display_name_hint')}
          required
        />
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('account.profile_email_title')}
        description={t('account.profile_email_desc')}
        badge={<LiveBadge />}
      >
        <Input label={t('account.email')} value={me.email} readOnly className="font-mono text-sm" />
        <p className="text-xs text-text-tertiary">{t('account.email_hint')}</p>
      </SettingsSectionCard>

      <SettingsTabFooter mode="preview" onSave={save} disabled={!isDirty} />
    </div>
  )
}