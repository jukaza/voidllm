import { useState } from 'react'
import { Button } from '../../../components/ui/Button'
import { Dialog } from '../../../components/ui/Dialog'
import { Input } from '../../../components/ui/Input'
import { useSetPassword } from '../../../hooks/useAccountSecurity'
import { useToast } from '../../../hooks/useToast'
import { useTranslation } from '../../../lib/i18n'

interface SetPasswordDialogProps {
  open: boolean
  onClose: () => void
}

export function SetPasswordDialog({ open, onClose }: SetPasswordDialogProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const setPassword = useSetPassword()

  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  function reset() {
    setNewPassword('')
    setConfirmPassword('')
    setErrors({})
  }

  function handleClose() {
    reset()
    onClose()
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const nextErrors: Record<string, string> = {}
    if (!newPassword || newPassword.length < 8) nextErrors.new = t('account.password_min_length')
    if (newPassword !== confirmPassword) nextErrors.confirm = t('account.password_mismatch')
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) return

    setPassword.mutate(newPassword, {
      onSuccess: () => {
        toast({ variant: 'success', message: t('account.password_set_success') })
        handleClose()
      },
      onError: (err) => toast({ variant: 'error', message: err.message }),
    })
  }

  return (
    <Dialog open={open} onClose={handleClose} title={t('account.password_set_title')}>
      <form onSubmit={handleSubmit} className="space-y-4">
        <p className="text-sm text-text-tertiary">{t('account.password_set_desc')}</p>
        <Input
          label={t('account.new_password')}
          type="password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          error={errors.new}
        />
        <Input
          label={t('account.confirm_password')}
          type="password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          error={errors.confirm}
        />
        <div className="flex justify-end gap-2">
          <Button variant="ghost" type="button" onClick={handleClose}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" loading={setPassword.isPending}>
            {t('account.password_set_action')}
          </Button>
        </div>
      </form>
    </Dialog>
  )
}