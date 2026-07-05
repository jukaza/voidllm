import { useState } from 'react'
import { useMe } from '../../hooks/useMe'
import { useTranslation } from '../../lib/i18n'
import { LiveBadge, PreviewBadge } from '../settings/components/PreviewBadge'
import { SettingsSectionCard } from '../settings/components/SettingsSectionCard'
import { ChangePasswordDialog } from './components/ChangePasswordDialog'
import { SecurityActionTiles } from './components/SecurityActionTiles'
import { SessionsDialog } from './components/SessionsDialog'
import { TwoFactorDialog } from './components/TwoFactorDialog'
import { useAccountDraft } from './useAccountDraft'

export function SecurityTab() {
  const { t } = useTranslation()
  const { data: me } = useMe()
  const { draft } = useAccountDraft()

  const [showPassword, setShowPassword] = useState(false)
  const [show2FA, setShow2FA] = useState(false)
  const [showSessions, setShowSessions] = useState(false)

  if (!me) return null

  return (
    <>
      <div className="space-y-6">
        <SettingsSectionCard
          title={t('account.security_title')}
          description={t('account.security_desc')}
          badge={<PreviewBadge />}
        >
          <SecurityActionTiles
            twoFAEnabled={draft.two_fa_enabled}
            sessionCount={draft.sessions.length}
            onTwoFA={() => setShow2FA(true)}
            onPassword={() => setShowPassword(true)}
            onSessions={() => setShowSessions(true)}
          />
        </SettingsSectionCard>

        <SettingsSectionCard
          title={t('account.password_title')}
          description={t('account.password_live_desc')}
          badge={<LiveBadge />}
        >
          <p className="text-sm text-text-tertiary">{t('account.password_live_hint')}</p>
        </SettingsSectionCard>
      </div>

      <ChangePasswordDialog open={showPassword} onClose={() => setShowPassword(false)} />
      <TwoFactorDialog open={show2FA} onClose={() => setShow2FA(false)} email={me.email} />
      <SessionsDialog open={showSessions} onClose={() => setShowSessions(false)} />
    </>
  )
}