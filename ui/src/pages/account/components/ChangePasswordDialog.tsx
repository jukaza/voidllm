import { useState } from 'react'
import { Button } from '../../../components/ui/Button'
import { Dialog } from '../../../components/ui/Dialog'
import { Input } from '../../../components/ui/Input'
import { useChangePassword } from '../../../hooks/useProfile'
import { useToast } from '../../../hooks/useToast'
import { useTranslation } from '../../../lib/i18n'

interface ChangePasswordDialogProps {
  open: boolean
  onClose: () => void
}

export function ChangePasswordDialog({ open, onClose }: ChangePasswordDialogProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const changePassword = useChangePassword()

  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  function reset() {
    setCurrentPassword('')
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

    if (!currentPassword) nextErrors.current = t('account.password_current_required')
    if (!newPassword || newPassword.length < 8) nextErrors.new = t('account.password_min_length')
    if (newPassword !== confirmPassword) nextErrors.confirm = t('account.password_mismatch')

    setErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) return

    changePassword.mutate(
      { current_password: currentPassword, new_password: newPassword },
      {
        onSuccess: () => {
          toast({ variant: 'success', message: t('account.password_changed') })
          handleClose()
        },
        onError: (err) => {
          const msg = err instanceof Error ? err.message : t('account.password_change_failed')
          if (msg.toLowerCase().includes('current')) {
            setErrors({ current: msg })
          } else {
            toast({ variant: 'error', message: msg })
          }
        },
      },
    )
  }

  return (
    <Dialog open={open} onClose={handleClose} title={t('account.change_password_title')}>
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label={t('account.current_password')}
          type="password"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          error={errors.current}
          autoComplete="current-password"
        />
        <Input
          label={t('account.new_password')}
          type="password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          error={errors.new}
          autoComplete="new-password"
          description={t('account.password_min_length')}
        />
        <Input
          label={t('account.confirm_password')}
          type="password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          error={errors.confirm}
          autoComplete="new-password"
        />
        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="ghost" onClick={handleClose}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" loading={changePassword.isPending}>
            {t('account.password_change')}
          </Button>
        </div>
      </form>
    </Dialog>
  )
}