import { useState } from 'react'
import { oauthStartUrl } from '../../hooks/useSecuritySettings'
import type { PublicAuthConfig } from '../../hooks/useSecuritySettings'
import { LOCAL_STORAGE_KEY } from '../../lib/constants'
import { OAUTH_PROVIDERS } from '../../lib/oauthProviders'
import { useTranslation } from '../../lib/i18n'
import { OAuthProviderIcon } from './OAuthProviderIcon'

const buttonClass =
  'flex w-full items-center justify-center gap-2.5 rounded-lg border border-white/10 bg-bg-tertiary px-4 py-2.5 text-sm font-medium text-text-primary hover:bg-bg-secondary disabled:opacity-50'

async function startBindFlow(provider: string) {
  const token = localStorage.getItem(LOCAL_STORAGE_KEY)
  if (!token) {
    window.location.href = '/login'
    return
  }
  const res = await fetch(`/api/v1/me/connections/${provider}/link`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error((body as { error?: { message?: string } })?.error?.message ?? 'Failed to start OAuth link')
  }
  const data = (await res.json()) as { redirect_url?: string }
  if (!data.redirect_url) {
    throw new Error('Failed to start OAuth link')
  }
  window.location.href = data.redirect_url
}

function isProviderVisible(
  config: PublicAuthConfig,
  id: (typeof OAUTH_PROVIDERS)[number]['id'],
  mode: 'login' | 'signup' | 'bind',
) {
  const o = config.oauth?.[id]
  if (!o?.enabled) return false
  if (mode === 'login') return o.login
  if (mode === 'signup') return o.signup
  return true
}

export function OAuthButtons({
  config,
  mode,
  acceptTerms,
  termsRequired,
}: {
  config?: PublicAuthConfig
  mode: 'login' | 'signup' | 'bind'
  acceptTerms?: boolean
  termsRequired?: boolean
}) {
  const { t } = useTranslation()
  const [linking, setLinking] = useState<string | null>(null)
  const [linkError, setLinkError] = useState<string | null>(null)

  if (!config) return null

  const visible = OAUTH_PROVIDERS.filter((p) => isProviderVisible(config, p.id, mode))
  if (visible.length === 0) return null

  const signupBlocked = mode === 'signup' && termsRequired && !acceptTerms

  return (
    <div className="space-y-2">
      {linkError && <p className="text-xs text-red-400">{linkError}</p>}
      {signupBlocked && (
        <p className="text-xs text-text-tertiary">{t('register.terms_required')}</p>
      )}
      {visible.map((p) => {
        const label = t('login.continue_with', { provider: t(p.labelKey) })
        const icon = <OAuthProviderIcon provider={p.id} size={18} />
        return mode === 'bind' ? (
          <button
            key={p.id}
            type="button"
            disabled={linking === p.id}
            onClick={() => {
              setLinkError(null)
              setLinking(p.id)
              void startBindFlow(p.id)
                .catch((e: unknown) => {
                  setLinkError(e instanceof Error ? e.message : t('login.oauth_failed'))
                })
                .finally(() => setLinking(null))
            }}
            className={buttonClass}
          >
            {icon}
            {label}
          </button>
        ) : (
          <a
            key={p.id}
            href={
              signupBlocked
                ? undefined
                : oauthStartUrl(p.id, mode, { acceptTerms: mode === 'signup' ? acceptTerms : undefined })
            }
            aria-disabled={signupBlocked}
            onClick={signupBlocked ? (e) => e.preventDefault() : undefined}
            className={`${buttonClass} no-underline ${signupBlocked ? 'pointer-events-none opacity-50' : ''}`}
          >
            {icon}
            {label}
          </a>
        )
      })}
    </div>
  )
}