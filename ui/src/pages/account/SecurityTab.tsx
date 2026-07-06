import { useEffect, useState } from 'react'
import { Button } from '../../components/ui/Button'
import { useMe } from '../../hooks/useMe'
import { useTranslation } from '../../lib/i18n'
import { LiveBadge } from '../settings/components/PreviewBadge'
import { SettingsSectionCard } from '../settings/components/SettingsSectionCard'
import { ChangePasswordDialog } from './components/ChangePasswordDialog'
import { SetPasswordDialog } from './components/SetPasswordDialog'
import { SessionsDialog } from './components/SessionsDialog'

export function SecurityTab() {
  const { t } = useTranslation()
  const { data: me, refetch: refetchMe } = useMe()

  useEffect(() => {
    void refetchMe()
  }, [refetchMe])

  const [showPassword, setShowPassword] = useState(false)
  const [showSetPassword, setShowSetPassword] = useState(false)
  const [showSessions, setShowSessions] = useState(false)

  if (!me) {
    return (
      <div className="space-y-6">
        <div className="h-40 animate-pulse rounded-lg border border-border bg-bg-secondary" />
        <div className="h-48 animate-pulse rounded-lg border border-border bg-bg-secondary" />
      </div>
    )
  }

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

        <SettingsSectionCard
          title={t('account.sessions_title')}
          description={t('account.sessions_count', { count: me.active_session_count })}
          badge={<LiveBadge />}
        >
          <Button size="sm" variant="secondary" onClick={() => setShowSessions(true)}>
            {t('account.sessions_manage')}
          </Button>
        </SettingsSectionCard>
      </div>

      <ChangePasswordDialog open={showPassword} onClose={() => setShowPassword(false)} />
      <SetPasswordDialog open={showSetPassword} onClose={() => setShowSetPassword(false)} />
      <SessionsDialog open={showSessions} onClose={() => setShowSessions(false)} />
    </>
  )
}