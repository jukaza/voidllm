import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Banner } from '../../components/ui/Banner'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Toggle } from '../../components/ui/Toggle'
import { OAuthProviderIcon } from '../../components/auth/OAuthProviderIcon'
import { useAdminSiteSettings, useUpdateSiteConfig } from '../../hooks/useSiteConfig'
import {
  useSecuritySettings,
  useUpdateSecuritySettings,
  type SecurityConfigUpdate,
} from '../../hooks/useSecuritySettings'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import type { OAuthProviderId } from '../../lib/oauthProviders'
import {
  OAuthSetupGuide,
  TurnstileSetupGuide,
  oauthRedirectUri,
  resolveApiBaseUrl,
} from './components/SecuritySetupGuide'
import { LiveBadge } from './components/PreviewBadge'
import { SettingsSectionCard } from './components/SettingsSectionCard'
import { SettingsTabFooter } from './components/SettingsTabFooter'

type OAuthDraft = {
  enabled: boolean
  allow_login: boolean
  allow_signup: boolean
  client_id: string
  client_secret: string
}

const emptyOAuth = (): OAuthDraft => ({
  enabled: false,
  allow_login: true,
  allow_signup: true,
  client_id: '',
  client_secret: '',
})

function OAuthCard({
  provider,
  title,
  draft,
  secretConfigured,
  redirectUri,
  onChange,
}: {
  provider: OAuthProviderId
  title: string
  draft: OAuthDraft
  secretConfigured: boolean
  redirectUri: string
  onChange: (patch: Partial<OAuthDraft>) => void
}) {
  const { t } = useTranslation()
  return (
    <div className="rounded-md border border-border p-4 space-y-4">
      <div className="flex items-center gap-2">
        <OAuthProviderIcon provider={provider} size={20} />
        <Toggle checked={draft.enabled} onChange={(v) => onChange({ enabled: v })} label={title} />
      </div>
      {draft.enabled && (
        <>
          <OAuthSetupGuide provider={provider} redirectUri={redirectUri} />
          <div className="flex flex-wrap gap-4">
            <Toggle
              checked={draft.allow_login}
              onChange={(v) => onChange({ allow_login: v })}
              label={t('settings.oauth_allow_login')}
            />
            <Toggle
              checked={draft.allow_signup}
              onChange={(v) => onChange({ allow_signup: v })}
              label={t('settings.oauth_allow_signup')}
            />
          </div>
          <Input
            label={t('settings.oauth_client_id')}
            value={draft.client_id}
            onChange={(e) => onChange({ client_id: e.target.value })}
            className="font-mono text-sm"
          />
          <Input
            label={t('settings.oauth_client_secret')}
            type="password"
            value={draft.client_secret}
            onChange={(e) => onChange({ client_secret: e.target.value })}
            placeholder={secretConfigured ? '••••••••' : ''}
            className="font-mono text-sm"
          />
        </>
      )}
    </div>
  )
}

export function SecuritySettingsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const queryClient = useQueryClient()
  const { data: site } = useAdminSiteSettings()
  const updateSite = useUpdateSiteConfig()
  const {
    data: security,
    isLoading,
    isError,
    error,
    refetch,
  } = useSecuritySettings()
  const updateSecurity = useUpdateSecuritySettings()

  const [registerEnabled, setRegisterEnabled] = useState(true)
  const [turnstileEnabled, setTurnstileEnabled] = useState(false)
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('')
  const [turnstileSecretKey, setTurnstileSecretKey] = useState('')
  const [google, setGoogle] = useState<OAuthDraft>(emptyOAuth())
  const [github, setGithub] = useState<OAuthDraft>(emptyOAuth())
  const [twoFAAllowUser, setTwoFAAllowUser] = useState(false)
  const [sessionTTL, setSessionTTL] = useState(24)
  const [sessionAllowMultiple, setSessionAllowMultiple] = useState(false)
  const [sessionMaxConcurrent, setSessionMaxConcurrent] = useState(5)
  const [passwordMinLength, setPasswordMinLength] = useState(8)
  const [passwordAllowOAuthSet, setPasswordAllowOAuthSet] = useState(true)

  useEffect(() => {
    if (site) setRegisterEnabled(site.register_enabled)
  }, [site])

  useEffect(() => {
    const googleCfg = security?.oauth?.google
    const githubCfg = security?.oauth?.github
    if (!googleCfg || !githubCfg) return

    setTurnstileEnabled(security.turnstile?.enabled ?? false)
    setTurnstileSiteKey(security.turnstile?.site_key ?? '')
    setTurnstileSecretKey('')
    setGoogle({
      enabled: googleCfg.enabled,
      allow_login: googleCfg.allow_login,
      allow_signup: googleCfg.allow_signup,
      client_id: googleCfg.client_id ?? '',
      client_secret: '',
    })
    setGithub({
      enabled: githubCfg.enabled,
      allow_login: githubCfg.allow_login,
      allow_signup: githubCfg.allow_signup,
      client_id: githubCfg.client_id ?? '',
      client_secret: '',
    })
    setTwoFAAllowUser(security.two_fa?.allow_user_enable ?? false)
    setSessionTTL(security.session?.ttl_hours ?? 24)
    setSessionAllowMultiple(security.session?.allow_multiple ?? false)
    setSessionMaxConcurrent(security.session?.max_concurrent ?? 5)
    setPasswordMinLength(security.password?.min_length ?? 8)
    setPasswordAllowOAuthSet(security.password?.allow_oauth_set_password ?? true)
  }, [security])

  function oauthPayload(draft: OAuthDraft) {
    const payload: Record<string, string | boolean> = {
      enabled: draft.enabled,
      allow_login: draft.allow_login,
      allow_signup: draft.allow_signup,
      client_id: draft.client_id.trim(),
    }
    if (draft.client_secret.trim()) {
      payload.client_secret = draft.client_secret.trim()
    }
    return payload
  }

  function save() {
    const securityPayload: SecurityConfigUpdate = {
      turnstile: {
        enabled: turnstileEnabled,
        site_key: turnstileSiteKey.trim(),
      },
      oauth: {
        google: oauthPayload(google),
        github: oauthPayload(github),
      },
      two_fa: { allow_user_enable: twoFAAllowUser },
      session: {
        ttl_hours: sessionTTL,
        allow_multiple: sessionAllowMultiple,
        max_concurrent: sessionMaxConcurrent,
      },
      password: {
        min_length: passwordMinLength,
        allow_oauth_set_password: passwordAllowOAuthSet,
      },
    }
    if (turnstileSecretKey.trim()) {
      securityPayload.turnstile!.secret_key = turnstileSecretKey.trim()
    }

    updateSite.mutate(
      { register_enabled: registerEnabled },
      {
        onSuccess: () => {
          updateSecurity.mutate(securityPayload, {
            onSuccess: () => {
              void queryClient.invalidateQueries({ queryKey: ['me'] })
              void queryClient.invalidateQueries({ queryKey: ['public-auth-config'] })
              toast({ variant: 'success', message: t('settings.saved_security') })
            },
            onError: (err) => toast({ variant: 'error', message: err.message }),
          })
        },
        onError: (err) => toast({ variant: 'error', message: err.message }),
      },
    )
  }

  if (isLoading) {
    return <p className="text-sm text-text-tertiary">{t('common.loading')}</p>
  }

  const apiBase = resolveApiBaseUrl(site?.server_address)
  const googleRedirect =
    security?.oauth_callback_urls?.google ?? oauthRedirectUri(apiBase, 'google')
  const githubRedirect =
    security?.oauth_callback_urls?.github ?? oauthRedirectUri(apiBase, 'github')

  if (isError) {
    return (
      <div className="space-y-4">
        <Banner
          variant="error"
          title={t('settings.security_load_error')}
          description={error?.message ?? t('settings.security_load_error_hint')}
        />
        <Button variant="secondary" onClick={() => void refetch()}>
          {t('common.refresh')}
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <SettingsSectionCard
        title={t('settings.security_registration_title')}
        description={t('settings.security_registration_desc')}
        badge={<LiveBadge />}
      >
        <Toggle
          checked={registerEnabled}
          onChange={setRegisterEnabled}
          label={t('settings.register_enabled')}
        />
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.turnstile_title')}
        description={t('settings.turnstile_desc')}
        badge={<LiveBadge />}
      >
        <Toggle
          checked={turnstileEnabled}
          onChange={setTurnstileEnabled}
          label={t('settings.turnstile_enabled')}
        />
        {turnstileEnabled && (
          <div className="space-y-4">
            <TurnstileSetupGuide />
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Input
                label={t('settings.turnstile_site_key')}
                value={turnstileSiteKey}
                onChange={(e) => setTurnstileSiteKey(e.target.value)}
                description={t('settings.turnstile_site_key_hint')}
              />
              <Input
                label={t('settings.turnstile_secret_key')}
                type="password"
                value={turnstileSecretKey}
                onChange={(e) => setTurnstileSecretKey(e.target.value)}
                placeholder={security?.turnstile?.secret_key_configured ? '••••••••' : ''}
                description={t('settings.turnstile_secret_key_hint')}
              />
            </div>
          </div>
        )}
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.oauth_title')}
        description={t('settings.oauth_desc')}
        badge={<LiveBadge />}
      >
        <Banner variant="info" title={t('settings.oauth_account_hint')} />
        {!site?.server_address?.trim() && (
          <div className="space-y-1">
            <Banner
              variant="warning"
              title={t('settings.oauth_api_base_missing')}
              description={t('settings.oauth_api_base_missing_hint')}
            />
            <Link to="/settings?tab=general" className="text-xs text-accent no-underline hover:opacity-80">
              {t('settings.tab_general')} →
            </Link>
          </div>
        )}
        <OAuthCard
          provider="google"
          title="Google"
          draft={google}
          redirectUri={googleRedirect}
          secretConfigured={security?.oauth?.google?.client_secret_configured ?? false}
          onChange={(patch) => setGoogle((prev) => ({ ...prev, ...patch }))}
        />
        <OAuthCard
          provider="github"
          title="GitHub"
          draft={github}
          redirectUri={githubRedirect}
          secretConfigured={security?.oauth?.github?.client_secret_configured ?? false}
          onChange={(patch) => setGithub((prev) => ({ ...prev, ...patch }))}
        />
        <p className="text-xs text-text-tertiary">
          {t('settings.oauth_user_link')}{' '}
          <Link to="/account" className="text-accent hover:underline">
            /account
          </Link>
        </p>
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.policy_2fa_title')}
        description={t('settings.policy_2fa_desc')}
        badge={<LiveBadge />}
      >
        <Toggle
          checked={twoFAAllowUser}
          onChange={setTwoFAAllowUser}
          label={t('settings.policy_2fa_allow')}
        />
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.policy_session_title')}
        description={t('settings.policy_session_desc')}
        badge={<LiveBadge />}
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <label className="block text-sm">
            <span className="mb-1 block text-text-secondary">{t('settings.policy_session_ttl')}</span>
            <select
              className="w-full rounded-md border border-border bg-bg-primary px-3 py-2 text-sm"
              value={sessionTTL}
              onChange={(e) => setSessionTTL(Number(e.target.value))}
            >
              <option value={8}>8h</option>
              <option value={24}>24h</option>
              <option value={168}>7d</option>
              <option value={720}>30d</option>
            </select>
          </label>
          <Input
            label={t('settings.policy_session_max')}
            type="number"
            min={1}
            max={50}
            value={String(sessionMaxConcurrent)}
            onChange={(e) => setSessionMaxConcurrent(Number(e.target.value) || 5)}
            disabled={!sessionAllowMultiple}
          />
        </div>
        <Toggle
          checked={sessionAllowMultiple}
          onChange={setSessionAllowMultiple}
          label={t('settings.policy_session_multiple')}
        />
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.policy_password_title')}
        description={t('settings.policy_password_desc')}
        badge={<LiveBadge />}
      >
        <div className="space-y-4">
          <Input
            label={t('settings.policy_password_min')}
            type="number"
            min={8}
            max={72}
            value={String(passwordMinLength)}
            onChange={(e) => setPasswordMinLength(Number(e.target.value) || 8)}
          />
          <Toggle
            checked={passwordAllowOAuthSet}
            onChange={setPasswordAllowOAuthSet}
            label={t('settings.policy_password_oauth_set')}
          />
        </div>
      </SettingsSectionCard>

      <SettingsTabFooter
        mode="live"
        loading={updateSite.isPending || updateSecurity.isPending}
        onSave={save}
      />
    </div>
  )
}