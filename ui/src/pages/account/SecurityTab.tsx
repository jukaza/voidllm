import { useEffect, useState } from 'react'
import { Button } from '../../components/ui/Button'
import { Banner } from '../../components/ui/Banner'
import { useMe } from '../../hooks/useMe'
import { usePublicAuthConfig } from '../../hooks/useSecuritySettings'
import { useTranslation } from '../../lib/i18n'
import { LiveBadge } from '../settings/components/PreviewBadge'
import { SettingsSectionCard } from '../settings/components/SettingsSectionCard'
import { ChangePasswordDialog } from './components/ChangePasswordDialog'
import { SetPasswordDialog } from './components/SetPasswordDialog'
import { SecurityActionTiles } from './components/SecurityActionTiles'
import { SessionsDialog } from './components/SessionsDialog'
import { TwoFactorDialog } from './components/TwoFactorDialog'

export function SecurityTab() {
  const { t } = useTranslation()
  const { data: me, refetch: refetchMe } = useMe()
  const { data: authConfig } = usePublicAuthConfig()

  useEffect(() => {
    void refetchMe()
  }, [refetchMe])

  const [showPassword, setShowPassword] = useState(false)
  const [showSetPassword, setShowSetPassword] = useState(false)
  const [show2FA, setShow2FA] = useState(false)
  const [showSessions, setShowSessions] = useState(false)

  if (!me) return null

  // Prefer fresh /me; fall back to public config when React Query cache predates policy fields.
  const twoFAAvailable = me.two_fa_available ?? authConfig?.two_fa?.available ?? false
  const show2FACard = twoFAAvailable || me.two_fa_enabled

  return (
    <>
      <div className="space-y-6">
        <SettingsSectionCard
          title={t('account.password_title')}
          description={
            me.has_password ? t('account.password_desc') : t('account.password_oauth_desc')
          }
          badge={<LiveBadge />}
        >
          <Button
            size="sm"
            variant="secondary"
            onClick={() => (me.has_password ? setShowPassword(true) : setShowSetPassword(true))}
          >
            {me.has_password ? t('account.password_change') : t('account.password_set_action')}
          </Button>
        </SettingsSectionCard>

        {show2FACard && (
          <SettingsSectionCard
            title={t('account.twofa_title')}
            description={t('account.twofa_desc')}
            badge={<LiveBadge />}
          >
            <SecurityActionTiles
              twoFAEnabled={me.two_fa_enabled}
              sessionCount={me.active_session_count}
              onTwoFA={() => setShow2FA(true)}
              onPassword={() => undefined}
              onSessions={() => undefined}
              mode="twofa-only"
            />
          </SettingsSectionCard>
        )}

        {!show2FACard && (
          <SettingsSectionCard
            title={t('account.twofa_title')}
            description={t('account.twofa_admin_disabled')}
            badge={<LiveBadge />}
          >
            <Banner variant="info" title={t('account.twofa_unavailable')} />
          </SettingsSectionCard>
        )}

        <SettingsSectionCard
          title={t('account.sessions_title')}
          description={t('account.sessions_count', { count: me.active_session_count })}
          badge={<LiveBadge />}
        >
          <SecurityActionTiles
            twoFAEnabled={me.two_fa_enabled}
            sessionCount={me.active_session_count}
            onTwoFA={() => undefined}
            onPassword={() => undefined}
            onSessions={() => setShowSessions(true)}
            mode="sessions-only"
          />
        </SettingsSectionCard>
      </div>

      <ChangePasswordDialog open={showPassword} onClose={() => setShowPassword(false)} />
      <SetPasswordDialog open={showSetPassword} onClose={() => setShowSetPassword(false)} />
      <TwoFactorDialog open={show2FA} onClose={() => setShow2FA(false)} email={me.email} />
      <SessionsDialog open={showSessions} onClose={() => setShowSessions(false)} />
    </>
  )
}