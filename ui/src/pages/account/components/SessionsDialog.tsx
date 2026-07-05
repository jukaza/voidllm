import { Button } from '../../../components/ui/Button'
import { Dialog } from '../../../components/ui/Dialog'
import { Badge } from '../../../components/ui/Badge'
import { useToast } from '../../../hooks/useToast'
import { useTranslation } from '../../../lib/i18n'
import { useAccountDraft } from '../useAccountDraft'

interface SessionsDialogProps {
  open: boolean
  onClose: () => void
}

export function SessionsDialog({ open, onClose }: SessionsDialogProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { draft, setDraft } = useAccountDraft()

  function revokeSession(id: string) {
    setDraft({ sessions: draft.sessions.filter((s) => s.id !== id) })
    toast({ variant: 'success', message: t('account.sessions_revoked') })
  }

  function revokeAllOthers() {
    const current = draft.sessions.find((s) => s.current) ?? draft.sessions[0]
    setDraft({ sessions: current ? [current] : [] })
    toast({ variant: 'success', message: t('account.sessions_revoked_all') })
  }

  const hasOthers = draft.sessions.some((s) => !s.current) || draft.sessions.length > 1

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t('account.sessions_dialog_title')}
      footer={
        hasOthers ? (
          <Button variant="destructive" size="sm" onClick={revokeAllOthers}>
            {t('account.sessions_revoke_all')}
          </Button>
        ) : undefined
      }
    >
      <div className="space-y-2">
        {draft.sessions.length === 0 && (
          <p className="text-sm text-text-tertiary">{t('account.sessions_empty')}</p>
        )}
        {draft.sessions.map((session) => (
          <div
            key={session.id}
            className="flex items-center justify-between rounded border border-border p-3 text-sm"
          >
            <div>
              <div className="flex items-center gap-2">
                <span className="font-mono text-xs text-text-tertiary">{session.ip}</span>
                {session.current && (
                  <Badge variant="info" className="text-[10px]">
                    {t('account.sessions_current')}
                  </Badge>
                )}
              </div>
              <div>
                {session.device} • {session.lastActive}
              </div>
            </div>
            {!session.current && (
              <Button size="sm" variant="ghost" onClick={() => revokeSession(session.id)}>
                {t('account.sessions_revoke')}
              </Button>
            )}
          </div>
        ))}
      </div>
    </Dialog>
  )
}