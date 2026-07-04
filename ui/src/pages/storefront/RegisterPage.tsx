import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Input } from '../../components/ui/Input'
import { Button } from '../../components/ui/Button'
import { Banner } from '../../components/ui/Banner'
import { LOCAL_STORAGE_KEY } from '../../lib/constants'
import type { MeResponse } from '../../hooks/useMe'
import { useTranslation } from '../../lib/i18n'

interface RegisterResponse {
  token: string
  expires_at: string
  user: MeResponse
}

export default function RegisterPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { t } = useTranslation()

  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      const res = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password, display_name: displayName }),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: { message: res.statusText } }))
        setError((body as { error?: { message?: string } })?.error?.message ?? 'Signup failed')
        return
      }

      const data = (await res.json()) as RegisterResponse
      localStorage.setItem(LOCAL_STORAGE_KEY, data.token)
      queryClient.setQueryData(['me'], data.user)
      navigate('/')
    } catch {
      setError('Unable to reach the server. Check your connection.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-primary px-4">
      <div className="w-full max-w-sm bg-bg-secondary border border-white/5 rounded-xl p-8">
        <div className="mb-8 text-center">
          <h1 className="text-3xl font-bold gradient-text">VoidLLM</h1>
          <p className="mt-2 text-sm text-text-tertiary">{t('register.subtitle')}</p>
        </div>

        <form onSubmit={(e) => void handleSubmit(e)} className="space-y-5">
          <Input
            label={t('register.display_name')}
            required
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="Nguyen Van A"
          />
          <Input
            label={t('login.email')}
            type="email"
            autoComplete="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
          />
          <Input
            label={t('login.password')}
            type="password"
            autoComplete="new-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            description={t('register.password_hint')}
          />

          {error !== null && <Banner variant="error" title={error} />}

          <Button type="submit" loading={loading} fullWidth size="lg">
            {t('register.submit')}
          </Button>
        </form>

        <p className="mt-6 text-center text-xs text-text-tertiary">
          {t('register.have_account')}{' '}
          <Link to="/login" className="text-accent no-underline">
            {t('storefront.sign_in')}
          </Link>
        </p>
      </div>
    </div>
  )
}