import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Input } from '../../components/ui/Input'
import { Button } from '../../components/ui/Button'
import { Banner } from '../../components/ui/Banner'
import { LOCAL_STORAGE_KEY } from '../../lib/constants'
import type { MeResponse } from '../../hooks/useMe'
import { useTranslation } from '../../lib/i18n'

export default function LoginPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { t } = useTranslation()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: { message: res.statusText } }))
        setError((body as { error?: { message?: string } })?.error?.message ?? 'Login failed')
        return
      }

      const data = (await res.json()) as { token: string; expires_at: string; user: MeResponse }
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
          <p className="mt-2 text-sm text-text-tertiary">{t('login.subtitle')}</p>
        </div>

        <form onSubmit={(e) => void handleSubmit(e)} className="space-y-5">
          <Input
            label={t('login.email')}
            type="text"
            autoComplete="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
          />

          <Input
            label={t('login.password')}
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
          />

          {error !== null && <Banner variant="error" title={error} />}

          <Button type="submit" loading={loading} fullWidth size="lg">
            {t('login.sign_in')}
          </Button>
        </form>
      </div>
    </div>
  )
}
