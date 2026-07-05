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
import { BrandMark } from '../../components/brand/BrandMark'
import { OAuthButtons } from '../../components/auth/OAuthButtons'
import { usePublicAuthConfig } from '../../hooks/useSecuritySettings'
import { hasOAuthLogin } from '../../lib/oauthProviders'
import { TwoFactorLoginStep } from './TwoFactorLoginStep'

type LoginStep = 'credentials' | 'twofa'

export default function LoginPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { t } = useTranslation()
  const { data: authConfig } = usePublicAuthConfig()

  const [step, setStep] = useState<LoginStep>('credentials')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [tempToken, setTempToken] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const hasOAuth = hasOAuthLogin(authConfig)

  async function completeLogin(data: { token: string; user: MeResponse }) {
    localStorage.setItem(LOCAL_STORAGE_KEY, data.token)
    queryClient.setQueryData(['me'], data.user)
    navigate('/dashboard')
  }

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

      const body = (await res.json().catch(() => ({}))) as {
        error?: { message?: string }
        requires_2fa?: boolean
        temp_token?: string
        token?: string
        user?: MeResponse
      }

      if (!res.ok) {
        setError(body?.error?.message ?? 'Login failed')
        return
      }

      if (body.requires_2fa && body.temp_token) {
        setTempToken(body.temp_token)
        setStep('twofa')
        return
      }

      if (body.token && body.user) {
        await completeLogin({ token: body.token, user: body.user })
      }
    } catch {
      setError('Unable to reach the server. Check your connection.')
    } finally {
      setLoading(false)
    }
  }

  async function handle2FA(code: string) {
    setError(null)
    setLoading(true)
    try {
      const res = await fetch('/api/v1/auth/login/2fa', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ temp_token: tempToken, code }),
      })
      const body = (await res.json().catch(() => ({}))) as {
        error?: { message?: string }
        token?: string
        user?: MeResponse
      }
      if (!res.ok) {
        setError(body?.error?.message ?? t('login.twofa_invalid'))
        return
      }
      if (body.token && body.user) {
        await completeLogin({ token: body.token, user: body.user })
      }
    } catch {
      setError('Unable to reach the server. Check your connection.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-primary px-4">
      <div className="w-full max-w-sm bg-bg-secondary border border-white/5 rounded-xl p-8">
        <div className="mb-8 flex flex-col items-center gap-3">
          <BrandMark nameClassName="gradient-text text-3xl font-bold" />
          <p className="text-sm text-text-tertiary">
            {step === 'twofa' ? t('login.twofa_title') : t('login.subtitle')}
          </p>
        </div>

        {step === 'credentials' && hasOAuth && (
          <div className="mb-6 space-y-4">
            <OAuthButtons config={authConfig} mode="login" />
            <div className="flex items-center gap-3">
              <div className="h-px flex-1 bg-border" />
              <span className="text-xs text-text-tertiary">{t('login.or_email')}</span>
              <div className="h-px flex-1 bg-border" />
            </div>
          </div>
        )}

        {step === 'credentials' ? (
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
        ) : (
          <TwoFactorLoginStep
            loading={loading}
            error={error}
            onSubmit={(code) => void handle2FA(code)}
            onBack={() => {
              setStep('credentials')
              setTempToken('')
              setError(null)
            }}
          />
        )}

        {step === 'credentials' && (
          <p className="mt-6 text-center text-xs text-text-tertiary">
            {t('login.no_account')}{' '}
            <Link to="/register" className="text-accent no-underline">
              {t('login.create_account')}
            </Link>
          </p>
        )}
      </div>
    </div>
  )
}