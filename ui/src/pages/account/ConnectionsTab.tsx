import { Button } from '../../components/ui/Button'
import { Badge } from '../../components/ui/Badge'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { OAuthProviderIcon } from '../../components/auth/OAuthProviderIcon'
import { OAUTH_PROVIDERS } from '../../lib/oauthProviders'
import { LiveBadge } from '../settings/components/PreviewBadge'
import { SettingsSectionCard } from '../settings/components/SettingsSectionCard'
import { OAuthButtons } from '../../components/auth/OAuthButtons'
import { usePublicAuthConfig, useMeConnections, useDeleteConnection } from '../../hooks/useSecuritySettings'

export function ConnectionsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data: authConfig } = usePublicAuthConfig()
  const { data: connections, isLoading } = useMeConnections()
  const deleteConnection = useDeleteConnection()

  function unlink(provider: string, label: string) {
    deleteConnection.mutate(provider, {
      onSuccess: () => toast({ variant: 'success', message: t('account.oauth_unlinked', { provider: label }) }),
      onError: (err) => toast({ variant: 'error', message: err.message }),
    })
  }

  const enabledProviders = OAUTH_PROVIDERS.filter((p) => authConfig?.oauth?.[p.id]?.enabled)

  return (
    <div className="space-y-6">
      <SettingsSectionCard
        title={t('account.connections_title')}
        description={t('account.connections_desc')}
        badge={<LiveBadge />}
      >
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {enabledProviders.map((provider) => {
            const binding = connections?.[provider.id]
            const linked = binding?.linked ?? false
            return (
              <div
                key={provider.id}
                className="flex items-center justify-between gap-3 rounded-md border border-border p-4"
              >
                <div className="flex min-w-0 items-center gap-3">
                  <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-bg-tertiary">
                    <OAuthProviderIcon provider={provider.id} size={22} />
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{t(provider.labelKey)}</span>
                      {linked && (
                        <Badge variant="success" className="text-[10px]">
                          {t('account.status_on')}
                        </Badge>
                      )}
                    </div>
                    {linked && binding?.label && (
                      <p className="truncate text-xs text-text-tertiary">
                        {t('account.oauth_linked', { value: binding.label })}
                      </p>
                    )}
                  </div>
                </div>
                {linked ? (
                  <Button
                    size="sm"
                    variant="ghost"
                    className="shrink-0"
                    loading={deleteConnection.isPending}
                    onClick={() => unlink(provider.id, t(provider.labelKey))}
                  >
                    {t('account.oauth_unlink')}
                  </Button>
                ) : (
                  <span className="text-xs text-text-tertiary shrink-0">{t('account.oauth_not_linked')}</span>
                )}
              </div>
            )
          })}
        </div>

        {isLoading && (
          <p className="text-xs text-text-tertiary">{t('common.loading')}</p>
        )}

        {enabledProviders.length > 0 && (
          <div className="space-y-2 pt-2">
            <p className="text-xs text-text-tertiary">{t('account.connections_link_hint')}</p>
            <OAuthButtons config={authConfig} mode="bind" />
          </div>
        )}

        <p className="text-xs text-text-tertiary">{t('account.connections_hint')}</p>
      </SettingsSectionCard>
    </div>
  )
}