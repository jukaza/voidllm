import { Button } from '../../../components/ui/Button'
import { Dialog } from '../../../components/ui/Dialog'
import { Badge } from '../../../components/ui/Badge'
import { useToast } from '../../../hooks/useToast'
import { useTranslation } from '../../../lib/i18n'
import { useRevokeOtherSessions, useRevokeSession, useSessions } from '../../../hooks/useAccountSecurity'

interface SessionsDialogProps {
  open: boolean
  onClose: () => void
}

export function SessionsDialog({ open, onClose }: SessionsDialogProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data: sessions = [], isLoading } = useSessions()
  const revokeSession = useRevokeSession()
  const revokeOthers = useRevokeOtherSessions()

  const hasOthers = sessions.some((s) => !s.current)

  function revokeOne(id: string) {
    revokeSession.mutate(id, {
      onSuccess: () => toast({ variant: 'success', message: t('account.sessions_revoked') }),
      onError: (err) => toast({ variant: 'error', message: err.message }),
    })
  }

  function revokeAllOthers() {
    revokeOthers.mutate(undefined, {
      onSuccess: () => toast({ variant: 'success', message: t('account.sessions_revoked_all') }),
      onError: (err) => toast({ variant: 'error', message: err.message }),
    })
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t('account.sessions_dialog_title')}
      footer={
        hasOthers ? (
          <Button
            variant="destructive"
            size="sm"
            loading={revokeOthers.isPending}
            onClick={revokeAllOthers}
          >
            {t('account.sessions_revoke_all')}
          </Button>
        ) : undefined
      }
    >
      <div className="space-y-2">
        {isLoading && <p className="text-sm text-text-tertiary">{t('common.loading')}</p>}
        {!isLoading && sessions.length === 0 && (
          <p className="text-sm text-text-tertiary">{t('account.sessions_empty')}</p>
        )}
        {sessions.map((session) => (
          <div
            key={session.id}
            className="flex items-center justify-between rounded border border-border p-3 text-sm"
          >
            <div>
              <div className="flex items-center gap-2">
                <span className="font-medium text-text-primary">{session.device_label}</span>
                {session.current && (
                  <Badge variant="info" className="text-[10px]">
                    {t('account.sessions_current')}
                  </Badge>
                )}
              </div>
              <div className="text-xs text-text-tertiary">
                {session.ip || '—'}
                {session.last_seen_at ? ` · ${session.last_seen_at}` : ''}
              </div>
            </div>
            {!session.current && (
              <Button
                size="sm"
                variant="ghost"
                loading={revokeSession.isPending}
                onClick={() => revokeOne(session.id)}
              >
                {t('account.sessions_revoke')}
              </Button>
            )}
          </div>
        ))}
      </div>
    </Dialog>
  )
}