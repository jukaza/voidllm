import { CopyButton } from '../../../components/ui/CopyButton'
import { useTranslation } from '../../../lib/i18n'
import type { OAuthProviderId } from '../../../lib/oauthProviders'

const SETUP_LINKS = {
  turnstile: 'https://dash.cloudflare.com/?to=/:account/turnstile',
  google: 'https://console.cloud.google.com/apis/credentials',
  github: 'https://github.com/settings/developers',
} as const

export function oauthRedirectUri(base: string, provider: OAuthProviderId): string {
  const root = base.replace(/\/$/, '')
  return `${root}/api/v1/auth/oauth/${provider}/callback`
}

export function resolveApiBaseUrl(serverAddress?: string): string {
  const trimmed = serverAddress?.trim()
  if (trimmed) return trimmed.replace(/\/$/, '')

  if (typeof window === 'undefined') return ''

  const { protocol, hostname } = window.location
  if (hostname === 'localhost' || hostname === '127.0.0.1') {
    return `${protocol}//${hostname}:8080`
  }
  return window.location.origin.replace(/\/$/, '')
}

function RedirectUriField({ uri }: { uri: string }) {
  const { t } = useTranslation()
  return (
    <div className="space-y-1">
      <div className="text-xs font-medium text-text-secondary">{t('settings.oauth_redirect_uri')}</div>
      <div className="flex flex-wrap items-center gap-2 rounded-md border border-border bg-bg-primary px-2 py-1.5">
        <code className="min-w-0 flex-1 break-all font-mono text-[11px] text-text-tertiary">{uri}</code>
        <CopyButton text={uri} label={t('common.copy')} copiedLabel={t('common.copied')} />
      </div>
      <p className="text-[11px] text-text-tertiary">{t('settings.oauth_redirect_uri_hint')}</p>
    </div>
  )
}

export function SetupGuideBlock({
  linkLabel,
  linkHref,
  hint,
  redirectUri,
}: {
  linkLabel: string
  linkHref: string
  hint: string
  redirectUri?: string
}) {
  return (
    <div className="rounded-md border border-border/70 bg-bg-tertiary/30 px-3 py-2.5 space-y-2">
      <a
        href={linkHref}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-1 text-xs font-medium text-accent no-underline hover:opacity-80"
      >
        {linkLabel}
        <span aria-hidden>↗</span>
      </a>
      <p className="text-xs text-text-tertiary whitespace-pre-line leading-relaxed">{hint}</p>
      {redirectUri ? <RedirectUriField uri={redirectUri} /> : null}
    </div>
  )
}

export function TurnstileSetupGuide() {
  const { t } = useTranslation()
  return (
    <SetupGuideBlock
      linkHref={SETUP_LINKS.turnstile}
      linkLabel={t('settings.turnstile_setup_link')}
      hint={t('settings.turnstile_setup_hint')}
    />
  )
}

export function OAuthSetupGuide({
  provider,
  redirectUri,
}: {
  provider: OAuthProviderId
  redirectUri: string
}) {
  const { t } = useTranslation()
  const redirect = redirectUri
  const linkLabel =
    provider === 'google' ? t('settings.oauth_google_setup_link') : t('settings.oauth_github_setup_link')
  const hint =
    provider === 'google' ? t('settings.oauth_google_setup_hint') : t('settings.oauth_github_setup_hint')

  return (
    <SetupGuideBlock
      linkHref={SETUP_LINKS[provider]}
      linkLabel={linkLabel}
      hint={hint}
      redirectUri={redirect}
    />
  )
}