import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Input } from '../../components/ui/Input'
import { Button } from '../../components/ui/Button'
import { Banner } from '../../components/ui/Banner'
import { BrandMark } from '../../components/brand/BrandMark'
import { LOCAL_STORAGE_KEY } from '../../lib/constants'
import type { MeResponse } from '../../hooks/useMe'
import { useSiteConfig } from '../../hooks/useSiteConfig'
import { useTranslation } from '../../lib/i18n'
import { OAuthButtons } from '../../components/auth/OAuthButtons'
import { TurnstileWidget } from '../../components/auth/TurnstileWidget'
import { usePublicAuthConfig } from '../../hooks/useSecuritySettings'
import { hasOAuthSignup } from '../../lib/oauthProviders'

interface RegisterResponse {
  token: string
  expires_at: string
  user: MeResponse
}

export default function RegisterPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { t } = useTranslation()
  const { data: site, isLoading: siteLoading } = useSiteConfig()
  const { data: authConfig } = usePublicAuthConfig()

  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [acceptTerms, setAcceptTerms] = useState(false)
  const [turnstileToken, setTurnstileToken] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  if (!siteLoading && site && !site.register_enabled) {
    return <Navigate to="/login" replace />
  }

  const termsRequired = site?.user_agreement_enabled ?? false
  const turnstileEnabled = Boolean(authConfig?.turnstile.enabled && authConfig.turnstile.site_key)
  const hasOAuth = hasOAuthSignup(authConfig)

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setError(null)

    if (termsRequired && !acceptTerms) {
      setError(t('register.terms_required'))
      return
    }

    if (turnstileEnabled && !turnstileToken) {
      setError(t('register.turnstile_required'))
      return
    }

    setLoading(true)

    try {
      const body: Record<string, string | boolean> = {
        email,
        password,
        display_name: displayName,
        accept_terms: acceptTerms,
      }
      if (turnstileEnabled) {
        body.turnstile_token = turnstileToken
      }

      const res = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })

      if (!res.ok) {
        const errBody = await res.json().catch(() => ({ error: { message: res.statusText } }))
        setError((errBody as { error?: { message?: string } })?.error?.message ?? 'Signup failed')
        return
      }

      const data = (await res.json()) as RegisterResponse
      localStorage.setItem(LOCAL_STORAGE_KEY, data.token)
      queryClient.setQueryData(['me'], data.user)
      navigate('/dashboard')
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
          <p className="text-sm text-text-tertiary">{t('register.subtitle')}</p>
        </div>

        {hasOAuth && (
          <div className="mb-6 space-y-4">
            <OAuthButtons
              config={authConfig}
              mode="signup"
              acceptTerms={acceptTerms}
              termsRequired={termsRequired}
            />
            <div className="flex items-center gap-3">
              <div className="h-px flex-1 bg-border" />
              <span className="text-xs text-text-tertiary">{t('login.or_email')}</span>
              <div className="h-px flex-1 bg-border" />
            </div>
          </div>
        )}

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

          {termsRequired && (
            <label className="flex items-start gap-2 text-xs text-text-secondary cursor-pointer">
              <input
                type="checkbox"
                checked={acceptTerms}
                onChange={(e) => setAcceptTerms(e.target.checked)}
                className="mt-0.5 accent-accent"
              />
              <span>
                {t('register.accept_terms')}{' '}
                <Link to="/legal/terms" target="_blank" className="text-accent no-underline">
                  {t('register.terms_link')}
                </Link>
                {site?.privacy_policy_enabled && (
                  <>
                    {' '}{t('register.and')}{' '}
                    <Link to="/legal/privacy" target="_blank" className="text-accent no-underline">
                      {t('register.privacy_link')}
                    </Link>
                  </>
                )}
              </span>
            </label>
          )}

          {turnstileEnabled && authConfig?.turnstile.site_key && (
            <TurnstileWidget siteKey={authConfig.turnstile.site_key} onToken={setTurnstileToken} />
          )}

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