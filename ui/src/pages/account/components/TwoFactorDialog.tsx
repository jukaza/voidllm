import { useEffect, useState } from 'react'
import { Button } from '../../../components/ui/Button'
import { Dialog } from '../../../components/ui/Dialog'
import { Input } from '../../../components/ui/Input'
import { useTwoFADisable, useTwoFASetup, useTwoFAVerify } from '../../../hooks/useAccountSecurity'
import { useMe } from '../../../hooks/useMe'
import { useToast } from '../../../hooks/useToast'
import { useTranslation } from '../../../lib/i18n'
import { TwoFactorQR } from './TwoFactorQR'

type Step = 'manage' | 'setup' | 'verify' | 'backup'

interface TwoFactorDialogProps {
  open: boolean
  onClose: () => void
  email: string
}

export function TwoFactorDialog({ open, onClose, email }: TwoFactorDialogProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data: me, refetch } = useMe()
  const setup = useTwoFASetup()
  const verify = useTwoFAVerify()
  const disable = useTwoFADisable()

  const [step, setStep] = useState<Step>('manage')
  const [secret, setSecret] = useState('')
  const [otpauthUrl, setOtpauthUrl] = useState('')
  const [code, setCode] = useState('')
  const [backupCodes, setBackupCodes] = useState<string[]>([])
  const [savedAck, setSavedAck] = useState(false)
  const [disablePassword, setDisablePassword] = useState('')
  const [disableCode, setDisableCode] = useState('')

  useEffect(() => {
    if (!open) return
    setStep(me?.two_fa_enabled ? 'manage' : 'setup')
    setSecret('')
    setOtpauthUrl('')
    setCode('')
    setBackupCodes([])
    setSavedAck(false)
    setDisablePassword('')
    setDisableCode('')
  }, [open, me?.two_fa_enabled])

  function handleClose() {
    onClose()
  }

  function startSetup() {
    setup.mutate(undefined, {
      onSuccess: (data) => {
        setSecret(data.secret)
        setOtpauthUrl(data.otpauth_url)
        setStep('verify')
      },
      onError: (err) => toast({ variant: 'error', message: err.message }),
    })
  }

  function confirmVerify() {
    if (code.length < 6) {
      toast({ variant: 'error', message: t('account.twofa_code_invalid') })
      return
    }
    verify.mutate(code, {
      onSuccess: (data) => {
        setBackupCodes(data.backup_codes)
        setStep('backup')
        void refetch()
      },
      onError: (err) => toast({ variant: 'error', message: err.message }),
    })
  }

  function finishBackup() {
    if (!savedAck) {
      toast({ variant: 'error', message: t('account.twofa_backup_ack_required') })
      return
    }
    toast({ variant: 'success', message: t('account.twofa_enabled') })
    handleClose()
  }

  async function copySecret() {
    if (!secret) return
    try {
      await navigator.clipboard.writeText(secret)
      toast({ variant: 'success', message: t('account.twofa_secret_copied') })
    } catch {
      toast({ variant: 'error', message: t('account.twofa_copy_failed') })
    }
  }

  function disable2FA() {
    disable.mutate(
      {
        password: disablePassword || undefined,
        code: disableCode || undefined,
      },
      {
        onSuccess: () => {
          toast({ variant: 'success', message: t('account.twofa_disabled') })
          void refetch()
          handleClose()
        },
        onError: (err) => toast({ variant: 'error', message: err.message }),
      },
    )
  }

  if (me?.two_fa_enabled && step === 'manage') {
    return (
      <Dialog open={open} onClose={handleClose} title={t('account.twofa_setup_title')}>
        <div className="space-y-4">
          <p className="text-sm text-text-tertiary">{t('account.twofa_manage_desc')}</p>
          <Input
            label={t('account.current_password')}
            type="password"
            value={disablePassword}
            onChange={(e) => setDisablePassword(e.target.value)}
          />
          <Input
            label={t('account.twofa_code')}
            value={disableCode}
            onChange={(e) => setDisableCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
            placeholder="123456"
          />
          <p className="text-xs text-text-tertiary">{t('account.twofa_disable_hint')}</p>
          <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={handleClose}>
              {t('common.close')}
            </Button>
            <Button variant="destructive" loading={disable.isPending} onClick={disable2FA}>
              {t('account.twofa_disable')}
            </Button>
          </div>
        </div>
      </Dialog>
    )
  }

  if (step === 'backup') {
    return (
      <Dialog open={open} onClose={handleClose} title={t('account.twofa_backup_title')}>
        <div className="space-y-4">
          <p className="text-sm text-text-tertiary">{t('account.twofa_backup_desc')}</p>
          <div className="grid grid-cols-2 gap-2 rounded border border-border bg-bg-primary p-3 font-mono text-sm">
            {backupCodes.map((c) => (
              <span key={c}>{c}</span>
            ))}
          </div>
          <label className="flex items-center gap-2 text-sm text-text-secondary">
            <input type="checkbox" checked={savedAck} onChange={(e) => setSavedAck(e.target.checked)} />
            {t('account.twofa_backup_saved')}
          </label>
          <div className="flex justify-end">
            <Button onClick={finishBackup}>{t('common.done')}</Button>
          </div>
        </div>
      </Dialog>
    )
  }

  if (step === 'verify') {
    const scanUrl = otpauthUrl || (secret ? `otpauth://totp/VoidLLM:${encodeURIComponent(email)}?secret=${secret}&issuer=VoidLLM` : '')
    return (
      <Dialog open={open} onClose={handleClose} title={t('account.twofa_setup_title')}>
        <div className="space-y-4">
          <p className="text-sm text-text-tertiary">{t('account.twofa_setup_desc')}</p>
          <div className="flex flex-col items-center gap-3 rounded-md border border-border bg-bg-primary p-4">
            <TwoFactorQR otpauthUrl={scanUrl} />
            <p className="text-xs text-text-tertiary">{t('account.twofa_scan_qr')}</p>
          </div>
          <div className="space-y-2">
            <p className="text-xs text-text-tertiary">{t('account.twofa_manual_secret')}</p>
            <div className="flex items-center gap-2 rounded border border-border bg-bg-primary p-3">
              <code className="min-w-0 flex-1 break-all font-mono text-sm">{secret}</code>
              <Button size="sm" variant="ghost" type="button" onClick={() => void copySecret()}>
                {t('common.copy')}
              </Button>
            </div>
          </div>
          <Input
            label={t('account.twofa_code')}
            value={code}
            onChange={(e) => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
            placeholder="123456"
            className="font-mono text-center text-lg tracking-[6px]"
          />
          <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={handleClose}>
              {t('common.cancel')}
            </Button>
            <Button loading={verify.isPending} onClick={confirmVerify}>
              {t('account.twofa_verify')}
            </Button>
          </div>
        </div>
      </Dialog>
    )
  }

  return (
    <Dialog open={open} onClose={handleClose} title={t('account.twofa_setup_title')}>
      <div className="space-y-4">
        <p className="text-sm text-text-tertiary">{t('account.twofa_intro')}</p>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={handleClose}>
            {t('common.cancel')}
          </Button>
          <Button loading={setup.isPending} onClick={startSetup}>
            {t('account.twofa_enable')}
          </Button>
        </div>
      </div>
    </Dialog>
  )
}