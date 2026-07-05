import { useState } from 'react'
import { Button } from '../../../components/ui/Button'
import { Dialog } from '../../../components/ui/Dialog'
import { Input } from '../../../components/ui/Input'
import { useToast } from '../../../hooks/useToast'
import { useTranslation } from '../../../lib/i18n'
import { useAccountDraft } from '../useAccountDraft'

const DEMO_SECRET = 'JBSWY3DPEHPK3PXP'

interface TwoFactorDialogProps {
  open: boolean
  onClose: () => void
  email: string
}

export function TwoFactorDialog({ open, onClose, email }: TwoFactorDialogProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { draft, setDraft } = useAccountDraft()
  const [code, setCode] = useState('')

  function handleClose() {
    setCode('')
    onClose()
  }

  function disable2FA() {
    setDraft({ two_fa_enabled: false })
    toast({ variant: 'success', message: t('account.twofa_disabled') })
    handleClose()
  }

  function verifyCode() {
    if (code.length !== 6) {
      toast({ variant: 'error', message: t('account.twofa_code_invalid') })
      return
    }
    setDraft({ two_fa_enabled: true })
    toast({ variant: 'success', message: t('account.twofa_enabled') })
    handleClose()
  }

  if (draft.two_fa_enabled) {
    return (
      <Dialog open={open} onClose={handleClose} title={t('account.twofa_setup_title')}>
        <div className="space-y-4">
          <p className="text-sm text-text-tertiary">{t('account.twofa_manage_desc')}</p>
          <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={handleClose}>
              {t('common.close')}
            </Button>
            <Button variant="destructive" onClick={disable2FA}>
              {t('account.twofa_disable')}
            </Button>
          </div>
        </div>
      </Dialog>
    )
  }

  return (
    <Dialog open={open} onClose={handleClose} title={t('account.twofa_setup_title')}>
      <div className="space-y-4">
        <p className="text-sm text-text-tertiary">{t('account.twofa_setup_desc')}</p>

        <div className="rounded border border-border bg-bg-primary p-3 text-center font-mono text-sm">
          {DEMO_SECRET}
          <div className="mt-1 text-[10px] text-text-tertiary">
            otpauth://totp/VoidLLM:{email}
          </div>
        </div>

        <Input
          label={t('account.twofa_code')}
          value={code}
          onChange={(e) => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
          placeholder="123456"
          className="font-mono text-center text-lg tracking-[6px]"
        />
        <p className="text-xs text-text-tertiary">{t('account.twofa_secret_hint')}</p>

        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={handleClose}>
            {t('common.cancel')}
          </Button>
          <Button onClick={verifyCode}>{t('account.twofa_verify')}</Button>
        </div>
      </div>
    </Dialog>
  )
}