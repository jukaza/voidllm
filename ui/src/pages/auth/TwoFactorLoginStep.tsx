import { useState } from 'react'
import type { FormEvent } from 'react'
import { Input } from '../../components/ui/Input'
import { Button } from '../../components/ui/Button'
import { Banner } from '../../components/ui/Banner'
import { useTranslation } from '../../lib/i18n'

interface TwoFactorLoginStepProps {
  loading: boolean
  error: string | null
  onSubmit: (code: string) => void
  onBack: () => void
}

export function TwoFactorLoginStep({ loading, error, onSubmit, onBack }: TwoFactorLoginStepProps) {
  const { t } = useTranslation()
  const [code, setCode] = useState('')

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    onSubmit(code.trim())
  }

  return (
    <form onSubmit={(e) => void handleSubmit(e)} className="space-y-5">
      <p className="text-sm text-text-tertiary">{t('login.twofa_desc')}</p>
      <Input
        label={t('login.twofa_code')}
        value={code}
        onChange={(e) => setCode(e.target.value.replace(/\s/g, '').slice(0, 9))}
        placeholder="123456"
        className="font-mono text-center text-lg tracking-[4px]"
        autoComplete="one-time-code"
      />
      {error !== null && <Banner variant="error" title={error} />}
      <Button type="submit" loading={loading} fullWidth size="lg">
        {t('login.twofa_verify')}
      </Button>
      <Button type="button" variant="ghost" fullWidth onClick={onBack}>
        {t('login.twofa_back')}
      </Button>
    </form>
  )
}