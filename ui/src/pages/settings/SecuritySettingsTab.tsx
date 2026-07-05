import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Banner } from '../../components/ui/Banner'
import { Input } from '../../components/ui/Input'
import { Toggle } from '../../components/ui/Toggle'
import { useAdminSiteSettings, useUpdateSiteConfig } from '../../hooks/useSiteConfig'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { PreviewBadge } from './components/PreviewBadge'
import { SettingsSectionCard } from './components/SettingsSectionCard'
import { SettingsTabFooter } from './components/SettingsTabFooter'
import { useSettingsDraft } from './useSettingsDraft'

function OAuthCard({
  title,
  enabled,
  clientId,
  clientSecret,
  onEnabledChange,
  onClientIdChange,
  onClientSecretChange,
}: {
  title: string
  enabled: boolean
  clientId: string
  clientSecret: string
  onEnabledChange: (v: boolean) => void
  onClientIdChange: (v: string) => void
  onClientSecretChange: (v: string) => void
}) {
  const { t } = useTranslation()
  return (
    <div className="rounded-md border border-border p-4 space-y-4">
      <Toggle checked={enabled} onChange={onEnabledChange} label={title} />
      {enabled && (
        <>
          <Input
            label={t('settings.oauth_client_id')}
            value={clientId}
            onChange={(e) => onClientIdChange(e.target.value)}
            className="font-mono text-sm"
          />
          <Input
            label={t('settings.oauth_client_secret')}
            type="password"
            value={clientSecret}
            onChange={(e) => onClientSecretChange(e.target.value)}
            placeholder="••••••••"
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
  const { data } = useAdminSiteSettings()
  const updateSite = useUpdateSiteConfig()
  const { draft, setDraft } = useSettingsDraft()

  const [registerEnabled, setRegisterEnabled] = useState(true)

  useEffect(() => {
    if (data) setRegisterEnabled(data.register_enabled)
  }, [data])

  return (
    <div className="space-y-6">
      <SettingsSectionCard
        title={t('settings.security_registration_title')}
        description={t('settings.security_registration_desc')}
        badge={<PreviewBadge />}
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
        badge={<PreviewBadge />}
      >
        <Toggle
          checked={draft.turnstile_enabled}
          onChange={(v) => setDraft({ turnstile_enabled: v })}
          label={t('settings.turnstile_enabled')}
        />
        {draft.turnstile_enabled && (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Input
              label={t('settings.turnstile_site_key')}
              value={draft.turnstile_site_key}
              onChange={(e) => setDraft({ turnstile_site_key: e.target.value })}
            />
            <Input
              label={t('settings.turnstile_secret_key')}
              type="password"
              value={draft.turnstile_secret_key}
              onChange={(e) => setDraft({ turnstile_secret_key: e.target.value })}
            />
          </div>
        )}
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.oauth_title')}
        description={t('settings.oauth_desc')}
        badge={<PreviewBadge />}
      >
        <Banner variant="info" title={t('settings.oauth_account_hint')} />
        <OAuthCard
          title="Google"
          enabled={draft.oauth_google.enabled}
          clientId={draft.oauth_google.client_id}
          clientSecret={draft.oauth_google.client_secret}
          onEnabledChange={(v) => setDraft({ oauth_google: { ...draft.oauth_google, enabled: v } })}
          onClientIdChange={(v) => setDraft({ oauth_google: { ...draft.oauth_google, client_id: v } })}
          onClientSecretChange={(v) =>
            setDraft({ oauth_google: { ...draft.oauth_google, client_secret: v } })
          }
        />
        <OAuthCard
          title="GitHub"
          enabled={draft.oauth_github.enabled}
          clientId={draft.oauth_github.client_id}
          clientSecret={draft.oauth_github.client_secret}
          onEnabledChange={(v) => setDraft({ oauth_github: { ...draft.oauth_github, enabled: v } })}
          onClientIdChange={(v) => setDraft({ oauth_github: { ...draft.oauth_github, client_id: v } })}
          onClientSecretChange={(v) =>
            setDraft({ oauth_github: { ...draft.oauth_github, client_secret: v } })}
        />
        <div className="rounded-md border border-border p-4 space-y-4">
          <Toggle
            checked={draft.oauth_oidc.enabled}
            onChange={(v) => setDraft({ oauth_oidc: { ...draft.oauth_oidc, enabled: v } })}
            label="OIDC / SSO"
          />
          {draft.oauth_oidc.enabled && (
            <>
              <Input
                label={t('settings.oauth_issuer')}
                value={draft.oauth_oidc.issuer_url}
                onChange={(e) =>
                  setDraft({ oauth_oidc: { ...draft.oauth_oidc, issuer_url: e.target.value } })
                }
              />
              <Input
                label={t('settings.oauth_client_id')}
                value={draft.oauth_oidc.client_id}
                onChange={(e) =>
                  setDraft({ oauth_oidc: { ...draft.oauth_oidc, client_id: e.target.value } })
                }
              />
              <Input
                label={t('settings.oauth_client_secret')}
                type="password"
                value={draft.oauth_oidc.client_secret}
                onChange={(e) =>
                  setDraft({ oauth_oidc: { ...draft.oauth_oidc, client_secret: e.target.value } })
                }
              />
            </>
          )}
        </div>
        <p className="text-xs text-text-tertiary">
          {t('settings.oauth_user_link')}{' '}
          <Link to="/account" className="text-accent hover:underline">
            /account
          </Link>
        </p>
      </SettingsSectionCard>

      <SettingsTabFooter
        mode="preview"
        loading={updateSite.isPending}
        onSave={() => {
          updateSite.mutate(
            { register_enabled: registerEnabled },
            {
              onSuccess: () =>
                toast({ variant: 'success', message: t('settings.saved_security') }),
              onError: (err) => toast({ variant: 'error', message: err.message }),
            },
          )
        }}
      />
    </div>
  )
}