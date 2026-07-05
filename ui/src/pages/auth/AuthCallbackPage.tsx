import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Banner } from '../../components/ui/Banner'
import { LOCAL_STORAGE_KEY } from '../../lib/constants'
import type { MeResponse } from '../../hooks/useMe'
import { useTranslation } from '../../lib/i18n'

interface ExchangeResponse {
  token: string
  expires_at: string
  user: MeResponse
}

export default function AuthCallbackPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [params] = useSearchParams()
  const [error, setError] = useState<string | null>(null)
  const exchangeStarted = useRef(false)

  useEffect(() => {
    if (exchangeStarted.current) return
    exchangeStarted.current = true

    const err = params.get('error')
    if (err) {
      setError(err)
      return
    }
    const code = params.get('code')
    if (!code) {
      setError(t('login.oauth_missing_code'))
      return
    }
    void fetch('/api/v1/auth/oauth/exchange', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code }),
    })
      .then(async (res) => {
        if (!res.ok) {
          const body = await res.json().catch(() => ({}))
          throw new Error((body as { error?: { message?: string } })?.error?.message ?? 'exchange failed')
        }
        return res.json() as Promise<ExchangeResponse>
      })
      .then((data) => {
        localStorage.setItem(LOCAL_STORAGE_KEY, data.token)
        queryClient.setQueryData(['me'], data.user)
        navigate('/dashboard', { replace: true })
      })
      .catch((e: unknown) => {
        setError(e instanceof Error ? e.message : t('login.oauth_failed'))
      })
  }, [navigate, params, queryClient, t])

  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-primary px-4">
      <div className="w-full max-w-sm">
        {error ? (
          <Banner variant="error" title={error} />
        ) : (
          <p className="text-center text-sm text-text-tertiary">{t('login.oauth_completing')}</p>
        )}
      </div>
    </div>
  )
}